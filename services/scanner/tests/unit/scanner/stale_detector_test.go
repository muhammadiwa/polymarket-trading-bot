package scanner_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/pqap/services/scanner/internal/catalog"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func TestStaleDetector_DetectStale(t *testing.T) {
	c := catalog.NewCatalog(nil)

	oldTime := time.Now().Add(-10 * time.Minute)
	c.Add(catalog.Market{
		ID:          "stale-market",
		YESPrice:    decimal.NewFromFloat(0.50),
		LastUpdated: oldTime,
	})

	var staleMarket catalog.Market
	var mu sync.Mutex
	staleDetected := make(chan struct{}, 1)

	sd := catalog.NewStaleDetector(
		c,
		5*time.Minute,
		100*time.Millisecond,
		zap.NewNop(),
		func(m catalog.Market) {
			mu.Lock()
			staleMarket = m
			mu.Unlock()
			select {
			case staleDetected <- struct{}{}:
			default:
			}
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go sd.Run(ctx)

	select {
	case <-staleDetected:
	case <-ctx.Done():
		t.Fatal("timed out waiting for stale detection")
	}

	mu.Lock()
	defer mu.Unlock()
	if staleMarket.ID != "stale-market" {
		t.Errorf("stale market ID = %q, want stale-market", staleMarket.ID)
	}

	got, _ := c.Get("stale-market")
	if !got.IsStale {
		t.Error("expected market to be marked stale in catalog")
	}
}

func TestStaleDetector_FreshMarket(t *testing.T) {
	c := catalog.NewCatalog(nil)

	c.Add(catalog.Market{
		ID:          "fresh-market",
		YESPrice:    decimal.NewFromFloat(0.50),
		LastUpdated: time.Now(),
	})

	staleCalled := false
	sd := catalog.NewStaleDetector(
		c,
		5*time.Minute,
		50*time.Millisecond,
		zap.NewNop(),
		func(m catalog.Market) {
			staleCalled = true
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	sd.Run(ctx)

	if staleCalled {
		t.Error("onStale should not be called for fresh market")
	}

	got, _ := c.Get("fresh-market")
	if got.IsStale {
		t.Error("fresh market should not be marked stale")
	}
}

func TestStaleDetector_StaleCount(t *testing.T) {
	c := catalog.NewCatalog(nil)

	oldTime := time.Now().Add(-10 * time.Minute)
	c.Add(catalog.Market{ID: "m1", YESPrice: decimal.NewFromFloat(0.50), LastUpdated: oldTime})
	c.Add(catalog.Market{ID: "m2", YESPrice: decimal.NewFromFloat(0.50), LastUpdated: oldTime})
	c.Add(catalog.Market{ID: "m3", YESPrice: decimal.NewFromFloat(0.50), LastUpdated: time.Now()})

	var wg sync.WaitGroup
	wg.Add(2)
	staleCount := 0
	var mu sync.Mutex

	sd := catalog.NewStaleDetector(
		c,
		5*time.Minute,
		100*time.Millisecond,
		zap.NewNop(),
		func(m catalog.Market) {
			mu.Lock()
			staleCount++
			mu.Unlock()
			wg.Done()
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go sd.Run(ctx)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("timed out waiting for stale detection of 2 markets")
	}

	mu.Lock()
	defer mu.Unlock()
	if staleCount != 2 {
		t.Errorf("stale callback count = %d, want 2", staleCount)
	}

	if c.StaleCount() != 2 {
		t.Errorf("catalog.StaleCount() = %d, want 2", c.StaleCount())
	}
}

func TestStaleDetector_ConfigurableThreshold(t *testing.T) {
	c := catalog.NewCatalog(nil)

	oldTime := time.Now().Add(-10 * time.Second)
	c.Add(catalog.Market{
		ID:          "market-1",
		YESPrice:    decimal.NewFromFloat(0.50),
		LastUpdated: oldTime,
	})

	staleDetected := make(chan struct{}, 1)
	sd := catalog.NewStaleDetector(
		c,
		5*time.Second,
		100*time.Millisecond,
		zap.NewNop(),
		func(m catalog.Market) {
			select {
			case staleDetected <- struct{}{}:
			default:
			}
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go sd.Run(ctx)

	select {
	case <-staleDetected:
		t.Log("market detected as stale with 5s threshold")
	case <-ctx.Done():
		t.Fatal("timed out waiting for stale detection with configurable threshold")
	}

	got, _ := c.Get("market-1")
	if !got.IsStale {
		t.Error("expected market to be marked stale")
	}
}

func TestStaleDetector_MarketRecovery(t *testing.T) {
	c := catalog.NewCatalog(nil)

	oldTime := time.Now().Add(-10 * time.Minute)
	c.Add(catalog.Market{
		ID:          "market-1",
		YESPrice:    decimal.NewFromFloat(0.50),
		LastUpdated: oldTime,
	})

	staleDetected := make(chan struct{}, 1)
	sd := catalog.NewStaleDetector(
		c,
		5*time.Minute,
		100*time.Millisecond,
		zap.NewNop(),
		func(m catalog.Market) {
			select {
			case staleDetected <- struct{}{}:
			default:
			}
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go sd.Run(ctx)

	select {
	case <-staleDetected:
	case <-ctx.Done():
		t.Fatal("timed out waiting for stale detection")
	}

	got, _ := c.Get("market-1")
	if !got.IsStale {
		t.Fatal("expected market to be stale")
	}

	c.Update(catalog.Market{
		ID:       "market-1",
		YESPrice: decimal.NewFromFloat(0.60),
		NOPrice:  decimal.NewFromFloat(0.35),
	})

	got, _ = c.Get("market-1")
	if got.IsStale {
		t.Error("expected market to be recovered (not stale) after price update")
	}
	if c.StaleCount() != 0 {
		t.Errorf("expected stale count to be 0 after recovery, got %d", c.StaleCount())
	}
}

func TestStaleDetector_MultipleMarketsStaleSimultaneously(t *testing.T) {
	c := catalog.NewCatalog(nil)

	oldTime := time.Now().Add(-10 * time.Minute)
	c.Add(catalog.Market{ID: "m1", YESPrice: decimal.NewFromFloat(0.50), LastUpdated: oldTime})
	c.Add(catalog.Market{ID: "m2", YESPrice: decimal.NewFromFloat(0.60), LastUpdated: oldTime})
	c.Add(catalog.Market{ID: "m3", YESPrice: decimal.NewFromFloat(0.70), LastUpdated: oldTime})

	var mu sync.Mutex
	staleMarkets := make(map[string]bool)
	allStale := make(chan struct{}, 1)

	sd := catalog.NewStaleDetector(
		c,
		5*time.Minute,
		100*time.Millisecond,
		zap.NewNop(),
		func(m catalog.Market) {
			mu.Lock()
			staleMarkets[m.ID] = true
			if len(staleMarkets) >= 3 {
				select {
				case allStale <- struct{}{}:
				default:
				}
			}
			mu.Unlock()
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go sd.Run(ctx)

	select {
	case <-allStale:
	case <-ctx.Done():
		t.Fatal("timed out waiting for all 3 markets to be detected as stale")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(staleMarkets) != 3 {
		t.Errorf("expected 3 stale markets, got %d", len(staleMarkets))
	}

	if c.StaleCount() != 3 {
		t.Errorf("catalog.StaleCount() = %d, want 3", c.StaleCount())
	}
}

func TestStaleDetector_CheckMethod(t *testing.T) {
	c := catalog.NewCatalog(nil)

	oldTime := time.Now().Add(-10 * time.Minute)
	c.Add(catalog.Market{ID: "m1", YESPrice: decimal.NewFromFloat(0.50), LastUpdated: oldTime})
	c.Add(catalog.Market{ID: "m2", YESPrice: decimal.NewFromFloat(0.60), LastUpdated: time.Now()})

	sd := catalog.NewStaleDetector(
		c,
		5*time.Minute,
		100*time.Millisecond,
		zap.NewNop(),
		func(m catalog.Market) {},
	)

	count := sd.Check()
	if count != 1 {
		t.Errorf("Check() returned %d, want 1", count)
	}

	got, _ := c.Get("m1")
	if !got.IsStale {
		t.Error("expected m1 to be stale after Check()")
	}

	got2, _ := c.Get("m2")
	if got2.IsStale {
		t.Error("expected m2 to not be stale after Check()")
	}
}
