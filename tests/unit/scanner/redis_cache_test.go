package scanner_test

import (
	"context"
	"testing"
	"time"

	"github.com/pqap/services/scanner/adapters"
	"github.com/pqap/services/scanner/internal/catalog"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func TestRedisSetAndGetMarket(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Redis test in short mode")
	}

	logger, _ := zap.NewDevelopment()
	cache, err := adapters.NewRedisCache("localhost:6379", 5*time.Second, logger)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer cache.Close()

	ctx := context.Background()

	market := catalog.Market{
		ID:       "test-market-redis-1",
		Slug:     "test-redis",
		YESPrice: decimal.NewFromFloat(0.55),
		NOPrice:  decimal.NewFromFloat(0.42),
		IsActive: true,
	}

	err = cache.SetMarket(ctx, market)
	if err != nil {
		t.Fatalf("failed to set market: %v", err)
	}

	got, err := cache.GetMarket(ctx, "test-market-redis-1")
	if err != nil {
		t.Fatalf("failed to get market: %v", err)
	}

	if got == nil {
		t.Fatal("expected market to exist")
	}
	if got.ID != "test-market-redis-1" {
		t.Errorf("expected ID 'test-market-redis-1', got '%s'", got.ID)
	}
	if !got.YESPrice.Equal(decimal.NewFromFloat(0.55)) {
		t.Errorf("expected YES price 0.55, got %s", got.YESPrice)
	}
}

func TestRedisTTL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Redis test in short mode")
	}

	logger, _ := zap.NewDevelopment()
	cache, err := adapters.NewRedisCache("localhost:6379", 1*time.Second, logger)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer cache.Close()

	ctx := context.Background()

	market := catalog.Market{
		ID:       "test-market-ttl",
		YESPrice: decimal.NewFromFloat(0.50),
		NOPrice:  decimal.NewFromFloat(0.50),
	}

	cache.SetMarket(ctx, market)

	got, _ := cache.GetMarket(ctx, "test-market-ttl")
	if got == nil {
		t.Fatal("expected market to exist immediately after set")
	}

	time.Sleep(2 * time.Second)

	got, _ = cache.GetMarket(ctx, "test-market-ttl")
	if got != nil {
		t.Error("expected market to be expired after TTL")
	}
}

func TestRedisActiveMarketIDs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Redis test in short mode")
	}

	logger, _ := zap.NewDevelopment()
	cache, err := adapters.NewRedisCache("localhost:6379", 10*time.Second, logger)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer cache.Close()

	ctx := context.Background()

	cache.SetMarket(ctx, catalog.Market{ID: "m1", YESPrice: decimal.NewFromFloat(0.5), NOPrice: decimal.NewFromFloat(0.5)})
	cache.SetMarket(ctx, catalog.Market{ID: "m2", YESPrice: decimal.NewFromFloat(0.5), NOPrice: decimal.NewFromFloat(0.5)})

	ids, err := cache.GetActiveMarketIDs(ctx)
	if err != nil {
		t.Fatalf("failed to get active market IDs: %v", err)
	}

	if len(ids) < 2 {
		t.Errorf("expected at least 2 active markets, got %d", len(ids))
	}
}

func TestRedisKeyPattern(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Redis test in short mode")
	}

	logger, _ := zap.NewDevelopment()
	cache, err := adapters.NewRedisCache("localhost:6379", 5*time.Second, logger)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer cache.Close()

	ctx := context.Background()

	market := catalog.Market{
		ID:       "pattern-test-123",
		YESPrice: decimal.NewFromFloat(0.5),
		NOPrice:  decimal.NewFromFloat(0.5),
	}

	cache.SetMarket(ctx, market)

	got, _ := cache.GetMarket(ctx, "pattern-test-123")
	if got == nil {
		t.Fatal("expected market to exist with key pattern pqap:market:{id}")
	}
}

func TestRedisRemoveMarket(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Redis test in short mode")
	}

	logger, _ := zap.NewDevelopment()
	cache, err := adapters.NewRedisCache("localhost:6379", 10*time.Second, logger)
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer cache.Close()

	ctx := context.Background()

	cache.SetMarket(ctx, catalog.Market{ID: "remove-test", YESPrice: decimal.NewFromFloat(0.5), NOPrice: decimal.NewFromFloat(0.5)})

	cache.RemoveMarket(ctx, "remove-test")

	got, _ := cache.GetMarket(ctx, "remove-test")
	if got != nil {
		t.Error("expected market to be removed")
	}
}
