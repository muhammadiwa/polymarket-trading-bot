package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/pqap/services/risk-manager/adapters"
	"github.com/pqap/services/risk-manager/config"
	"github.com/pqap/services/risk-manager/internal/emergency"
	"github.com/pqap/services/risk-manager/internal/pitboss"
	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/pqap/services/risk-manager/metrics"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	cfg := config.Load()

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "invalid configuration: %v\n", err)
		os.Exit(1)
	}

	log := initLogger(cfg.LogLevel)
	defer log.Sync()

	log.Info("starting risk-manager service",
		zap.Float64("daily_loss_limit_pct", cfg.DailyLossLimitPct),
		zap.Float64("market_limit_pct", cfg.MarketLimitPct),
		zap.Float64("strategy_limit_pct", cfg.StrategyLimitPct),
		zap.Duration("state_refresh_interval", cfg.StateRefreshInterval),
		zap.Duration("state_ttl", cfg.StateTTL),
		zap.String("capital_total", cfg.CapitalTotal), // #21: string
		zap.Float64("drawdown_limit_pct", cfg.DrawdownLimitPct),
		zap.Float64("drawdown_warning_threshold", cfg.DrawdownWarningThreshold),
		zap.Int("api_timeout_minutes", cfg.APITimeoutMinutes),
		zap.Int("recon_mismatch_limit", cfg.ReconMismatchLimit),
		zap.Duration("order_cancel_timeout", cfg.OrderCancelTimeout),
		zap.String("nats_url", config.RedactCredentials(cfg.NATSURL)),
		zap.String("redis_url", config.RedactCredentials(cfg.RedisURL)),
		zap.String("postgres_url", config.RedactCredentials(cfg.PostgresURL)),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// #1: Fail startup if JWTSecret is empty
	if cfg.JWTSecret == "" {
		log.Fatal("RISK_JWT_SECRET must be set for API authentication")
	}

	// #21: Parse capital as decimal to avoid float64 precision loss
	capital, err := decimal.NewFromString(cfg.CapitalTotal)
	if err != nil {
		log.Fatal("invalid capital total", zap.String("value", cfg.CapitalTotal), zap.Error(err))
	}

	postgresRepo, err := adapters.NewPostgresRepo(cfg.PostgresURL, log)
	if err != nil {
		log.Fatal("failed to connect to PostgreSQL", zap.Error(err))
	}
	defer postgresRepo.Close()

	redisWriter, err := adapters.NewRedisWriter(cfg.RedisURL, log)
	if err != nil {
		log.Fatal("failed to connect to Redis", zap.Error(err))
	}
	defer redisWriter.Close()

	natsPublisher, err := adapters.NewNATSPublisher(cfg.NATSURL, log, metrics.NATSConnectionStatus)
	if err != nil {
		log.Fatal("failed to connect to NATS publisher", zap.Error(err))
	}

	dailyBudget := pitboss.NewDailyBudget(capital, cfg.DailyLossLimitPct, cfg.DailyBudgetWarningThreshold)
	marketLimits := pitboss.NewMarketLimit(capital, cfg.MarketLimitPct)
	strategyLimits := pitboss.NewStrategyLimit(capital, cfg.StrategyLimitPct)
	stateBuilder := pitboss.NewStateBuilder(dailyBudget, marketLimits, strategyLimits, capital, log)
	riskLogger := pitboss.NewLogger(postgresRepo, log)

	drawdownTracker := pitboss.NewDrawdownTracker(
		capital,
		cfg.DrawdownLimitPct,
		cfg.DrawdownWarningThreshold,
		natsPublisher,
		log,
	)

	pb := pitboss.NewPitBoss(
		dailyBudget,
		marketLimits,
		strategyLimits,
		stateBuilder,
		riskLogger,
		natsPublisher,
		capital,
		drawdownTracker,
		log,
	)

	reconstructor := pitboss.NewReconstructor(
		postgresRepo,
		dailyBudget,
		marketLimits,
		strategyLimits,
		stateBuilder,
		drawdownTracker,
		log,
	)

	if err := reconstructor.Reconstruct(ctx); err != nil {
		log.Error("state reconstruction failed", zap.Error(err))
	}

	emergencyStop := emergency.NewEmergencyStop(natsPublisher, log)
	orderCanceler := emergency.NewOrderCanceler(natsPublisher, cfg.OrderCancelTimeout, log)
	emergencyStop.SetOrderCanceler(orderCanceler)
	emergencyStop.SetCallbacks(
		func() {
			pb.SetEmergencyStop(true)
			// #4: Write state to Redis immediately after emergency activation
			state := stateBuilder.BuildState()
			if err := redisWriter.WriteState(state, cfg.StateTTL, cfg.EmergencyStopTTL); err != nil {
				log.Error("failed to write emergency state to Redis immediately", zap.Error(err))
			}
		},
		func() { pb.SetEmergencyStop(false) },
	)

	// #10: Guard OnBreach handler with sync.Once to prevent goroutine leak
	var breachOnce sync.Once
	drawdownTracker.SetOnBreach(func() {
		breachOnce.Do(func() {
			peak, current, drawdown, _ := drawdownTracker.GetState()
			if err := emergencyStop.ActivateWithDetails(
				ctx,
				"drawdown_exceeded",
				&drawdown,
				&peak,
				&current,
				pb.DailyBudget().DailyLossValue().Neg(),
				0,
				nil,
			); err != nil {
				log.Error("failed to activate emergency stop from drawdown breach", zap.Error(err))
			}
		})
	})

	natsSubscriber, err := adapters.NewNATSSubscriber(cfg.NATSURL, log, metrics.NATSConnectionStatus)
	if err != nil {
		log.Fatal("failed to connect to NATS subscriber", zap.Error(err))
	}

	if err := natsSubscriber.SubscribePositionOpened(ctx, func(event ports.PositionOpened) {
		pb.HandlePositionOpened(event)
	}); err != nil {
		log.Fatal("failed to subscribe to position opened", zap.Error(err))
	}

	// #19: Track warning state to avoid repeated fires
	var warningFired bool
	var warningDate string
	var warningMu sync.Mutex

	if err := natsSubscriber.SubscribePositionClosed(ctx, func(event ports.PositionClosed) {
		pb.HandlePositionClosed(event)

		today := time.Now().UTC().Format("2006-01-02")
		warningMu.Lock()
		if warningDate != today {
			warningFired = false
			warningDate = today
		}
		shouldWarn := !warningFired && dailyBudget.ShouldWarn()
		if shouldWarn {
			warningFired = true
		}
		warningMu.Unlock()

		if shouldWarn {
			warnEvent := ports.DailyBudgetWarning{
				EventID:   uuid.New().String(),
				EventType: "DailyBudgetWarning",
				Timestamp: time.Now().UTC(),
				Source:    "risk-manager",
				Payload: ports.DailyBudgetWarningPayload{
					DailyLoss:       dailyBudget.DailyLossValue(),
					DailyLossLimit:  dailyBudget.DailyLossLimitValue(),
					Utilization:     dailyBudget.Utilization(),
					BudgetRemaining: dailyBudget.BudgetRemaining(),
				},
			}
			if pubErr := natsPublisher.PublishDailyBudgetWarning(ctx, warnEvent); pubErr != nil {
				log.Error("failed to publish daily budget warning", zap.Error(pubErr))
			}
		}
	}); err != nil {
		log.Fatal("failed to subscribe to position closed", zap.Error(err))
	}

	if err := natsSubscriber.SubscribePositionUpdated(ctx, func(event ports.PositionUpdated) {
		pb.HandlePositionUpdated(event)
	}); err != nil {
		log.Fatal("failed to subscribe to position updated", zap.Error(err))
	}

	if err := natsSubscriber.SubscribeCapitalUpdated(ctx, func(event ports.CapitalUpdated) {
		pb.HandleCapitalUpdated(event)
	}); err != nil {
		log.Fatal("failed to subscribe to capital updated", zap.Error(err))
	}

	// #2: Use atomic.Int32 for reconMismatchCount (data race fix)
	var reconMismatchCount atomic.Int32
	if err := natsSubscriber.SubscribeRiskAlert(ctx, func(event ports.RiskAlert) {
		if event.Payload.AlertType == "data_corruption" || event.Payload.Severity == "critical" {
			count := reconMismatchCount.Add(1)
			if int(count) >= cfg.ReconMismatchLimit {
				log.Error("data corruption threshold reached",
					zap.Int32("consecutive_mismatches", count),
				)
				if err := emergencyStop.Activate(ctx, "data_corruption"); err != nil {
					log.Error("failed to activate emergency stop from data corruption", zap.Error(err))
				}
				reconMismatchCount.Store(0)
			}
		} else {
			reconMismatchCount.Store(0)
		}
	}); err != nil {
		log.Fatal("failed to subscribe to risk alert", zap.Error(err))
	}

	// #7, #14: Single JetStream durable subscription for emergency stop (removed duplicate raw NATS)
	if err := natsSubscriber.SubscribeEmergencyStop(ctx, func(event ports.EmergencyStop) {
		if err := emergencyStop.Activate(ctx, event.Payload.Reason); err != nil {
			log.Error("failed to activate emergency stop", zap.Error(err))
		}
	}); err != nil {
		log.Fatal("failed to subscribe to emergency stop", zap.Error(err))
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		runStateRefreshLoop(ctx, pb, stateBuilder, redisWriter, natsPublisher, emergencyStop, cfg, log)
	}()

	apiServer := startAPIServer(cfg, pb, emergencyStop, redisWriter, postgresRepo, natsPublisher, log)

	// #1: Separate metrics server on different port
	metricsServer := startMetricsServer(cfg.MetricsBindAddress, cfg.MetricsPort, log)

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := apiServer.Shutdown(shutdownCtx); err != nil {
			log.Error("api server shutdown error", zap.Error(err))
		}
		if err := metricsServer.Shutdown(shutdownCtx); err != nil {
			log.Error("metrics server shutdown error", zap.Error(err))
		}
	}()

	log.Info("risk-manager service started",
		zap.String("api_addr", cfg.APIBindAddress+":"+cfg.APIPort),
		zap.String("metrics_addr", cfg.MetricsBindAddress+":"+cfg.MetricsPort),
	)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Info("shutting down risk-manager service")
	cancel()
	riskLogger.Close() // #8: drain logger channel
	pb.Close()         // #18: drain publish channel

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("all goroutines stopped gracefully")
	case <-time.After(10 * time.Second):
		log.Warn("shutdown timed out waiting for goroutines")
	}

	natsSubscriber.Close()
	natsPublisher.Close()
}

