package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/pqap/services/execution-engine/adapters"
	"github.com/pqap/services/execution-engine/config"
	circuitbreaker "github.com/pqap/services/execution-engine/internal/circuit_breaker"
	"github.com/pqap/services/execution-engine/internal/executor"
	"github.com/pqap/services/execution-engine/internal/history"
	"github.com/pqap/services/execution-engine/internal/logger"
	"github.com/pqap/services/execution-engine/internal/monitor"
	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/pqap/services/execution-engine/metrics"
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

	log.Info("starting execution-engine service",
		zap.Float64("slippage_tolerance", cfg.SlippageTolerance),
		zap.String("time_in_force", cfg.TimeInForce),
		zap.Int("circuit_breaker_threshold", cfg.CircuitBreakerThreshold),
		zap.Duration("atomic_timeout", cfg.AtomicTimeout),
		zap.Duration("leg_cancel_timeout", cfg.LegCancelTimeout),
		zap.String("nats_url", config.RedactCredentials(cfg.NATSURL)),
		zap.String("redis_url", config.RedactCredentials(cfg.RedisURL)),
		zap.String("postgres_url", config.RedactCredentials(cfg.PostgresURL)),
		zap.Int("max_concurrency", cfg.MaxConcurrency),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	postgresRepo, err := adapters.NewPostgresRepo(cfg.PostgresURL, log)
	if err != nil {
		log.Fatal("failed to connect to PostgreSQL", zap.Error(err))
	}
	defer postgresRepo.Close()

	orderLogger := logger.NewOrderLogger(postgresRepo, log)
	defer orderLogger.Close()

	atomicLogger := logger.NewAtomicLogger(postgresRepo, log)
	defer atomicLogger.Close()

	breakerLogger := logger.NewBreakerLogger(postgresRepo, log)
	defer breakerLogger.Close()

	redisRisk, err := adapters.NewRedisRisk(cfg.RedisURL, log, cfg.MarketPositionLimit, cfg.StrategyPositionLimit)
	if err != nil {
		log.Fatal("failed to connect to Redis", zap.Error(err))
	}
	defer redisRisk.Close()

	marketPricePort, err := adapters.NewRedisMarketPrice(cfg.RedisURL, log)
	if err != nil {
		log.Fatal("failed to connect to Redis market price", zap.Error(err))
	}
	defer marketPricePort.Close()

	clobAdapter := adapters.NewPolymarketCLOB(cfg.PolymarketCLOBURL, log)

	natsPublisher, err := adapters.NewNATSPublisher(cfg.NATSURL, log, metrics.NATSPublisherConnectionStatus)
	if err != nil {
		log.Fatal("failed to connect to NATS publisher", zap.Error(err))
	}

	tradeHistoryRepo := history.NewPostgresRepository(postgresRepo.Pool(), log)
	tradeHistoryHandler := history.NewHandler(tradeHistoryRepo, natsPublisher, log)

	cb, err := circuitbreaker.NewCircuitBreaker(
		cfg.CircuitBreakerThreshold,
		cfg.CircuitBreakerCooldown,
		cfg.CircuitBreakerProbeTimeout,
		log,
		natsPublisher,
		func(lastError string, consecutiveErrors int) {
			log.Error("circuit breaker tripped - halting all trading",
				zap.String("last_error", lastError),
				zap.Int("consecutive_errors", consecutiveErrors),
			)
			if logErr := breakerLogger.LogTrip(ctx, consecutiveErrors, lastError, int(cfg.CircuitBreakerCooldown.Seconds())); logErr != nil {
				log.Error("failed to log circuit breaker trip", zap.Error(logErr))
			}
		},
	)
	if err != nil {
		log.Fatal("failed to create circuit breaker", zap.Error(err))
	}

	wrappedCLOB := &WrappedCLOB{
		clob: clobAdapter,
		cb:   cb,
	}

	fillMon := monitor.NewFillMonitor(
		wrappedCLOB,
		natsPublisher,
		cfg.FillPollInterval,
		cfg.FillPollTimeout,
		log,
	)

	fillMonAdapter := &FillMonitorAdapter{
		fillMon: fillMon,
		wg:      &wg,
	}

	exec := executor.NewExecutor(
		wrappedCLOB,
		redisRisk,
		natsPublisher,
		marketPricePort,
		cfg.SlippageTolerance,
		cfg.TimeInForce,
		cfg.MaxRetries,
		cfg.RetryBackoffInitial,
		cfg.RetryBackoffMax,
		cfg.MaxConcurrency,
		orderLogger,
		fillMonAdapter,
		postgresRepo,
		tradeHistoryHandler,
		log,
	)

	atomicExec := executor.NewAtomicExecutor(
		wrappedCLOB,
		redisRisk,
		natsPublisher,
		postgresRepo,
		marketPricePort,
		cfg.SlippageTolerance,
		cfg.TimeInForce,
		cfg.AtomicTimeout,
		cfg.LegCancelTimeout,
		atomicLogger,
		log,
	)

	natsSubscriber, err := adapters.NewNATSSubscriber(cfg.NATSURL, log, metrics.NATSConnectionStatus)
	if err != nil {
		log.Fatal("failed to connect to NATS subscriber", zap.Error(err))
	}

	natsConcurrencySem := make(chan struct{}, cfg.MaxConcurrency)

	err = natsSubscriber.Subscribe(ctx, func(event ports.OpportunityDetected) {
		wg.Add(1)
		go func() {
			defer wg.Done()

			natsConcurrencySem <- struct{}{}
			defer func() { <-natsConcurrencySem }()

			isAtomic := event.Payload.YESPrice.IsPositive() && event.Payload.NOPrice.IsPositive()
			if isAtomic {
				if err := atomicExec.ExecuteAtomic(ctx, event); err != nil {
					log.Error("atomic execution failed",
						zap.String("opportunity_id", event.Payload.OpportunityID),
						zap.Error(err),
					)
				}
			} else {
				if err := exec.Execute(ctx, event); err != nil {
					log.Error("execution failed",
						zap.String("opportunity_id", event.Payload.OpportunityID),
						zap.Error(err),
					)
				}
			}
		}()
	})
	if err != nil {
		log.Fatal("failed to subscribe to NATS", zap.Error(err))
	}

	subscribeEmergencyStop(ctx, natsPublisher, cb, log, &wg)

	// Subscribe to exit order requests from position-manager
	err = natsSubscriber.SubscribeExitOrderRequest(ctx, func(event ports.ExitOrderRequest) {
		log.Info("received exit order request",
			zap.String("position_id", event.Payload.PositionID),
			zap.String("market_id", event.Payload.MarketID),
			zap.String("reason", event.Payload.Reason),
		)
		// TODO: Implement exit order logic
		// 1. Find open orders for the position
		// 2. Cancel them or place exit order
		// 3. Publish OrderCancelled or OrderFilled event
	})
	if err != nil {
		log.Error("failed to subscribe to exit order requests", zap.Error(err))
	}

	// Subscribe to cancel-all-orders from risk-manager (emergency stop)
	err = natsSubscriber.SubscribeCancelAllOrders(ctx, func(event ports.CancelAllOrders) {
		log.Warn("received cancel all orders request",
			zap.String("reason", event.Payload.Reason),
			zap.String("user_id", event.Payload.UserID),
		)
		// TODO: Implement cancel-all logic
		// 1. Get all open orders from CLOB
		// 2. Cancel each one
		// 3. Publish OrderCancelled events
	})
	if err != nil {
		log.Error("failed to subscribe to cancel all orders", zap.Error(err))
	}

	resumeHandler := circuitbreaker.NewResumeHandler(cb, cfg.JWTSecret, log)

	metricsServer := startMetricsServer(cfg.MetricsBindAddress, cfg.MetricsPort, resumeHandler, log, natsPublisher, postgresRepo, redisRisk)

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

	log.Info("execution-engine service started")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Info("shutting down execution-engine service")
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

type FillMonitorAdapter struct {
	fillMon *monitor.FillMonitor
	wg      *sync.WaitGroup
}

func (a *FillMonitorAdapter) StartMonitoring(ctx context.Context, order *ports.Order) {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.fillMon.MonitorOrder(ctx, order)
	}()
}

