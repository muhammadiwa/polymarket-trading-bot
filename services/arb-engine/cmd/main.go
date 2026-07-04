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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/pqap/services/arb-engine/adapters"
	"github.com/pqap/services/arb-engine/config"
	"github.com/pqap/services/arb-engine/internal/detector"
	"github.com/pqap/services/arb-engine/internal/filter"
	"github.com/pqap/services/arb-engine/internal/logger"
	"github.com/pqap/services/arb-engine/internal/ports"
	"github.com/pqap/services/arb-engine/internal/scorer"
	"github.com/pqap/services/arb-engine/metrics"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type deduplicator struct {
	mu      sync.Mutex
	seen    map[string]struct{}
	maxSize int
}

func newDeduplicator(maxSize int) *deduplicator {
	return &deduplicator{
		seen:    make(map[string]struct{}),
		maxSize: maxSize,
	}
}

func (d *deduplicator) isDuplicate(id string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.seen[id]; exists {
		return true
	}
	if len(d.seen) >= d.maxSize {
		d.seen = make(map[string]struct{})
	}
	d.seen[id] = struct{}{}
	return false
}

func main() {
	cfg := config.Load()

	log := initLogger(cfg.LogLevel)
	defer log.Sync()

	log.Info("starting arb-engine service",
		zap.String("min_profit_threshold", cfg.MinProfitThreshold),
		zap.String("score_threshold", cfg.ScoreThreshold),
		zap.String("nats_url", config.RedactCredentials(cfg.NATSURL)),
		zap.String("timescale_url", config.RedactCredentials(cfg.TimescaleURL)),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	timescaleRepo, err := adapters.NewTimescaleRepo(cfg.TimescaleURL, log)
	if err != nil {
		log.Fatal("failed to connect to TimescaleDB", zap.Error(err))
	}
	defer timescaleRepo.Close()

	oppLogger := logger.NewOpportunityLogger(timescaleRepo, log)
	defer oppLogger.Close()

	fillProbEstimator := scorer.NewFillProbabilityEstimator(
		oppLogger,
		cfg.FillProbWeightOrderbook,
		cfg.FillProbWeightHistorical,
		cfg.FillProbRequiredDepth,
	)

	scorerEngine := scorer.NewScorer(fillProbEstimator, cfg.LiquidityMaxDepth)

	arbDetector := detector.NewSimpleArbDetector(cfg.MinProfitThreshold)

	thresholdFilter := filter.NewThresholdFilter(cfg.ScoreThreshold)

	natsPublisher, err := adapters.NewNATSPublisher(cfg.NATSURL, log)
	if err != nil {
		log.Fatal("failed to connect to NATS publisher", zap.Error(err))
	}

	natsSubscriber, err := adapters.NewNATSSubscriber(cfg.NATSURL, log, metrics.NATSConnectionStatus, 4)
	if err != nil {
		log.Fatal("failed to connect to NATS subscriber", zap.Error(err))
	}

	dedup := newDeduplicator(10000)

	natsSubscriber.StartWorkers(ctx, func(event ports.MarketPriceUpdated) {
		if dedup.isDuplicate(event.MarketID + event.Timestamp.String()) {
			return
		}
		processMarketEvent(ctx, event, arbDetector, scorerEngine, thresholdFilter, natsPublisher, oppLogger, log)
	}, 4)

	err = natsSubscriber.Subscribe(ctx, func(event ports.MarketPriceUpdated) {
		if dedup.isDuplicate(event.MarketID + event.Timestamp.String()) {
			return
		}
		processMarketEvent(ctx, event, arbDetector, scorerEngine, thresholdFilter, natsPublisher, oppLogger, log)
	})
	if err != nil {
		log.Fatal("failed to subscribe to NATS", zap.Error(err))
	}

	metricsServer := startMetricsServer(cfg.MetricsBindAddress, cfg.MetricsPort, log, natsPublisher, timescaleRepo)

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

	log.Info("arb-engine service started")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Info("shutting down arb-engine service")
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

func processMarketEvent(
	ctx context.Context,
	event ports.MarketPriceUpdated,
	detector *detector.SimpleArbDetector,
	scorerEngine *scorer.Scorer,
	thresholdFilter *filter.ThresholdFilter,
	publisher *adapters.NATSPublisher,
	oppLogger *logger.OpportunityLogger,
	log *zap.Logger,
) {
	if event.IsStale {
		metrics.StaleMarketIgnored.Inc()
		log.Debug("ignoring stale market", zap.String("market_id", event.MarketID))
		return
	}

	opp, latencyMs := detector.Detect(event)
	if opp == nil {
		return
	}

	metrics.OpportunitiesDetected.Inc()
	metrics.DetectionLatency.Observe(float64(latencyMs))

	if latencyMs > 100 {
		log.Warn("detection latency exceeded 100ms",
			zap.String("market_id", event.MarketID),
			zap.Int64("latency_ms", latencyMs),
		)
	}

	scorerEngine.Score(ctx, opp, event.LiquidityDepth, event.MarketID)

	metrics.ScoreDistribution.Observe(opp.Score.InexactFloat64())
	metrics.FillProbabilityEstimate.Observe(opp.FillProbability.InexactFloat64())

	emitted := thresholdFilter.Filter(opp)

	if emitted {
		metrics.OpportunitiesEmitted.Inc()

		oppEvent := ports.OpportunityDetected{
			EventID:   uuid.New().String(),
			EventType: "OpportunityDetected",
			Timestamp: time.Now().UTC(),
			Source:    "arb-engine",
			Payload: ports.OpportunityPayload{
				OpportunityID:  opp.ID,
				MarketID:       opp.MarketID,
				YESPrice:       opp.YESPrice,
				NOPrice:        opp.NOPrice,
				Spread:         opp.Spread,
				Score:          opp.Score,
				FillProbability: opp.FillProbability,
				Liquidity:      opp.Liquidity,
			},
		}

		if err := publisher.PublishOpportunityDetected(ctx, oppEvent); err != nil {
			log.Error("failed to publish opportunity",
				zap.String("opportunity_id", opp.ID),
				zap.Error(err),
			)
		}
	} else {
		metrics.OpportunitiesFiltered.Inc()
	}

	if err := oppLogger.Log(ctx, *opp); err != nil {
		log.Error("failed to log opportunity",
			zap.String("opportunity_id", opp.ID),
			zap.Error(err),
		)
	}
}

func startMetricsServer(bindAddr, port string, log *zap.Logger, natsPublisher *adapters.NATSPublisher, timescaleRepo *adapters.TimescaleRepo) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		checks := map[string]bool{
			"nats":      natsPublisher.IsConnected(),
			"timescaledb": timescaleRepo.Ping(r.Context()) == nil,
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
		fmt.Fprintf(w, `{"status":"%s","checks":{"nats":%v,"timescaledb":%v}}`, status, checks["nats"], checks["timescaledb"])
	})

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
