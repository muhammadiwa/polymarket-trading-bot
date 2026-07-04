package scanner_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pqap/services/scanner/internal/catalog"
	"github.com/pqap/services/scanner/internal/rest"
	"github.com/pqap/services/scanner/internal/websocket"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func TestReconciler_PricesMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		markets := []rest.MarketResponse{
			{
				ConditionID: "market-1",
				Active:      true,
				Tokens: []rest.TokenResponse{
					{TokenID: "yes", Outcome: "Yes", Price: "0.65"},
					{TokenID: "no", Outcome: "No", Price: "0.30"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rest.MarketsResponse{
			Markets: markets,
		})
	}))
	defer server.Close()

	cat := catalog.NewCatalog(nil)
	cat.Add(catalog.Market{
		ID:       "market-1",
		YESPrice: decimal.NewFromFloat(0.65),
		NOPrice:  decimal.NewFromFloat(0.30),
	})

	logger := zap.NewNop()
	client := rest.NewClient(server.URL, logger)
	reconciler := websocket.NewReconciler(cat, client, logger)

	ctx := context.Background()
	err := reconciler.Reconcile(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, _ := cat.Get("market-1")
	if !m.YESPrice.Equal(decimal.NewFromFloat(0.65)) {
		t.Errorf("YESPrice = %v, want 0.65", m.YESPrice)
	}
	if !m.NOPrice.Equal(decimal.NewFromFloat(0.30)) {
		t.Errorf("NOPrice = %v, want 0.30", m.NOPrice)
	}
}

func TestReconciler_PriceDiscrepancyGreaterThanOneTick(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		markets := []rest.MarketResponse{
			{
				ConditionID: "market-1",
				Active:      true,
				Tokens: []rest.TokenResponse{
					{TokenID: "yes", Outcome: "Yes", Price: "0.70"},
					{TokenID: "no", Outcome: "No", Price: "0.25"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rest.MarketsResponse{
			Markets: markets,
		})
	}))
	defer server.Close()

	cat := catalog.NewCatalog(nil)
	cat.Add(catalog.Market{
		ID:       "market-1",
		YESPrice: decimal.NewFromFloat(0.65),
		NOPrice:  decimal.NewFromFloat(0.30),
	})

	logger := zap.NewNop()
	client := rest.NewClient(server.URL, logger)
	reconciler := websocket.NewReconciler(cat, client, logger)

	ctx := context.Background()
	err := reconciler.Reconcile(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, _ := cat.Get("market-1")
	if !m.YESPrice.Equal(decimal.NewFromFloat(0.70)) {
		t.Errorf("YESPrice = %v, want 0.70 (should be updated to snapshot)", m.YESPrice)
	}
	if !m.NOPrice.Equal(decimal.NewFromFloat(0.25)) {
		t.Errorf("NOPrice = %v, want 0.25 (should be updated to snapshot)", m.NOPrice)
	}
}

func TestReconciler_PriceDiscrepancyLessThanOneTick(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		markets := []rest.MarketResponse{
			{
				ConditionID: "market-1",
				Active:      true,
				Tokens: []rest.TokenResponse{
					{TokenID: "yes", Outcome: "Yes", Price: "0.655"},
					{TokenID: "no", Outcome: "No", Price: "0.305"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rest.MarketsResponse{
			Markets: markets,
		})
	}))
	defer server.Close()

	cat := catalog.NewCatalog(nil)
	cat.Add(catalog.Market{
		ID:       "market-1",
		YESPrice: decimal.NewFromFloat(0.65),
		NOPrice:  decimal.NewFromFloat(0.30),
	})

	logger := zap.NewNop()
	client := rest.NewClient(server.URL, logger)
	reconciler := websocket.NewReconciler(cat, client, logger)

	ctx := context.Background()
	err := reconciler.Reconcile(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, _ := cat.Get("market-1")
	if !m.YESPrice.Equal(decimal.NewFromFloat(0.655)) {
		t.Errorf("YESPrice = %v, want 0.655", m.YESPrice)
	}
}

func TestReconciler_ClearsStaleFlags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		markets := []rest.MarketResponse{
			{
				ConditionID: "market-1",
				Active:      true,
				Tokens: []rest.TokenResponse{
					{TokenID: "yes", Outcome: "Yes", Price: "0.65"},
					{TokenID: "no", Outcome: "No", Price: "0.30"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rest.MarketsResponse{
			Markets: markets,
		})
	}))
	defer server.Close()

	cat := catalog.NewCatalog(nil)
	cat.Add(catalog.Market{
		ID:       "market-1",
		YESPrice: decimal.NewFromFloat(0.65),
		NOPrice:  decimal.NewFromFloat(0.30),
	})
	cat.MarkStale("market-1")

	m, _ := cat.Get("market-1")
	if !m.IsStale {
		t.Fatal("market should be stale before reconciliation")
	}

	logger := zap.NewNop()
	client := rest.NewClient(server.URL, logger)
	reconciler := websocket.NewReconciler(cat, client, logger)

	ctx := context.Background()
	err := reconciler.Reconcile(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, _ = cat.Get("market-1")
	if m.IsStale {
		t.Error("market should not be stale after reconciliation")
	}
}

func TestReconciler_EmptyCatalog(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rest.MarketsResponse{})
	}))
	defer server.Close()

	cat := catalog.NewCatalog(nil)
	logger := zap.NewNop()
	client := rest.NewClient(server.URL, logger)
	reconciler := websocket.NewReconciler(cat, client, logger)

	ctx := context.Background()
	err := reconciler.Reconcile(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 0 {
		t.Errorf("expected 0 API calls for empty catalog, got %d", callCount)
	}
}

