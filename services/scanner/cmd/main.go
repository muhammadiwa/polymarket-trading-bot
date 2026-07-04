package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/pqap/services/scanner/adapters"
	"github.com/pqap/services/scanner/config"
	"github.com/pqap/services/scanner/internal/catalog"
	"github.com/pqap/services/scanner/internal/rest"
	"github.com/pqap/services/scanner/internal/websocket"
	"github.com/pqap/services/scanner/metrics"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	cfg := config.Load()

	logger := initLogger(cfg.LogLevel)
	defer logger.Sync()

	logger.Info("starting scanner service",
		zap.String("ws_url", cfg.WSURL),
		zap.String("rest_url", cfg.RESTURL),
		zap.Duration("stale_threshold", cfg.StaleThreshold),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	natsPublisher, err := adapters.NewNATSPublisher(cfg.NATSURL, logger)
	if err != nil {
		logger.Fatal("failed to connect to NATS", zap.Error(err))
	}
	defer natsPublisher.Close()

	redisCache, err := adapters.NewRedisCache(cfg.RedisURL, cfg.RedisTTL, logger)
	if err != nil {
		logger.Fatal("failed to connect to Redis", zap.Error(err))
	}
	defer redisCache.Close()

	cat := catalog.NewCatalog(func(market catalog.Market) {
		reqID := uuid.New().String()
		if err := natsPublisher.PublishPriceUpdate(ctx, market); err != nil {
			logger.Error("failed to publish price update", zap.Error(err), zap.String("request_id", reqID))
		} else {
			metrics.EventsPublished.WithLabelValues("MarketPriceUpdated").Inc()
		}
		if err := redisCache.SetMarket(ctx, market); err != nil {
			logger.Error("failed to update redis cache", zap.Error(err), zap.String("request_id", reqID))
		}
	})

	restClient := rest.NewClient(cfg.RESTURL, logger)

	wsClient := websocket.NewClient(cfg.WSURL, cfg.ReconnectInitial, cfg.ReconnectMax, logger)

	subscriber := websocket.NewSubscriber(wsClient, logger)

	reconciler := websocket.NewReconciler(cat, restClient, logger)

	var hasConnectedOnce atomic.Bool

	wsClient.SetOnConnect(func() {
		metrics.WSConnectionStatus.Set(1)

		// Fix #15: Only log reconnection, counter now incremented in reconnectLoop
		if hasConnectedOnce.Swap(true) {
			logger.Info("ws_reconnected", zap.String("service", "scanner"))
		}

		markets := cat.List()
		if len(markets) > 0 {
			ids := make([]string, len(markets))
			for i, m := range markets {
				ids[i] = m.ID
			}
			subscriber.Subscribe(ctx, ids)

			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := reconciler.Reconcile(ctx); err != nil {
					logger.Error("reconciliation failed", zap.Error(err))
				}
			}()
		}
	})

	// Fix #14: Reset WSConnectionStatus on disconnect
	wsClient.SetOnDisconnect(func() {
		metrics.WSConnectionStatus.Set(0)
	})

	wsClient.SetOnMessage(func(market catalog.Market) {
		start := time.Now()

		// Fix #6: Use Upsert to eliminate TOCTOU race between Get and Add/Update
		_, wasStale := cat.Upsert(market)
		if wasStale {
			logger.Info("market_recovered",
				zap.String("service", "scanner"),
				zap.String("market_id", market.ID),
			)
			metrics.StaleMarkets.Set(float64(cat.StaleCount()))
		}

		elapsed := time.Since(start)
		metrics.UpdateLatency.Observe(float64(elapsed.Milliseconds()))
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		wsClient.ReadMessages(ctx)
	}()

	staleDetector := catalog.NewStaleDetector(cat, cfg.StaleThreshold, 5*time.Second, logger, func(market catalog.Market) {
		reqID := uuid.New().String()
		if err := natsPublisher.PublishMarketStale(ctx, market); err != nil {
			logger.Error("failed to publish market stale event", zap.Error(err), zap.String("request_id", reqID))
		} else {
			// Fix #23: Count MarketStale events in EventsPublished metric
			metrics.EventsPublished.WithLabelValues("MarketStaleDetected").Inc()
		}
		// Fix #19: Set StaleMarkets gauge to actual stale count from catalog
		metrics.StaleMarkets.Set(float64(cat.StaleCount()))
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		staleDetector.Run(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runMarketDiscovery(ctx, cat, restClient, natsPublisher, redisCache, cfg.MarketPollInterval, logger)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		startMetricsServer(cfg.MetricsBindAddress, cfg.MetricsPort, logger, natsPublisher, redisCache, wsClient)
	}()

	if err := wsClient.Connect(ctx); err != nil {
		logger.Fatal("failed to connect to WebSocket", zap.Error(err))
	}

	initialMarkets, err := restClient.FetchActiveMarkets(ctx)
	if err != nil {
		logger.Error("failed to fetch initial markets", zap.Error(err))
	} else {
		for _, m := range initialMarkets {
			if cat.Add(m) {
				if err := natsPublisher.PublishMarketDiscovered(ctx, m); err != nil {
					logger.Error("failed to publish market discovered", zap.Error(err))
				} else {
					metrics.EventsPublished.WithLabelValues("MarketDiscovered").Inc()
				}
				if err := redisCache.SetMarket(ctx, m); err != nil {
					logger.Error("failed to set market in redis", zap.Error(err))
				}
			}
		}

		ids := make([]string, len(initialMarkets))
		for i, m := range initialMarkets {
			ids[i] = m.ID
		}
		if err := subscriber.Subscribe(ctx, ids); err != nil {
			logger.Error("failed to subscribe to markets", zap.Error(err))
		}
	}

	metrics.MarketsTracked.Set(float64(cat.Count()))

	// Fix #19: Sync stale count gauge with actual stale count
	metrics.StaleMarkets.Set(float64(cat.StaleCount()))

	logger.Info("scanner service started", zap.Int("markets", cat.Count()))

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("shutting down scanner service")
	cancel()

	// Fix #24: Wait for all goroutines to finish with a timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("all goroutines stopped gracefully")
	case <-time.After(10 * time.Second):
		logger.Warn("shutdown timed out waiting for goroutines")
	}
}

func runMarketDiscovery(ctx context.Context, cat *catalog.Catalog, restClient *rest.Client, publisher *adapters.NATSPublisher, cache *adapters.RedisCache, interval time.Duration, logger *zap.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			markets, err := restClient.FetchActiveMarkets(ctx)
			if err != nil {
				logger.Error("failed to fetch markets for discovery", zap.Error(err))
				continue
			}

			newCount := 0
			for _, m := range markets {
				if cat.Add(m) {
					newCount++
					if err := publisher.PublishMarketDiscovered(ctx, m); err != nil {
						logger.Error("failed to publish market discovered", zap.Error(err))
					}
					if err := cache.SetMarket(ctx, m); err != nil {
						logger.Error("failed to set market in redis", zap.Error(err))
					}
				}
			}

			if newCount > 0 {
				logger.Info("new markets discovered", zap.Int("count", newCount))
				metrics.MarketsTracked.Set(float64(cat.Count()))
			}
		}
	}
}

// Fix #26: Make metrics server bind address configurable
func startMetricsServer(bindAddr, port string, logger *zap.Logger, natsPublisher *adapters.NATSPublisher, redisCache *adapters.RedisCache, wsClient *websocket.Client) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		checks := map[string]bool{
			"nats":       natsPublisher.IsConnected(),
			"redis":      redisCache.IsConnected(),
			"websocket":  wsClient.IsConnected(),
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
		fmt.Fprintf(w, `{"status":"%s","checks":{"nats":%v,"redis":%v,"websocket":%v}}`, status, checks["nats"], checks["redis"], checks["websocket"])
	})

	addr := bindAddr + ":" + port
	logger.Info("starting metrics server", zap.String("addr", addr))
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Error("metrics server error", zap.Error(err))
	}
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

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapLevel),
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to init logger: %v", err))
	}

	return logger
}
