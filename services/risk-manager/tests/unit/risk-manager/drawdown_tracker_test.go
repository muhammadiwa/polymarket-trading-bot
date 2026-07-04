package riskmanager

import (
	"testing"

	"github.com/pqap/services/risk-manager/internal/pitboss"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func newTestDrawdownTracker(capital float64, limitPct, warningThreshold float64) *pitboss.DrawdownTracker {
	cap := decimal.NewFromFloat(capital)
	logger, _ := zap.NewDevelopment()
	return pitboss.NewDrawdownTracker(cap, limitPct, warningThreshold, nil, logger)
}

func TestDrawdownTracker_InitialZeroDrawdown(t *testing.T) {
	dt := newTestDrawdownTracker(10000, 0.10, 0.80)

	peak, current, drawdown, limit := dt.GetState()

	if !peak.Equal(decimal.NewFromFloat(10000)) {
		t.Errorf("expected peak 10000, got %s", peak)
	}
	if !current.Equal(decimal.NewFromFloat(10000)) {
		t.Errorf("expected current 10000, got %s", current)
	}
	if !drawdown.IsZero() {
		t.Errorf("expected drawdown 0, got %s", drawdown)
	}
	if !limit.Equal(decimal.NewFromFloat(0.10)) {
		t.Errorf("expected limit 0.10, got %s", limit)
	}
}

func TestDrawdownTracker_DrawdownCalculation(t *testing.T) {
	dt := newTestDrawdownTracker(10000, 0.10, 0.80)

	dt.UpdateEquity(decimal.NewFromFloat(9000))

	_, current, drawdown, _ := dt.GetState()
	if !current.Equal(decimal.NewFromFloat(9000)) {
		t.Errorf("expected current 9000, got %s", current)
	}
	expectedDrawdown := decimal.NewFromFloat(0.10)
	if !drawdown.Equal(expectedDrawdown) {
		t.Errorf("expected drawdown 0.10, got %s", drawdown)
	}
}

func TestDrawdownTracker_PeakEquityUpdates(t *testing.T) {
	dt := newTestDrawdownTracker(10000, 0.10, 0.80)

	dt.UpdateEquity(decimal.NewFromFloat(12000))

	peak, current, drawdown, _ := dt.GetState()
	if !peak.Equal(decimal.NewFromFloat(12000)) {
		t.Errorf("expected peak 12000, got %s", peak)
	}
	if !current.Equal(decimal.NewFromFloat(12000)) {
		t.Errorf("expected current 12000, got %s", current)
	}
	if !drawdown.IsZero() {
		t.Errorf("expected drawdown 0 at new peak, got %s", drawdown)
	}
}

func TestDrawdownTracker_DrawdownNeverNegative(t *testing.T) {
	dt := newTestDrawdownTracker(10000, 0.10, 0.80)

	dt.UpdateEquity(decimal.NewFromFloat(15000))

	_, _, drawdown, _ := dt.GetState()
	if drawdown.IsNegative() {
		t.Errorf("drawdown should never be negative, got %s", drawdown)
	}
	if !drawdown.IsZero() {
		t.Errorf("drawdown should be zero when equity above peak, got %s", drawdown)
	}
}

func TestDrawdownTracker_SetState(t *testing.T) {
	dt := newTestDrawdownTracker(10000, 0.10, 0.80)

	peak := decimal.NewFromFloat(50000)
	current := decimal.NewFromFloat(45000)
	dt.SetState(peak, current)

	gotPeak, gotCurrent, drawdown, _ := dt.GetState()
	if !gotPeak.Equal(peak) {
		t.Errorf("expected peak %s, got %s", peak, gotPeak)
	}
	if !gotCurrent.Equal(current) {
		t.Errorf("expected current %s, got %s", current, gotCurrent)
	}
	expectedDrawdown := decimal.NewFromFloat(0.10)
	if !drawdown.Equal(expectedDrawdown) {
		t.Errorf("expected drawdown 0.10, got %s", drawdown)
	}
}

func TestDrawdownTracker_UpdateCapital(t *testing.T) {
	dt := newTestDrawdownTracker(10000, 0.10, 0.80)

	dt.UpdateCapital(decimal.NewFromFloat(9500))

	_, current, drawdown, _ := dt.GetState()
	if !current.Equal(decimal.NewFromFloat(9500)) {
		t.Errorf("expected current 9500, got %s", current)
	}
	expectedDrawdown := decimal.NewFromFloat(0.05)
	if !drawdown.Equal(expectedDrawdown) {
		t.Errorf("expected drawdown 0.05, got %s", drawdown)
	}
}

func TestDrawdownTracker_MultipleUpdates(t *testing.T) {
	dt := newTestDrawdownTracker(10000, 0.10, 0.80)

	dt.UpdateEquity(decimal.NewFromFloat(9000))
	dt.UpdateEquity(decimal.NewFromFloat(8000))
	dt.UpdateEquity(decimal.NewFromFloat(8500))

	peak, current, drawdown, _ := dt.GetState()
	if !peak.Equal(decimal.NewFromFloat(10000)) {
		t.Errorf("expected peak to remain 10000, got %s", peak)
	}
	if !current.Equal(decimal.NewFromFloat(8500)) {
		t.Errorf("expected current 8500, got %s", current)
	}
	expectedDrawdown := decimal.NewFromFloat(0.15)
	if !drawdown.Equal(expectedDrawdown) {
		t.Errorf("expected drawdown 0.15, got %s", drawdown)
	}
}

func TestDrawdownTracker_PeakResetAfterRecovery(t *testing.T) {
	dt := newTestDrawdownTracker(10000, 0.10, 0.80)

	dt.UpdateEquity(decimal.NewFromFloat(8000))
	dt.UpdateEquity(decimal.NewFromFloat(11000))

	peak, _, drawdown, _ := dt.GetState()
	if !peak.Equal(decimal.NewFromFloat(11000)) {
		t.Errorf("expected new peak 11000, got %s", peak)
	}
	if !drawdown.IsZero() {
		t.Errorf("expected drawdown 0 at new peak, got %s", drawdown)
	}
}

func TestDrawdownTracker_ZeroPeakEquity(t *testing.T) {
	cap := decimal.Zero
	logger, _ := zap.NewDevelopment()
	dt := pitboss.NewDrawdownTracker(cap, 0.10, 0.80, nil, logger)

	_, _, drawdown, _ := dt.GetState()
	if !drawdown.IsZero() {
		t.Errorf("expected drawdown 0 with zero peak, got %s", drawdown)
	}
}
