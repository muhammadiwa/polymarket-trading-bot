package filter_test

import (
	"testing"

	"github.com/pqap/services/arb-engine/internal/filter"
	"github.com/pqap/services/arb-engine/internal/ports"
	"github.com/shopspring/decimal"
)

func TestFilter(t *testing.T) {
	tests := []struct {
		name          string
		score         string
		threshold     string
		wantEmitted   bool
		wantReason    string
	}{
		{
			name:        "above threshold",
			score:       "0.05",
			threshold:   "0.01",
			wantEmitted: true,
			wantReason:  "",
		},
		{
			name:        "below threshold",
			score:       "0.005",
			threshold:   "0.01",
			wantEmitted: false,
			wantReason:  "below_threshold",
		},
		{
			name:        "equal to threshold",
			score:       "0.01",
			threshold:   "0.01",
			wantEmitted: true,
			wantReason:  "",
		},
		{
			name:        "zero score",
			score:       "0.00",
			threshold:   "0.01",
			wantEmitted: false,
			wantReason:  "below_threshold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := filter.NewThresholdFilter(tt.threshold)
			opp := &ports.Opportunity{
				Score: decimal.RequireFromString(tt.score),
			}

			emitted := f.Filter(opp)

			if emitted != tt.wantEmitted {
				t.Errorf("Filter() = %v, want %v", emitted, tt.wantEmitted)
			}
			if opp.FilterReason != tt.wantReason {
				t.Errorf("FilterReason = %q, want %q", opp.FilterReason, tt.wantReason)
			}
		})
	}
}

func TestUpdateThreshold(t *testing.T) {
	f := filter.NewThresholdFilter("0.01")
	opp := &ports.Opportunity{
		Score: decimal.RequireFromString("0.02"),
	}

	if !f.Filter(opp) {
		t.Error("expected opportunity to pass filter with threshold 0.01")
	}

	err := f.UpdateThreshold("0.05")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	opp.Score = decimal.RequireFromString("0.02")
	if f.Filter(opp) {
		t.Error("expected opportunity to fail filter with threshold 0.05")
	}
}

func TestUpdateThresholdNegative(t *testing.T) {
	f := filter.NewThresholdFilter("0.01")
	err := f.UpdateThreshold("-0.05")
	if err == nil {
		t.Error("expected error for negative threshold")
	}
}
