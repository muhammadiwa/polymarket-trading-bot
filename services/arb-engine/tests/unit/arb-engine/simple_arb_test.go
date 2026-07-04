package detector_test

import (
	"testing"
	"time"

	"github.com/pqap/services/arb-engine/internal/detector"
	"github.com/pqap/services/arb-engine/internal/ports"
	"github.com/shopspring/decimal"
)

func TestCalculateSpread(t *testing.T) {
	tests := []struct {
		name     string
		yesPrice string
		noPrice  string
		want     string
	}{
		{
			name:     "opportunity exists",
			yesPrice: "0.55",
			noPrice:  "0.40",
			want:     "0.05",
		},
		{
			name:     "no opportunity at parity",
			yesPrice: "0.55",
			noPrice:  "0.45",
			want:     "0.00",
		},
		{
			name:     "negative spread",
			yesPrice: "0.60",
			noPrice:  "0.50",
			want:     "-0.10",
		},
		{
			name:     "max spread",
			yesPrice: "0.00",
			noPrice:  "0.00",
			want:     "1.00",
		},
		{
			name:     "impossible prices",
			yesPrice: "1.00",
			noPrice:  "1.00",
			want:     "-1.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yes := decimal.RequireFromString(tt.yesPrice)
			no := decimal.RequireFromString(tt.noPrice)
			got := detector.CalculateSpread(yes, no)
			want := decimal.RequireFromString(tt.want)
			if !got.Equal(want) {
				t.Errorf("CalculateSpread(%s, %s) = %s, want %s", tt.yesPrice, tt.noPrice, got, want)
			}
		})
	}
}

func TestDetect(t *testing.T) {
	d := detector.NewSimpleArbDetector("0.01")

	tests := []struct {
		name        string
		event       ports.MarketPriceUpdated
		wantOpp     bool
		wantSpread  string
	}{
		{
			name: "opportunity detected",
			event: ports.MarketPriceUpdated{
				MarketID: "market-1",
				YESPrice: decimal.RequireFromString("0.55"),
				NOPrice:  decimal.RequireFromString("0.40"),
				IsStale:  false,
			},
			wantOpp:    true,
			wantSpread: "0.05",
		},
		{
			name: "no opportunity - spread too small",
			event: ports.MarketPriceUpdated{
				MarketID: "market-2",
				YESPrice: decimal.RequireFromString("0.50"),
				NOPrice:  decimal.RequireFromString("0.495"),
				IsStale:  false,
			},
			wantOpp: false,
		},
		{
			name: "no opportunity - negative spread",
			event: ports.MarketPriceUpdated{
				MarketID: "market-3",
				YESPrice: decimal.RequireFromString("0.60"),
				NOPrice:  decimal.RequireFromString("0.50"),
				IsStale:  false,
			},
			wantOpp: false,
		},
		{
			name: "exact threshold - no opportunity",
			event: ports.MarketPriceUpdated{
				MarketID: "market-4",
				YESPrice: decimal.RequireFromString("0.50"),
				NOPrice:  decimal.RequireFromString("0.49"),
				IsStale:  false,
			},
			wantOpp: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opp, latencyMs := d.Detect(tt.event)

			if tt.wantOpp {
				if opp == nil {
					t.Fatal("expected opportunity, got nil")
				}
				if opp.MarketID != tt.event.MarketID {
					t.Errorf("MarketID = %s, want %s", opp.MarketID, tt.event.MarketID)
				}
				wantSpread := decimal.RequireFromString(tt.wantSpread)
				if !opp.Spread.Equal(wantSpread) {
					t.Errorf("Spread = %s, want %s", opp.Spread, wantSpread)
				}
				if opp.ID == "" {
					t.Error("expected non-empty ID")
				}
				if opp.DetectedAt.IsZero() {
					t.Error("expected non-zero DetectedAt")
				}
			} else {
				if opp != nil {
					t.Errorf("expected nil, got opportunity with spread %s", opp.Spread)
				}
			}

			if latencyMs < 0 {
				t.Errorf("latencyMs = %d, want >= 0", latencyMs)
			}
		})
	}
}

func TestDetectLatency(t *testing.T) {
	d := detector.NewSimpleArbDetector("0.01")
	event := ports.MarketPriceUpdated{
		MarketID: "market-latency",
		YESPrice: decimal.RequireFromString("0.55"),
		NOPrice:  decimal.RequireFromString("0.40"),
		IsStale:  false,
	}

	_, latencyMs := d.Detect(event)

	if latencyMs > 100 {
		t.Errorf("detection latency %dms exceeds 100ms target", latencyMs)
	}
}
