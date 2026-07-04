package scanner_test

import (
	"sync"
	"testing"
	"time"

	"github.com/pqap/services/scanner/internal/catalog"
	"github.com/shopspring/decimal"
)

func TestCatalog_Add(t *testing.T) {
	c := catalog.NewCatalog(nil)

	m := catalog.Market{
		ID:       "market-1",
		Slug:     "test-market",
		YESPrice: decimal.NewFromFloat(0.65),
		NOPrice:  decimal.NewFromFloat(0.30),
	}

	result := c.Add(m)
	if !result {
		t.Fatal("expected Add to return true for new market")
	}

	got, exists := c.Get("market-1")
	if !exists {
		t.Fatal("expected market to exist after Add")
	}
	if got.ID != "market-1" {
		t.Errorf("ID = %q, want market-1", got.ID)
	}
	if !got.IsActive {
		t.Error("expected market to be active after Add")
	}
	if got.LastUpdated.IsZero() {
		t.Error("expected LastUpdated to be set")
	}
}

func TestCatalog_AddDuplicate(t *testing.T) {
	c := catalog.NewCatalog(nil)

	m := catalog.Market{ID: "market-1"}
	c.Add(m)

	result := c.Add(m)
	if result {
		t.Fatal("expected Add to return false for duplicate market")
	}

	if c.Count() != 1 {
		t.Errorf("count = %d, want 1", c.Count())
	}
}

func TestCatalog_Update(t *testing.T) {
	onChangeCalled := false
	var onChangeMarket catalog.Market

	c := catalog.NewCatalog(func(m catalog.Market) {
		onChangeCalled = true
		onChangeMarket = m
	})

	m := catalog.Market{
		ID:       "market-1",
		YESPrice: decimal.NewFromFloat(0.65),
		NOPrice:  decimal.NewFromFloat(0.30),
	}
	c.Add(m)

	updated := catalog.Market{
		ID:       "market-1",
		YESPrice: decimal.NewFromFloat(0.70),
		NOPrice:  decimal.NewFromFloat(0.25),
		Spread:   decimal.NewFromFloat(0.05),
	}

	result := c.Update(updated)
	if !result {
		t.Fatal("expected Update to return true when price changed")
	}

	if !onChangeCalled {
		t.Error("expected onChange callback to be called")
	}
	if onChangeMarket.ID != "market-1" {
		t.Errorf("onChange market ID = %q, want market-1", onChangeMarket.ID)
	}

	got, _ := c.Get("market-1")
	if !got.YESPrice.Equal(decimal.NewFromFloat(0.70)) {
		t.Errorf("YES price = %v, want 0.70", got.YESPrice)
	}
	if !got.NOPrice.Equal(decimal.NewFromFloat(0.25)) {
		t.Errorf("NO price = %v, want 0.25", got.NOPrice)
	}
	if got.IsStale {
		t.Error("expected IsStale to be false after update")
	}
}

func TestCatalog_UpdateNoChange(t *testing.T) {
	onChangeCalled := false
	c := catalog.NewCatalog(func(m catalog.Market) {
		onChangeCalled = true
	})

	m := catalog.Market{
		ID:       "market-1",
		YESPrice: decimal.NewFromFloat(0.65),
		NOPrice:  decimal.NewFromFloat(0.30),
	}
	c.Add(m)

	result := c.Update(catalog.Market{
		ID:       "market-1",
		YESPrice: decimal.NewFromFloat(0.65),
		NOPrice:  decimal.NewFromFloat(0.30),
	})

	if result {
		t.Error("expected Update to return false when price unchanged")
	}
	if onChangeCalled {
		t.Error("expected onChange callback NOT to be called")
	}
}

func TestCatalog_UpdateNonexistent(t *testing.T) {
	c := catalog.NewCatalog(nil)

	result := c.Update(catalog.Market{
		ID:       "nonexistent",
		YESPrice: decimal.NewFromFloat(0.50),
	})

	if result {
		t.Error("expected Update to return false for nonexistent market")
	}
}

func TestCatalog_List(t *testing.T) {
	c := catalog.NewCatalog(nil)

	c.Add(catalog.Market{ID: "m1", IsActive: true})
	c.Add(catalog.Market{ID: "m2", IsActive: true})
	c.Add(catalog.Market{ID: "m3", IsActive: true})

	markets := c.List()
	if len(markets) != 3 {
		t.Errorf("List count = %d, want 3", len(markets))
	}

	ids := make(map[string]bool)
	for _, m := range markets {
		ids[m.ID] = true
	}
	for _, id := range []string{"m1", "m2", "m3"} {
		if !ids[id] {
			t.Errorf("expected market %q in list", id)
		}
	}
}

func TestCatalog_MarkStale(t *testing.T) {
	c := catalog.NewCatalog(nil)

	c.Add(catalog.Market{ID: "market-1"})

	result := c.MarkStale("market-1")
	if !result {
		t.Fatal("expected MarkStale to return true")
	}

	got, _ := c.Get("market-1")
	if !got.IsStale {
		t.Error("expected market to be stale")
	}

	result = c.MarkStale("market-1")
	if result {
		t.Error("expected MarkStale to return false for already stale market")
	}

	result = c.MarkStale("nonexistent")
	if result {
		t.Error("expected MarkStale to return false for nonexistent market")
	}
}