func runStateRefreshLoop(
	ctx context.Context,
	pb *pitboss.PitBoss,
	sb *pitboss.StateBuilder,
	rw *adapters.RedisWriter,
	pub *adapters.NATSPublisher,
	es *emergency.EmergencyStop,
	cfg *config.Config,
	log *zap.Logger,
) {
	ticker := time.NewTicker(cfg.StateRefreshInterval)
	defer ticker.Stop()

	// #3: Death spiral detection — consecutive write failure counter
	consecutiveWriteFailures := 0
	const maxConsecutiveFailures = 5

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			start := time.Now()

			state := sb.BuildState()

			if err := rw.WriteState(state, cfg.StateTTL, cfg.EmergencyStopTTL); err != nil {
				log.Error("failed to write state to Redis", zap.Error(err))
				consecutiveWriteFailures++
				metrics.StateWriteConsecutiveFailures.Set(float64(consecutiveWriteFailures))
				if consecutiveWriteFailures >= maxConsecutiveFailures {
					log.Error("API death spiral detected - consecutive state write failures",
						zap.Int("failures", consecutiveWriteFailures))
					if activateErr := es.Activate(ctx, "api_death_spiral"); activateErr != nil {
						log.Error("failed to activate emergency stop from death spiral", zap.Error(activateErr))
					}
					consecutiveWriteFailures = 0
				}
				continue
			}
			consecutiveWriteFailures = 0
			metrics.StateWriteConsecutiveFailures.Set(0)

			elapsed := time.Since(start)
			metrics.StateRefreshTotal.Inc()
			metrics.StateRefreshLatency.Observe(float64(elapsed.Milliseconds()))

			if es.IsActive() {
				metrics.EmergencyStopDuration.Set(es.Duration().Seconds())
			} else {
				metrics.EmergencyStopDuration.Set(0)
			}

			metrics.DailyBudgetRemaining.Set(func() float64 { f, _ := state.DailyBudgetRemaining.Float64(); return f }())
			metrics.DailyLoss.Set(func() float64 { f, _ := state.DailyLoss.Float64(); return f }())
			metrics.DailyLossLimit.Set(func() float64 { f, _ := state.DailyLossLimit.Float64(); return f }())

			for marketID, entry := range state.MarketLimits {
				metrics.MarketExposure.WithLabelValues(metrics.SafeLabel(marketID)).Set(func() float64 { f, _ := entry.Exposure.Float64(); return f }())
				metrics.MarketUtilization.WithLabelValues(metrics.SafeLabel(marketID)).Set(entry.Utilization)
			}

			for strategyID, entry := range state.StrategyLimits {
				metrics.StrategyExposure.WithLabelValues(metrics.SafeLabel(strategyID)).Set(func() float64 { f, _ := entry.Exposure.Float64(); return f }())
				metrics.StrategyUtilization.WithLabelValues(metrics.SafeLabel(strategyID)).Set(entry.Utilization)
			}

			event := ports.RiskStateUpdated{
				EventID:   uuid.New().String(),
				EventType: "RiskStateUpdated",
				Timestamp: time.Now().UTC(),
				Source:    "risk-manager",
				Payload: ports.RiskStateUpdatedPayload{
					DailyBudgetRemaining: state.DailyBudgetRemaining,
					DailyLoss:            state.DailyLoss,
					Capital:              state.Capital,
					MarketCount:          len(state.MarketLimits),
					StrategyCount:        len(state.StrategyLimits),
					EmergencyStop:        state.EmergencyStop,
				},
			}

			if err := pub.PublishRiskStateUpdated(ctx, event); err != nil {
				log.Error("failed to publish risk state updated", zap.Error(err))
			}

			log.Debug("state refresh completed",
				zap.Duration("elapsed", elapsed),
				zap.String("daily_budget_remaining", state.DailyBudgetRemaining.String()),
				zap.Int("market_count", len(state.MarketLimits)),
				zap.Int("strategy_count", len(state.StrategyLimits)),
			)
		}
	}
}

