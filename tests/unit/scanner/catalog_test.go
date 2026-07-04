package scanner_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/pqap/services/scanner/internal/catalog"
	"github.com/shopspring/decimal"
)

func TestCatalogAdd(t *testing.T) {
	cat := catalog.NewCatalog(nil)

	market := catalog.Market{
		ID:       "market-1",
		Slug:     "test-market",
		YESPrice: decimal.NewFromFloat(0.55),
		NOPrice:  decimal.NewFromFloat(0.42),
		IsActive: true,
	}

	added := cat.Add(market)
	if !added {
		t.Fatal("expected market to be added")
	}

	if cat.Count() != 1 {
		t.Errorf("expected 1 market, got %d", cat.Count())
	}

	got, exists := cat.Get("market-1")
	if !exists {
		t.Fatal("expected market to exist")
	}
	if got.ID != "market-1" {
		t.Errorf("expected ID 'market-1', got '%s'", got.ID)
	}
	if !got.YESPrice.Equal(decimal.NewFromFloat(0.55)) {
		t.Errorf("expected YES price 0.55, got %s", got.YESPrice)
	}
}

func TestCatalogAddDuplicate(t *testing.T) {
	cat := catalog.NewCatalog(nil)

	market := catalog.Market{
		ID:       "market-1",
		Slug:     "test-market",
		YESPrice: decimal.NewFromFloat(0.55),
		NOPrice:  decimal.NewFromFloat(0.42),
	}

	cat.Add(market)
	added := cat.Add(market)
	if added {
		t.Fatal("expected duplicate add to return false")
	}

	if cat.Count() != 1 {
		t.Errorf("expected 1 market, got %d", cat.Count())
	}
}

func TestCatalogUpdate(t *testing.T) {
	onChangeCalled := false
	cat := catalog.NewCatalog(func(m catalog.Market) {
		onChangeCalled = true
	})

	cat.Add(catalog.Market{
		ID:       "market-1",
		YESPrice: decimal.NewFromFloat(0.55),
		NOPrice:  decimal.NewFromFloat(0.42),
	})

	updated := cat.Update(catalog.Market{
		ID:       "market-1",
		YESPrice: decimal.NewFromFloat(0.60),
		NOPrice:  decimal.NewFromFloat(0.38),
	})

	if !updated {
		t.Fatal("expected update to return true (price changed)")
	}
	if !onChangeCalled {
		t.Fatal("expected onChange callback to be called")
	}

	got, _ := cat.Get("market-1")
	if !got.YESPrice.Equal(decimal.NewFromFloat(0.60)) {
		t.Errorf("expected YES price 0.60, got %s", got.YESPrice)
	}
}

func TestCatalogUpdateNoPriceChange(t *testing.T) {
	onChangeCalled := false
	cat := catalog.NewCatalog(func(m catalog.Market) {
		onChangeCalled = true
	})

	cat.Add(catalog.Market{
		ID:       "market-1",
		YESPrice: decimal.NewFromFloat(0.55),
		NOPrice:  decimal.NewFromFloat(0.42),
	})

	onChangeCalled = false
	updated := cat.Update(catalog.Market{
		ID:       "market-1",
		YESPrice: decimal.NewFromFloat(0.55),
		NOPrice:  decimal.NewFromFloat(0.42),
	})

	if updated {
		t.Fatal("expected update to return false (no price change)")
	}
	if onChangeCalled {
		t.Fatal("expected onChange callback not to be called")
	}
}

func TestCatalogUpdateNonexistent(t *testing.T) {
	cat := catalog.NewCatalog(nil)

	updated := cat.Update(catalog.Market{
		ID:       "nonexistent",
		YESPrice: decimal.NewFromFloat(0.55),
		NOPrice:  decimal.NewFromFloat(0.42),
	})

	if updated {
		t.Fatal("expected update of nonexistent market to return false")
	}
}

func TestCatalogList(t *testing.T) {
	cat := catalog.NewCatalog(nil)

	cat.Add(catalog.Market{ID: "m1"})
	cat.Add(catalog.Market{ID: "m2"})
	cat.Add(catalog.Market{ID: "m3"})

	list := cat.List()
	if len(list) != 3 {
		t.Errorf("expected 3 markets, got %d", len(list))
	}
}

func TestCatalogMarkStale(t *testing.T) {
	cat := catalog.NewCatalog(nil)

	cat.Add(catalog.Market{ID: "market-1"})

	marked := cat.MarkStale("market-1")
	if !marked {
		t.Fatal("expected market to be marked stale")
	}

	got, _ := cat.Get("market-1")
	if !got.IsStale {
		t.Fatal("expected market to be stale")
	}

	marked = cat.MarkStale("market-1")
	if marked {
		t.Fatal("expected already-stale market to return false")
	}
}

func TestCatalogConcurrentAccess(t *testing.T) {
	cat := catalog.NewCatalog(nil)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cat.Add(catalog.Market{
				ID:       fmt.Sprintf("market-%d", i),
				YESPrice: decimal.NewFromFloat(0.5),
				NOPrice:  decimal.NewFromFloat(0.5),
			})
		}(i)
	}
	wg.Wait()

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cat.List()
			cat.Count()
		}()
	}
	wg.Wait()
}