func TestReconciler_MultipleMarkets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ids := parseReconcilerIDParams(r.URL.Query().Get("id"))
		allMarkets := map[string]rest.MarketResponse{
			"market-1": {
				ConditionID: "market-1",
				Active:      true,
				Tokens: []rest.TokenResponse{
					{TokenID: "yes", Outcome: "Yes", Price: "0.65"},
					{TokenID: "no", Outcome: "No", Price: "0.30"},
				},
			},
			"market-2": {
				ConditionID: "market-2",
				Active:      true,
				Tokens: []rest.TokenResponse{
					{TokenID: "yes", Outcome: "Yes", Price: "0.80"},
					{TokenID: "no", Outcome: "No", Price: "0.15"},
				},
			},
			"market-3": {
				ConditionID: "market-3",
				Active:      true,
				Tokens: []rest.TokenResponse{
					{TokenID: "yes", Outcome: "Yes", Price: "0.50"},
					{TokenID: "no", Outcome: "No", Price: "0.45"},
				},
			},
		}
		var markets []rest.MarketResponse
		for _, id := range ids {
			if m, ok := allMarkets[id]; ok {
				markets = append(markets, m)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rest.MarketsResponse{
			Markets: markets,
		})
	}))
	defer server.Close()

	cat := catalog.NewCatalog(nil)
	cat.Add(catalog.Market{ID: "market-1", YESPrice: decimal.NewFromFloat(0.65), NOPrice: decimal.NewFromFloat(0.30)})
	cat.Add(catalog.Market{ID: "market-2", YESPrice: decimal.NewFromFloat(0.75), NOPrice: decimal.NewFromFloat(0.20)})
	cat.Add(catalog.Market{ID: "market-3", YESPrice: decimal.NewFromFloat(0.50), NOPrice: decimal.NewFromFloat(0.45)})

	logger := zap.NewNop()
	client := rest.NewClient(server.URL, logger)
	reconciler := websocket.NewReconciler(cat, client, logger)

	ctx := context.Background()
	err := reconciler.Reconcile(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m1, _ := cat.Get("market-1")
	if !m1.YESPrice.Equal(decimal.NewFromFloat(0.65)) {
		t.Errorf("market-1 YESPrice = %v, want 0.65", m1.YESPrice)
	}

	m2, _ := cat.Get("market-2")
	if !m2.YESPrice.Equal(decimal.NewFromFloat(0.80)) {
		t.Errorf("market-2 YESPrice = %v, want 0.80 (should be updated)", m2.YESPrice)
	}

	m3, _ := cat.Get("market-3")
	if !m3.YESPrice.Equal(decimal.NewFromFloat(0.50)) {
		t.Errorf("market-3 YESPrice = %v, want 0.50", m3.YESPrice)
	}
}