func TestCatalog_ConcurrentAccess(t *testing.T) {
	c := catalog.NewCatalog(nil)

	var wg sync.WaitGroup
	numWriters := 10
	numReaders := 10
	opsPerGoroutine := 100

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				marketID := "market-" + string(rune('A'+id))
				m := catalog.Market{
					ID:       marketID,
					YESPrice: decimal.NewFromFloat(float64(j) / 100.0),
					NOPrice:  decimal.NewFromFloat(float64(100-j) / 100.0),
				}
				c.Add(m)
				c.Update(m)
				c.MarkStale(marketID)
			}
		}(i)
	}

	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				c.List()
				c.Count()
				c.StaleCount()
				c.Get("market-A")
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent access test timed out — possible deadlock")
	}
}

func TestCatalog_StaleCount(t *testing.T) {
	c := catalog.NewCatalog(nil)

	c.Add(catalog.Market{ID: "m1"})
	c.Add(catalog.Market{ID: "m2"})
	c.Add(catalog.Market{ID: "m3"})

	if c.StaleCount() != 0 {
		t.Errorf("initial stale count = %d, want 0", c.StaleCount())
	}

	c.MarkStale("m1")
	c.MarkStale("m3")

	if c.StaleCount() != 2 {
		t.Errorf("stale count = %d, want 2", c.StaleCount())
	}
}

func TestCatalog_ClearStale(t *testing.T) {
	c := catalog.NewCatalog(nil)

	c.Add(catalog.Market{ID: "market-1"})

	c.MarkStale("market-1")
	got, _ := c.Get("market-1")
	if !got.IsStale {
		t.Error("expected market to be stale after MarkStale")
	}

	result := c.ClearStale("market-1")
	if !result {
		t.Fatal("expected ClearStale to return true")
	}

	got, _ = c.Get("market-1")
	if got.IsStale {
		t.Error("expected market to not be stale after ClearStale")
	}

	result = c.ClearStale("market-1")
	if result {
		t.Error("expected ClearStale to return false for already cleared market")
	}

	result = c.ClearStale("nonexistent")
	if result {
		t.Error("expected ClearStale to return false for nonexistent market")
	}
}

func TestCatalog_Count(t *testing.T) {
	c := catalog.NewCatalog(nil)

	if c.Count() != 0 {
		t.Errorf("empty catalog count = %d, want 0", c.Count())
	}

	c.Add(catalog.Market{ID: "m1"})
	c.Add(catalog.Market{ID: "m2"})

	if c.Count() != 2 {
		t.Errorf("count = %d, want 2", c.Count())
	}
}

func TestCatalog_Upsert_NewMarket(t *testing.T) {
	onChangeCalled := false
	c := catalog.NewCatalog(func(m catalog.Market) {
		onChangeCalled = true
	})

	existed, wasStale := c.Upsert(catalog.Market{
		ID:       "market-1",
		YESPrice: decimal.NewFromFloat(0.65),
		NOPrice:  decimal.NewFromFloat(0.30),
	})

	if existed {
		t.Error("expected existed=false for new market")
	}
	if wasStale {
		t.Error("expected wasStale=false for new market")
	}
	if onChangeCalled {
		t.Error("onChange should not be called for new market insert")
	}

	got, ok := c.Get("market-1")
	if !ok {
		t.Fatal("expected market to exist after Upsert")
	}
	if !got.IsActive {
		t.Error("expected market to be active")
	}
	if got.LastUpdated.IsZero() {
		t.Error("expected LastUpdated to be set")
	}
}

func TestCatalog_Upsert_ExistingMarket(t *testing.T) {
	onChangeCalled := false
	c := catalog.NewCatalog(func(m catalog.Market) {
		onChangeCalled = true
	})

	c.Add(catalog.Market{
		ID:       "market-1",
		YESPrice: decimal.NewFromFloat(0.65),
		NOPrice:  decimal.NewFromFloat(0.30),
	})

	existed, wasStale := c.Upsert(catalog.Market{
		ID:       "market-1",
		YESPrice: decimal.NewFromFloat(0.70),
		NOPrice:  decimal.NewFromFloat(0.25),
	})

	if !existed {
		t.Error("expected existed=true for existing market")
	}
	if wasStale {
		t.Error("expected wasStale=false for non-stale market")
	}
	if !onChangeCalled {
		t.Error("expected onChange to be called for price change")
	}

	got, _ := c.Get("market-1")
	if !got.YESPrice.Equal(decimal.NewFromFloat(0.70)) {
		t.Errorf("YESPrice = %v, want 0.70", got.YESPrice)
	}
	if got.IsStale {
		t.Error("expected IsStale=false after Upsert")
	}
}

func TestCatalog_Upsert_RecoveryFromStale(t *testing.T) {
	c := catalog.NewCatalog(nil)

	c.Add(catalog.Market{
		ID:       "market-1",
		YESPrice: decimal.NewFromFloat(0.65),
		NOPrice:  decimal.NewFromFloat(0.30),
	})
	c.MarkStale("market-1")

	existed, wasStale := c.Upsert(catalog.Market{
		ID:       "market-1",
		YESPrice: decimal.NewFromFloat(0.65),
		NOPrice:  decimal.NewFromFloat(0.30),
	})

	if !existed {
		t.Error("expected existed=true")
	}
	if !wasStale {
		t.Error("expected wasStale=true for stale market")
	}

	got, _ := c.Get("market-1")
	if got.IsStale {
		t.Error("expected IsStale=false after Upsert clears stale")
	}
}

func TestCatalog_Upsert_EmptyID(t *testing.T) {
	c := catalog.NewCatalog(nil)

	existed, wasStale := c.Upsert(catalog.Market{})

	if existed || wasStale {
		t.Error("expected both false for empty ID")
	}
	if c.Count() != 0 {
		t.Errorf("expected count=0, got %d", c.Count())
	}
}