// #6: JWT authentication middleware
func jwtAuthMiddleware(secret string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// #1: secret is validated at startup; no bypass needed

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func startAPIServer(cfg *config.Config, pb *pitboss.PitBoss, es *emergency.EmergencyStop, rw *adapters.RedisWriter, repo *adapters.PostgresRepo, pub *adapters.NATSPublisher, log *zap.Logger) *http.Server {
	mux := http.NewServeMux()

	// #14: JWT auth on all sensitive endpoints, #27: method checks on all endpoints

	mux.HandleFunc("/api/v1/risk/state", jwtAuthMiddleware(cfg.JWTSecret, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		state, err := rw.ReadState()
		if err != nil {
			http.Error(w, `{"error":"state not available"}`, http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(state)
	}))

	mux.HandleFunc("/api/v1/risk/daily-budget", jwtAuthMiddleware(cfg.JWTSecret, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		resp := map[string]interface{}{
			"daily_budget_remaining": pb.DailyBudget().BudgetRemaining().String(),
			"daily_loss":             pb.DailyBudget().DailyLossValue().String(),
			"daily_loss_limit":       pb.DailyBudget().DailyLossLimitValue().String(),
			"utilization":            pb.DailyBudget().Utilization(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))

	mux.HandleFunc("/api/v1/risk/limits", jwtAuthMiddleware(cfg.JWTSecret, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		resp := map[string]interface{}{
			"market_limits":   pb.MarketLimits().GetAllExposures(),
			"strategy_limits": pb.StrategyLimits().GetAllExposures(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))

	mux.HandleFunc("/api/v1/risk/emergency-stop", jwtAuthMiddleware(cfg.JWTSecret, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		if err := es.Activate(r.Context(), "manual"); err != nil { // #12: fixed reason
			http.Error(w, `{"error":"failed to activate"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "emergency_stop_activated"})
	}))

	mux.HandleFunc("/api/v1/risk/resume", jwtAuthMiddleware(cfg.JWTSecret, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		// #28: Check IsActive before resume
		if !es.IsActive() {
			http.Error(w, `{"error":"emergency stop is not active"}`, http.StatusBadRequest)
			return
		}
		if err := es.Resume(r.Context()); err != nil {
			http.Error(w, `{"error":"failed to resume"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "resumed"})
	}))

	mux.HandleFunc("/api/v1/risk/events", jwtAuthMiddleware(cfg.JWTSecret, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		decisions, err := repo.GetTodayDecisions(r.Context())
		if err != nil {
			http.Error(w, `{"error":"failed to fetch events"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(decisions)
	}))

	mux.HandleFunc("/api/v1/risk/config", jwtAuthMiddleware(cfg.JWTSecret, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		resp := map[string]interface{}{
			"daily_loss_limit_pct":            cfg.DailyLossLimitPct,
			"market_limit_pct":                cfg.MarketLimitPct,
			"strategy_limit_pct":              cfg.StrategyLimitPct,
			"state_refresh_interval":          cfg.StateRefreshInterval.String(),
			"state_ttl":                       cfg.StateTTL.String(),
			"daily_budget_warning_threshold":  cfg.DailyBudgetWarningThreshold,
			"capital_total":                   cfg.CapitalTotal,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))

	mux.HandleFunc("/api/v1/risk/emergency-status", jwtAuthMiddleware(cfg.JWTSecret, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		resp := map[string]interface{}{
			"active":        es.IsActive(),
			"reason":        es.Reason(),
			"activated_at":  es.ActivatedAt(),
			"duration_secs": es.Duration().Seconds(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","emergency_stop":%v}`, es.IsActive())
	})

	// #1: Use separate API port
	addr := cfg.APIBindAddress + ":" + cfg.APIPort
	log.Info("starting API server", zap.String("addr", addr))

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("api server error", zap.Error(err))
		}
	}()

	return server
}

func startMetricsServer(bindAddr, port string, log *zap.Logger) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	addr := bindAddr + ":" + port
	log.Info("starting metrics server", zap.String("addr", addr))

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("metrics server error", zap.Error(err))
		}
	}()

	return server
}

func initLogger(level string) *zap.Logger {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	zapConfig := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapLevel),
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := zapConfig.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to init logger: %v", err))
	}

	return logger
}
