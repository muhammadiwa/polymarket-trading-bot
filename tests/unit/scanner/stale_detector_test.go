package scanner_test

import (
	"context"
	"testing"
	"time"

	"github.com/pqap/services/scanner/internal/catalog"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func TestStaleDetectorMarksStaleMarkets(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cat := catalog.NewCatalog(nil)

	oldTime := time.Now().Add(-35 * time.Second)
	cat.Add(catalog.Market{
		ID:          "market-old",
		YESPrice:    decimal.NewFromFloat(0.55),
		NOPrice:     decimal.NewFromFloat(0.42),
		LastUpdated: oldTime,
	})

	staleMarkets := make([]string, 0)
	detector := catalog.NewStaleDetector(cat, 30*time.Second, 100*time.Millisecond, logger, func(m catalog.Market) {
		staleMarkets = append(staleMarkets, m.ID)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	detector.Run(ctx)

	if len(staleMarkets) == 0 {
		t.Fatal("expected stale markets to be detected")
	}

	found := false
	for _, id := range staleMarkets {
		if id == "market-old" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected market-old to be detected as stale")
	}
}

func TestStaleDetectorIgnoresFreshMarkets(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cat := catalog.NewCatalog(nil)

	cat.Add(catalog.Market{
		ID:       "market-fresh",
		YESPrice: decimal.NewFromFloat(0.55),
		NOPrice:  decimal.NewFromFloat(0.42),
	})

	staleMarkets := make([]string, 0)
	detector := catalog.NewStaleDetector(cat, 30*time.Second, 100*time.Millisecond, logger, func(m catalog.Market) {
		staleMarkets = append(staleMarkets, m.ID)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	detector.Run(ctx)

	if len(staleMarkets) > 0 {
		t.Errorf("expected no stale markets, got %d", len(staleMarkets))
	}
}

func TestStaleDetectorStaleCount(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cat := catalog.NewCatalog(nil)

	oldTime := time.Now().Add(-35 * time.Second)
	cat.Add(catalog.Market{ID: "m1", LastUpdated: oldTime})
	cat.Add(catalog.Market{ID: "m2", LastUpdated: oldTime})
	cat.Add(catalog.Market{ID: "m3", LastUpdated: time.Now()})

	detector := catalog.NewStaleDetector(cat, 30*time.Second, 100*time.Millisecond, logger, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	detector.Run(ctx)

	if cat.StaleCount() != 2 {
		t.Errorf("expected 2 stale markets, got %d", cat.StaleCount())
	}
}
