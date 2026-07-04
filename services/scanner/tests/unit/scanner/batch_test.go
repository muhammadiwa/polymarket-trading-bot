package scanner_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/pqap/services/scanner/internal/catalog"
	"github.com/pqap/services/scanner/internal/rest"
	"go.uber.org/zap"
)

func TestBatchFetcher_EmptyMarketList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rest.MarketsResponse{
			Markets: []rest.MarketResponse{},
		})
	}))
	defer server.Close()

	logger := zap.NewNop()
	client := rest.NewClient(server.URL, logger)
	batcher := rest.NewBatchFetcher(client, 100, logger)

	ctx := context.Background()
	markets, err := batcher.FetchMarketBatch(ctx, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(markets) != 0 {
		t.Errorf("expected 0 markets, got %d", len(markets))
	}
}

func TestBatchFetcher_SingleBatchUnder100(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		ids := parseIDParams(r.URL.Query())
		var markets []rest.MarketResponse
		for _, id := range ids {
			markets = append(markets, rest.MarketResponse{
				ConditionID: id,
				Slug:        id,
				Active:      true,
				Tokens: []rest.TokenResponse{
					{TokenID: "yes", Outcome: "Yes", Price: "0.60"},
					{TokenID: "no", Outcome: "No", Price: "0.35"},
				},
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rest.MarketsResponse{
			Markets: markets,
		})
	}))
	defer server.Close()

	logger := zap.NewNop()
	client := rest.NewClient(server.URL, logger)
	batcher := rest.NewBatchFetcher(client, 100, logger)

	ids := make([]string, 50)
	for i := 0; i < 50; i++ {
		ids[i] = fmt.Sprintf("market-%d", i)
	}

	ctx := context.Background()
	markets, err := batcher.FetchMarketBatch(ctx, ids)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(markets) != 50 {
		t.Errorf("expected 50 markets, got %d", len(markets))
	}

	if callCount != 1 {
		t.Errorf("expected 1 API call, got %d", callCount)
	}
}

func TestBatchFetcher_ExactBatchSize100(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		ids := parseIDParams(r.URL.Query())
		var markets []rest.MarketResponse
		for _, id := range ids {
			markets = append(markets, rest.MarketResponse{
				ConditionID: id,
				Slug:        id,
				Active:      true,
				Tokens: []rest.TokenResponse{
					{TokenID: "yes", Outcome: "Yes", Price: "0.60"},
					{TokenID: "no", Outcome: "No", Price: "0.35"},
				},
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rest.MarketsResponse{
			Markets: markets,
		})
	}))
	defer server.Close()

	logger := zap.NewNop()
	client := rest.NewClient(server.URL, logger)
	batcher := rest.NewBatchFetcher(client, 100, logger)

	ids := make([]string, 100)
	for i := 0; i < 100; i++ {
		ids[i] = fmt.Sprintf("market-%d", i)
	}

	ctx := context.Background()
	markets, err := batcher.FetchMarketBatch(ctx, ids)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(markets) != 100 {
		t.Errorf("expected 100 markets, got %d", len(markets))
	}

	if callCount != 1 {
		t.Errorf("expected 1 API call, got %d", callCount)
	}
}

func TestBatchFetcher_MultipleBatches150(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		ids := parseIDParams(r.URL.Query())
		var markets []rest.MarketResponse
		for _, id := range ids {
			markets = append(markets, rest.MarketResponse{
				ConditionID: id,
				Slug:        id,
				Active:      true,
				Tokens: []rest.TokenResponse{
					{TokenID: "yes", Outcome: "Yes", Price: "0.60"},
					{TokenID: "no", Outcome: "No", Price: "0.35"},
				},
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rest.MarketsResponse{
			Markets: markets,
		})
	}))
	defer server.Close()

	logger := zap.NewNop()
	client := rest.NewClient(server.URL, logger)
	batcher := rest.NewBatchFetcher(client, 100, logger)

	ids := make([]string, 150)
	for i := 0; i < 150; i++ {
		ids[i] = fmt.Sprintf("market-%d", i)
	}

	ctx := context.Background()
	markets, err := batcher.FetchMarketBatch(ctx, ids)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(markets) != 100 {
		t.Errorf("expected 100 markets (capped by maxBatch), got %d", len(markets))
	}

	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
}

func TestBatchFetcher_MultipleBatches250(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		ids := parseIDParams(r.URL.Query())
		var markets []rest.MarketResponse
		for _, id := range ids {
			markets = append(markets, rest.MarketResponse{
				ConditionID: id,
				Slug:        id,
				Active:      true,
				Tokens: []rest.TokenResponse{
					{TokenID: "yes", Outcome: "Yes", Price: "0.60"},
					{TokenID: "no", Outcome: "No", Price: "0.35"},
				},
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rest.MarketsResponse{
			Markets: markets,
		})
	}))
	defer server.Close()

	logger := zap.NewNop()
	client := rest.NewClient(server.URL, logger)
	batcher := rest.NewBatchFetcher(client, 100, logger)

	ids := make([]string, 250)
	for i := 0; i < 250; i++ {
		ids[i] = fmt.Sprintf("market-%d", i)
	}

	ctx := context.Background()
	markets, err := batcher.FetchMarketBatch(ctx, ids)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(markets) != 100 {
		t.Errorf("expected 100 markets (capped by maxBatch), got %d", len(markets))
	}

	if callCount != 3 {
		t.Errorf("expected 3 API calls, got %d", callCount)
	}
}

