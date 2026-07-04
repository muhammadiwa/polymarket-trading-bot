package scorer_test

import (
	"context"
	"testing"

	"github.com/pqap/services/arb-engine/internal/ports"
	"github.com/pqap/services/arb-engine/internal/scorer"
	"github.com/shopspring/decimal"
)

type mockScorerLogger struct{}

func (m *mockScorerLogger) Log(_ context.Context, _ ports.Opportunity) error { return nil }
func (m *mockScorerLogger) GetHistoricalFillRate(_ context.Context, _ string, _ int) (decimal.Decimal, int, error) {
	return decimal.NewFromFloat(0.5), 0, nil
}
func (m *mockScorerLogger) Close() error { return nil }

func TestCalculateScore(t *testing.T) {
	tests := []struct {
		name            string
		spread          string
		liquidity       string
		fillProbability string
		want            string
	}{
		{
			name:            "all positive",
			spread:          "0.05",
			liquidity:       "0.8",
			fillProbability: "0.7",
			want:            "0.028",
		},
		{
			name:            "zero spread",
			spread:          "0.00",
			liquidity:       "0.8",
			fillProbability: "0.7",
			want:            "0.000",
		},
		{
			name:            "zero liquidity",
			spread:          "0.05",
			liquidity:       "0.0",
			fillProbability: "0.7",
			want:            "0.000",
		},
		{
			name:            "zero fill probability",
			spread:          "0.05",
			liquidity:       "0.8",
			fillProbability: "0.0",
			want:            "0.000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spread := decimal.RequireFromString(tt.spread)
			liquidity := decimal.RequireFromString(tt.liquidity)
			fillProb := decimal.RequireFromString(tt.fillProbability)
			want := decimal.RequireFromString(tt.want)

			got := scorer.CalculateScore(spread, liquidity, fillProb)
			if !got.Equal(want) {
				t.Errorf("CalculateScore(%s, %s, %s) = %s, want %s", tt.spread, tt.liquidity, tt.fillProbability, got, want)
			}
		})
	}
}

func TestScoreDeterminism(t *testing.T) {
	spread := decimal.RequireFromString("0.05")
	liquidity := decimal.RequireFromString("0.8")
	fillProb := decimal.RequireFromString("0.7")

	for i := 0; i < 100; i++ {
		got := scorer.CalculateScore(spread, liquidity, fillProb)
		want := decimal.RequireFromString("0.028")
		if !got.Equal(want) {
			t.Fatalf("iteration %d: CalculateScore = %s, want %s", i, got, want)
		}
	}
}
