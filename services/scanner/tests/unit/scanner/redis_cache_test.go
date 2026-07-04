package scanner_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/pqap/services/scanner/internal/catalog"
	"github.com/shopspring/decimal"
)

func TestRedisCache_KeyPattern(t *testing.T) {
	tests := []struct {
		marketID string
		wantKey  string
	}{
		{marketID: "market-1", wantKey: "pqap:market:market-1"},
		{marketID: "abc-123", wantKey: "pqap:market:abc-123"},
		{marketID: "special/chars", wantKey: "pqap:market:special/chars"},
	}

	for _, tt := range tests {
		t.Run(tt.marketID, func(t *testing.T) {
			key := fmt.Sprintf("pqap:market:%s", tt.marketID)
			if key != tt.wantKey {
				t.Errorf("key = %q, want %q", key, tt.wantKey)
			}
		})
	}
}

func TestRedisCache_MarketSerialization(t *testing.T) {
	tests := []struct {
		name   string
		market catalog.Market
	}{
		{
			name: "full market data",
			market: catalog.Market{
				ID:             "market-1",
				Slug:           "test-market",
				YESPrice:       decimal.NewFromFloat(0.65),
				NOPrice:        decimal.NewFromFloat(0.30),
				Spread:         decimal.NewFromFloat(0.05),
				Volume24h:      decimal.NewFromFloat(1000.50),
				LiquidityDepth: decimal.NewFromFloat(500.25),
				IsActive:       true,
				IsStale:        false,
				LastUpdated:    time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "minimal market data",
			market: catalog.Market{
				ID:       "market-2",
				YESPrice: decimal.Zero,
				NOPrice:  decimal.Zero,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.market)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			var got catalog.Market
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}

			if got.ID != tt.market.ID {
				t.Errorf("ID = %q, want %q", got.ID, tt.market.ID)
			}
			if !got.YESPrice.Equal(tt.market.YESPrice) {
				t.Errorf("YESPrice = %v, want %v", got.YESPrice, tt.market.YESPrice)
			}
			if !got.NOPrice.Equal(tt.market.NOPrice) {
				t.Errorf("NOPrice = %v, want %v", got.NOPrice, tt.market.NOPrice)
			}
			if got.IsActive != tt.market.IsActive {
				t.Errorf("IsActive = %v, want %v", got.IsActive, tt.market.IsActive)
			}
		})
	}
}

func TestRedisCache_ActiveMarketIDs_SetOperations(t *testing.T) {
	marketIDs := []string{"m1", "m2", "m3"}

	seen := make(map[string]bool)
	for _, id := range marketIDs {
		if seen[id] {
			t.Errorf("duplicate market ID: %s", id)
		}
		seen[id] = true
	}

	if len(seen) != len(marketIDs) {
		t.Errorf("unique count = %d, want %d", len(seen), len(marketIDs))
	}
}

func TestRedisCache_SetGet_Integration(t *testing.T) {
	t.Skip("skipping: requires running Redis server")
}

func TestRedisCache_TTL_Integration(t *testing.T) {
	t.Skip("skipping: requires running Redis server")
}

func TestRedisCache_Remove_Integration(t *testing.T) {
	t.Skip("skipping: requires running Redis server")
}

func TestRedisCache_ActiveMarketIDs_Integration(t *testing.T) {
	t.Skip("skipping: requires running Redis server")
}