func TestBatchFetcher_FiltersInactiveMarkets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		markets := []rest.MarketResponse{
			{ConditionID: "active-1", Active: true, Tokens: []rest.TokenResponse{
				{TokenID: "yes", Outcome: "Yes", Price: "0.60"},
			}},
			{ConditionID: "inactive-1", Active: false, Tokens: []rest.TokenResponse{
				{TokenID: "yes", Outcome: "Yes", Price: "0.50"},
			}},
			{ConditionID: "active-2", Active: true, Tokens: []rest.TokenResponse{
				{TokenID: "yes", Outcome: "Yes", Price: "0.70"},
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rest.MarketsResponse{
			Markets: markets,
		})
	}))
	defer server.Close()

	logger := zap.NewNop()
	client := rest.NewClient(server.URL, logger)
	batcher := rest.NewBatchFetcher(client, 100, logger)

	ctx := context.Background()
	markets, err := batcher.FetchMarketBatch(ctx, []string{"active-1", "inactive-1", "active-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(markets) != 2 {
		t.Errorf("expected 2 active markets, got %d", len(markets))
	}

	for _, m := range markets {
		if m.ID == "inactive-1" {
			t.Error("inactive market should be filtered out")
		}
	}
}

func TestBatchFetcher_FiltersByRequestedIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ids := parseIDParams(r.URL.Query())
		allMarkets := map[string]rest.MarketResponse{
			"market-1": {ConditionID: "market-1", Active: true, Tokens: []rest.TokenResponse{
				{TokenID: "yes", Outcome: "Yes", Price: "0.60"},
			}},
			"market-2": {ConditionID: "market-2", Active: true, Tokens: []rest.TokenResponse{
				{TokenID: "yes", Outcome: "Yes", Price: "0.70"},
			}},
			"market-3": {ConditionID: "market-3", Active: true, Tokens: []rest.TokenResponse{
				{TokenID: "yes", Outcome: "Yes", Price: "0.80"},
			}},
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

	logger := zap.NewNop()
	client := rest.NewClient(server.URL, logger)
	batcher := rest.NewBatchFetcher(client, 100, logger)

	ctx := context.Background()
	markets, err := batcher.FetchMarketBatch(ctx, []string{"market-1", "market-3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(markets) != 2 {
		t.Errorf("expected 2 markets, got %d", len(markets))
	}

	for _, m := range markets {
		if m.ID == "market-2" {
			t.Error("market-2 should not be included (not in requested IDs)")
		}
	}
}

func TestBatchFetcher_APIError(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		markets := []rest.MarketResponse{
			{ConditionID: "market-1", Active: true, Tokens: []rest.TokenResponse{
				{TokenID: "yes", Outcome: "Yes", Price: "0.60"},
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rest.MarketsResponse{
			Markets: markets,
		})
	}))
	defer server.Close()

	logger := zap.NewNop()
	client := rest.NewClient(server.URL, logger)
	batcher := rest.NewBatchFetcher(client, 100, logger)

	ctx := context.Background()
	_, err := batcher.FetchMarketBatch(ctx, []string{"market-1"})
	if err == nil {
		t.Fatal("expected error from API failure")
	}
}

func TestBatchFetcher_ConvertMarketPrices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		markets := []rest.MarketResponse{
			{
				ConditionID: "market-1",
				Slug:        "test-market",
				Active:      true,
				Volume24h:   "1000.50",
				Liquidity:   "500.25",
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

	logger := zap.NewNop()
	client := rest.NewClient(server.URL, logger)
	batcher := rest.NewBatchFetcher(client, 100, logger)

	ctx := context.Background()
	markets, err := batcher.FetchMarketBatch(ctx, []string{"market-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(markets) != 1 {
		t.Fatalf("expected 1 market, got %d", len(markets))
	}

	m := markets[0]
	if m.ID != "market-1" {
		t.Errorf("ID = %q, want market-1", m.ID)
	}
	if m.Slug != "test-market" {
		t.Errorf("Slug = %q, want test-market", m.Slug)
	}
	if m.YESPrice.String() != "0.65" {
		t.Errorf("YESPrice = %v, want 0.65", m.YESPrice)
	}
	if m.NOPrice.String() != "0.30" {
		t.Errorf("NOPrice = %v, want 0.30", m.NOPrice)
	}
	if m.Volume24h.String() != "1000.50" {
		t.Errorf("Volume24h = %v, want 1000.50", m.Volume24h)
	}
	if m.LiquidityDepth.String() != "500.25" {
		t.Errorf("LiquidityDepth = %v, want 500.25", m.LiquidityDepth)
	}
}

func TestBatchFetcher_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rest.MarketsResponse{})
	}))
	defer server.Close()

	logger := zap.NewNop()
	client := rest.NewClient(server.URL, logger)
	batcher := rest.NewBatchFetcher(client, 100, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := batcher.FetchMarketBatch(ctx, []string{"market-1"})
	if err == nil {
		t.Fatal("expected error from context cancellation")
	}
}

func TestBatchFetcher_MaxBatchLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ids := parseIDParams(r.URL.Query())
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

	logger := zap.NewNop()
	client := rest.NewClient(server.URL, logger)
	batcher := rest.NewBatchFetcher(client, 50, logger)

	ids := make([]string, 200)
	for i := 0; i < 200; i++ {
		ids[i] = fmt.Sprintf("market-%d", i)
	}

	ctx := context.Background()
	markets, err := batcher.FetchMarketBatch(ctx, ids)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(markets) > 50 {
		t.Errorf("expected at most 50 markets (maxBatch), got %d", len(markets))
	}
}

// parseIDParams extracts repeated id query parameters from the URL.
func parseIDParams(q url.Values) []string {
	raw := q["id"]
	var ids []string
	for _, v := range raw {
		// Polymarket API may send comma-separated IDs in a single param
		for _, part := range strings.Split(v, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				ids = append(ids, part)
			}
		}
	}
	return ids
}