func TestReconciler_RESTError(t *testing.T) {
	callCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cat := catalog.NewCatalog(nil)
	cat.Add(catalog.Market{ID: "market-1", YESPrice: decimal.NewFromFloat(0.65)})

	logger := zap.NewNop()
	client := rest.NewClient(server.URL, logger)
	reconciler := websocket.NewReconciler(cat, client, logger)

	ctx := context.Background()
	err := reconciler.Reconcile(ctx)
	if err == nil {
		t.Fatal("expected error from REST failure")
	}
}

func TestReconciler_MarketNotInCatalog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		markets := []rest.MarketResponse{
			{
				ConditionID: "market-1",
				Active:      true,
				Tokens: []rest.TokenResponse{
					{TokenID: "yes", Outcome: "Yes", Price: "0.65"},
				},
			},
			{
				ConditionID: "market-unknown",
				Active:      true,
				Tokens: []rest.TokenResponse{
					{TokenID: "yes", Outcome: "Yes", Price: "0.50"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rest.MarketsResponse{
			Markets: markets,
		})
	}))
	defer server.Close()

	cat := catalog.NewCatalog(nil)
	cat.Add(catalog.Market{ID: "market-1", YESPrice: decimal.NewFromFloat(0.65)})

	logger := zap.NewNop()
	client := rest.NewClient(server.URL, logger)
	reconciler := websocket.NewReconciler(cat, client, logger)

	ctx := context.Background()
	err := reconciler.Reconcile(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, exists := cat.Get("market-unknown")
	if exists {
		t.Error("market-unknown should not be added to catalog during reconciliation")
	}
}

func TestReconciler_BatchFetching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		ids := parseReconcilerIDParams(r.URL.Query().Get("id"))
		var markets []rest.MarketResponse
		for _, id := range ids {
			markets = append(markets, rest.MarketResponse{
				ConditionID: id,
				Active:      true,
				Tokens: []rest.TokenResponse{
					{TokenID: "yes", Outcome: "Yes", Price: "0.60"},
				},
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rest.MarketsResponse{
			Markets: markets,
		})
	}))
	defer server.Close()

	cat := catalog.NewCatalog(nil)
	for i := 0; i < 150; i++ {
		cat.Add(catalog.Market{
			ID:       fmt.Sprintf("market-%d", i),
			YESPrice: decimal.NewFromFloat(0.55),
		})
	}

	logger := zap.NewNop()
	client := rest.NewClient(server.URL, logger)
	reconciler := websocket.NewReconciler(cat, client, logger)

	ctx := context.Background()
	err := reconciler.Reconcile(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 batch API calls, got %d", callCount)
	}
}

func TestReconciler_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rest.MarketsResponse{})
	}))
	defer server.Close()

	cat := catalog.NewCatalog(nil)
	cat.Add(catalog.Market{ID: "market-1", YESPrice: decimal.NewFromFloat(0.65)})

	logger := zap.NewNop()
	client := rest.NewClient(server.URL, logger)
	reconciler := websocket.NewReconciler(cat, client, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := reconciler.Reconcile(ctx)
	if err == nil {
		t.Fatal("expected error from context cancellation")
	}
}

// parseReconcilerIDParams extracts repeated id query parameters from a single comma-separated string.
func parseReconcilerIDParams(raw string) []string {
	var ids []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			ids = append(ids, part)
		}
	}
	return ids
}
