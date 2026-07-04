package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/pqap/services/position-manager/adapters"
	"github.com/pqap/services/position-manager/config"
	"github.com/pqap/services/position-manager/internal/ports"
	"github.com/pqap/services/position-manager/internal/tracker"
	"github.com/pqap/services/position-manager/metrics"
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

	log.Info("starting position-manager service",
		zap.Duration("reconciliation_interval", cfg.ReconciliationInterval),
		zap.Int("mismatch_threshold", cfg.ReconciliationMismatchThreshold),
		zap.Float64("market_limit_pct", cfg.MarketLimitPct),
		zap.Float64("strategy_limit_pct", cfg.StrategyLimitPct),
		zap.String("nats_url", config.RedactCredentials(cfg.NATSURL)),
		zap.String("postgres_url", config.RedactCredentials(cfg.PostgresURL)),
		zap.String("polymarket_api_url", cfg.PolymarketAPIURL),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	postgresRepo, err := adapters.NewPostgresRepo(ctx, cfg.PostgresURL, log)
	if err != nil {
		log.Fatal("failed to connect to PostgreSQL", zap.Error(err))
	}
	defer postgresRepo.Close()

	natsPublisher, err := adapters.NewNATSPublisher(cfg.NATSURL, log, metrics.NATSConnectionStatus)
	if err != nil {
		log.Fatal("failed to connect to NATS publisher", zap.Error(err))
	}

	polymarket := adapters.NewPolymarketAccount(cfg.PolymarketAPIURL, log)

	tr := tracker.NewTracker(postgresRepo, natsPublisher, log)
	tr.InitializeActiveCount(ctx)

	reconciler := tracker.NewReconciler(
		postgresRepo, polymarket, natsPublisher,
		cfg.ReconciliationInterval, cfg.ReconciliationMismatchThreshold, log,
	)
	resolution := tracker.NewResolutionDetector(postgresRepo, natsPublisher, log)
	exitHandler := tracker.NewExitHandler(postgresRepo, natsPublisher, log)
	limitAlert := tracker.NewLimitAlert(
		postgresRepo, natsPublisher,
		cfg.MarketLimitPct, cfg.StrategyLimitPct,
		decimal.NewFromFloat(cfg.TotalCapital),
		log,
	)

	natsSubscriber, err := adapters.NewNATSSubscriber(cfg.NATSURL, log, metrics.NATSConnectionStatus)
	if err != nil {
		log.Fatal("failed to connect to NATS subscriber", zap.Error(err))
	}

	if err := natsSubscriber.SubscribeOrderFilled(ctx, func(event ports.OrderFilled) {
		requestID := uuid.New().String()
		reqLog := log.With(zap.String("request_id", requestID))
		if err := tr.HandleOrderFilled(ctx, event); err != nil {
			reqLog.Error("failed to handle order filled", zap.Error(err))
		}
	}); err != nil {
		log.Fatal("failed to subscribe to order filled", zap.Error(err))
	}

	if err := natsSubscriber.SubscribeMarketPriceUpdated(ctx, func(marketID string, event ports.MarketPriceUpdated) {
		requestID := uuid.New().String()
		reqLog := log.With(zap.String("request_id", requestID))
		if err := tr.HandlePriceUpdate(ctx, marketID, event); err != nil {
			reqLog.Error("failed to handle price update", zap.Error(err))
		}
		limitAlert.CheckLimits(ctx, marketID)
	}); err != nil {
		log.Fatal("failed to subscribe to price updates", zap.Error(err))
	}

	if err := natsSubscriber.SubscribeMarketResolved(ctx, func(event ports.MarketResolved) {
		requestID := uuid.New().String()
		reqLog := log.With(zap.String("request_id", requestID))
		if err := resolution.HandleMarketResolved(ctx, event); err != nil {
			reqLog.Error("failed to handle market resolved", zap.Error(err))
		}
	}); err != nil {
		log.Fatal("failed to subscribe to market resolved", zap.Error(err))
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		reconciler.Start(ctx)
	}()

	metricsServer := startMetricsServer(cfg.MetricsBindAddress, cfg.MetricsPort, cfg.JWTSecret, exitHandler, reconciler, tr, postgresRepo, natsPublisher, log)

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := metricsServer.Shutdown(shutdownCtx); err != nil {
			log.Error("metrics server shutdown error", zap.Error(err))
		}
	}()

	log.Info("position-manager service started")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Info("shutting down position-manager service")
	cancel()

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

func writeJSONError(w http.ResponseWriter, statusCode int, errMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": errMsg})
}

func jwtMiddleware(jwtSecret string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if jwtSecret == "" {
			next(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeJSONError(w, http.StatusUnauthorized, "missing Authorization header")
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeJSONError(w, http.StatusUnauthorized, "invalid Authorization header format")
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			writeJSONError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		next(w, r)
	}
}

func startMetricsServer(
	bindAddr, port, jwtSecret string,
	exitHandler *tracker.ExitHandler,
	reconciler *tracker.Reconciler,
	tr *tracker.Tracker,
	repo *adapters.PostgresRepo,
	natsPublisher *adapters.NATSPublisher,
	log *zap.Logger,
) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		checks := map[string]bool{
			"nats":     natsPublisher.IsConnected(),
			"postgres": repo.Ping(r.Context()) == nil,
		}
		allOk := true
		for _, ok := range checks {
			if !ok {
				allOk = false
				break
			}
		}
		status := "ok"
		code := http.StatusOK
		if !allOk {
			status = "degraded"
			code = http.StatusServiceUnavailable
		}
		w.WriteHeader(code)
		fmt.Fprintf(w, `{"status":"%s","checks":{"nats":%v,"postgres":%v}}`, status, checks["nats"], checks["postgres"])
	})

	mux.HandleFunc("/api/v1/positions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		positions, err := tr.GetOpenPositions(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(positions)
	})

	mux.HandleFunc("/api/v1/positions/history", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		history, err := tr.GetHistory(r.Context(), 100, 0)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(history)
	})

	mux.HandleFunc("/api/v1/positions/reconciliation/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		state, err := reconciler.GetState()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(state)
	})

	exitHandlerFn := jwtMiddleware(jwtSecret, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Path[len("/api/v1/positions/"):]
		if len(id) > 6 && id[len(id)-6:] == "/exit" {
			positionID := id[:len(id)-6]
			if err := exitHandler.RequestExit(r.Context(), positionID); err != nil {
				if errors.Is(err, ports.ErrPositionNotFound) {
					writeJSONError(w, http.StatusNotFound, "position not found")
					return
				}
				writeJSONError(w, http.StatusBadRequest, err.Error())
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(map[string]string{"status": "exit_requested"})
			return
		}
		writeJSONError(w, http.StatusNotFound, "not found")
	})

	getPositionHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Path[len("/api/v1/positions/"):]
		if id == "" {
			return
		}
		pos, err := tr.GetPosition(r.Context(), id)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "position not found")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(pos)
	}

	mux.HandleFunc("/api/v1/positions/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			exitHandlerFn(w, r)
			return
		}
		getPositionHandler(w, r)
	})

	addr := bindAddr + ":" + port
	log.Info("starting metrics server", zap.String("addr", addr))

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		MaxHeaderBytes:    1 << 20,
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
