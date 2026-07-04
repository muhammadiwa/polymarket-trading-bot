package filter

import (
	"fmt"
	"sync"

	"github.com/pqap/services/arb-engine/internal/ports"
	"github.com/shopspring/decimal"
)

type ThresholdFilter struct {
	mu             sync.RWMutex
	scoreThreshold decimal.Decimal
}

func NewThresholdFilter(scoreThreshold string) *ThresholdFilter {
	th, err := decimal.NewFromString(scoreThreshold)
	if err != nil {
		th = decimal.NewFromFloat(0.01)
	}
	return &ThresholdFilter{
		scoreThreshold: th,
	}
}

func (f *ThresholdFilter) Filter(opp *ports.Opportunity) bool {
	f.mu.RLock()
	threshold := f.scoreThreshold
	f.mu.RUnlock()

	if opp.Score.GreaterThanOrEqual(threshold) {
		opp.FilterReason = ""
		return true
	}

	opp.FilterReason = "below_threshold"
	return false
}

func (f *ThresholdFilter) UpdateThreshold(threshold string) error {
	th, err := decimal.NewFromString(threshold)
	if err != nil {
		return fmt.Errorf("invalid threshold value %q: %w", threshold, err)
	}
	if th.IsNegative() {
		return fmt.Errorf("threshold must be non-negative, got %s", threshold)
	}

	f.mu.Lock()
	f.scoreThreshold = th
	f.mu.Unlock()
	return nil
}