func subscribeEmergencyStop(ctx context.Context, natsPublisher *adapters.NATSPublisher, cb *circuitbreaker.CircuitBreaker, log *zap.Logger, wg *sync.WaitGroup) {
	err := natsPublisher.SubscribeJetStream("pqap.risk.emergency", "execution-engine-emergency", func(msg *nats.Msg) {
		log.Error("EMERGENCY STOP RECEIVED - halting all trading")

		cb.Halt()

		alertEvent := ports.RiskAlert{
			EventID:   uuid.New().String(),
			EventType: "RiskAlert",
			Timestamp: time.Now().UTC(),
			Source:    "execution-engine",
			Payload: ports.RiskAlertPayload{
				AlertType: "emergency_stop",
				Message:   "Emergency stop triggered - all trading halted",
				Severity:  "critical",
			},
		}
		if pubErr := natsPublisher.PublishRiskAlert(ctx, alertEvent); pubErr != nil {
			log.Error("failed to publish emergency stop alert", zap.Error(pubErr))
		}
		msg.Ack()
	})
	if err != nil {
		log.Error("failed to subscribe to emergency stop", zap.Error(err))
		return
	}

	log.Info("subscribed to pqap.risk.emergency (JetStream)")
}

type WrappedCLOB struct {
	clob *adapters.PolymarketCLOB
	cb   *circuitbreaker.CircuitBreaker
}

func (w *WrappedCLOB) PlaceOrder(req ports.OrderRequest, clientOrderID string) (*ports.OrderResponse, error) {
	var resp *ports.OrderResponse
	err := w.cb.Execute(func() error {
		var err error
		resp, err = w.clob.PlaceOrder(req, clientOrderID)
		return err
	})
	return resp, err
}

func (w *WrappedCLOB) CancelOrder(ctx context.Context, orderID string) error {
	return w.cb.Execute(func() error {
		return w.clob.CancelOrder(ctx, orderID)
	})
}

func (w *WrappedCLOB) GetOrderStatus(orderID string) (*ports.OrderStatusResponse, error) {
	var resp *ports.OrderStatusResponse
	err := w.cb.Execute(func() error {
		var err error
		resp, err = w.clob.GetOrderStatus(orderID)
		return err
	})
	return resp, err
}

func startMetricsServer(bindAddr, port string, resumeHandler *circuitbreaker.ResumeHandler, log *zap.Logger, natsPublisher *adapters.NATSPublisher, postgresRepo *adapters.PostgresRepo, redisRisk *adapters.RedisRisk) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		checks := map[string]bool{
			"nats":     natsPublisher.IsConnected(),
			"postgres": postgresRepo.Ping(r.Context()) == nil,
			"redis":    redisRisk.Ping(r.Context()) == nil,
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
		fmt.Fprintf(w, `{"status":"%s","checks":{"nats":%v,"postgres":%v,"redis":%v}}`, status, checks["nats"], checks["postgres"], checks["redis"])
	})
	mux.HandleFunc("/api/v1/execution/circuit-breaker/resume", resumeHandler.HandleResume)
	mux.HandleFunc("/api/v1/execution/circuit-breaker/status", resumeHandler.HandleStatus)

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
