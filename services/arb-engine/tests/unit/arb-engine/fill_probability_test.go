package scorer_test

import (
	"context"
	"testing"

	"github.com/pqap/services/arb-engine/internal/ports"
	"github.com/pqap/services/arb-engine/internal/scorer"
	"github.com/shopspring/decimal"
)

type mockOppLogger struct {
	fillRate  float64
	sampleCnt int
	err       error
}

func (m *mockOppLogger) Log(_ context.Context, _ ports.Opportunity) error { return nil }
func (m *mockOppLogger) GetHistoricalFillRate(_ context.Context, _ string, _ int) (decimal.Decimal, int, error) {
	if m.err != nil {
		return decimal.Zero, 0, m.err
	}
	return decimal.NewFromFloat(m.fillRate), m.sampleCnt, nil
}
func (m *mockOppLogger) Close() error { return nil }

func TestEstimateFromOrderbook(t *testing.T) {
	tests := []struct {
		name  string
		depth string
		want  string
	}{
		{
			name:  "deep orderbook",
			depth: "5000",
			want:  "1.0",
		},
		{
			name:  "shallow orderbook",
			depth: "100",
			want:  "0.1",
		},
		{
			name:  "no orderbook data",
			depth: "0",
			want:  "0.5",
		},
		{
			name:  "exact required depth",
			depth: "1000",
			want:  "1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &mockOppLogger{fillRate: 0.5, sampleCnt: 0}
			estimator := scorer.NewFillProbabilityEstimator(logger, 0.7, 0.3, 1000.0)
			depth := decimal.RequireFromString(tt.depth)
			ctx := context.Background()
			got := estimator.Estimate(ctx, depth, "test-market")
			want := decimal.RequireFromString(tt.want)
			if !got.Equal(want) {
				t.Errorf("Estimate(%s) = %s, want %s", tt.depth, got, want)
			}
		})
	}
}

func TestEstimateWithHistoricalCalibration(t *testing.T) {
	logger := &mockOppLogger{fillRate: 0.8, sampleCnt: 150}
	estimator := scorer.NewFillProbabilityEstimator(logger, 0.7, 0.3, 1000.0)

	depth := decimal.RequireFromString("1000")
	ctx := context.Background()
	got := estimator.Estimate(ctx, depth, "test-market")

	// orderbook = min(1000/1000, 1.0) = 1.0
	// blended = 0.7 * 1.0 + 0.3 * 0.8 = 0.7 + 0.24 = 0.94
	want := decimal.RequireFromString("0.94")
	if !got.Equal(want) {
		t.Errorf("Estimate with historical = %s, want %s", got, want)
	}
}

func TestEstimateInsufficientHistory(t *testing.T) {
	logger := &mockOppLogger{fillRate: 0.8, sampleCnt: 50}
	estimator := scorer.NewFillProbabilityEstimator(logger, 0.7, 0.3, 1000.0)

	depth := decimal.RequireFromString("500")
	ctx := context.Background()
	got := estimator.Estimate(ctx, depth, "test-market")

	// Should use orderbook only since sampleCnt < 100
	want := decimal.RequireFromString("0.5")
	if !got.Equal(want) {
		t.Errorf("Estimate with insufficient history = %s, want %s", got, want)
	}
}
