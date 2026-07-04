package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/pqap/services/scanner/adapters"
	"github.com/pqap/services/scanner/internal/catalog"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func TestRedisSetAndGetMarket(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("skipping integration test: set INTEGRATION_TEST=1 to enable")
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	logger, _ := zap.NewDevelopment()
	cache, err := adapters.NewRedisCache(redisURL, 10*time.Second, logger)
	if err != nil {
		t.Fatalf("failed to connect to Redis: %v", err)
	}
	defer cache.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	market := catalog.Market{
		ID:             "integration-test-market-redis-001",
		Slug:           "test-redis-market",
		YESPrice:       decimal.NewFromFloat(0.55),
		NOPrice:        decimal.NewFromFloat(0.45),
		Spread:         decimal.NewFromFloat(0.0),
		Volume24h:      decimal.NewFromFloat(1000.0),
		LiquidityDepth: decimal.NewFromFloat(500.0),
		IsActive:       true,
		LastUpdated:    time.Now(),
	}

	if err := cache.SetMarket(ctx, market); err != nil {
		t.Fatalf("failed to set market: %v", err)
	}

	got, err := cache.GetMarket(ctx, market.ID)
	if err != nil {
		t.Fatalf("failed to get market: %v", err)
	}
	if got == nil {
		t.Fatal("expected market, got nil")
	}
	if got.ID != market.ID {
		t.Errorf("expected ID %s, got %s", market.ID, got.ID)
	}
}

func TestRedisRemoveMarket(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("skipping integration test: set INTEGRATION_TEST=1 to enable")
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	logger, _ := zap.NewDevelopment()
	cache, err := adapters.NewRedisCache(redisURL, 10*time.Second, logger)
	if err != nil {
		t.Fatalf("failed to connect to Redis: %v", err)
	}
	defer cache.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	marketID := "integration-test-market-redis-remove"

	market := catalog.Market{
		ID:       marketID,
		YESPrice: decimal.NewFromFloat(0.5),
		NOPrice:  decimal.NewFromFloat(0.5),
	}

	if err := cache.SetMarket(ctx, market); err != nil {
		t.Fatalf("failed to set market: %v", err)
	}

	if err := cache.RemoveMarket(ctx, marketID); err != nil {
		t.Fatalf("failed to remove market: %v", err)
	}

	got, err := cache.GetMarket(ctx, marketID)
	if err != nil {
		t.Fatalf("failed to get market after removal: %v", err)
	}
	if got != nil {
		t.Error("expected nil after removal, got market")
	}
}

func TestRedisGetActiveMarketIDs(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("skipping integration test: set INTEGRATION_TEST=1 to enable")
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	logger, _ := zap.NewDevelopment()
	cache, err := adapters.NewRedisCache(redisURL, 10*time.Second, logger)
	if err != nil {
		t.Fatalf("failed to connect to Redis: %v", err)
	}
	defer cache.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ids, err := cache.GetActiveMarketIDs(ctx)
	if err != nil {
		t.Fatalf("failed to get active market IDs: %v", err)
	}

	t.Logf("found %d active market IDs", len(ids))
}
