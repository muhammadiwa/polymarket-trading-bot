package riskmanager

import (
	"testing"

	"github.com/pqap/services/risk-manager/internal/risk"
	"go.uber.org/zap"
)

func newTestCorrelationTracker() *risk.CorrelationTracker {
	logger, _ := zap.NewDevelopment()
	engine := risk.NewCorrelationEngine(0.7, 1000000000, 3, nil, nil, logger)
	engine.SetCategoryGroup("election", []string{"market_1", "market_2", "market_3", "market_4"})
	engine.UpdateGroups()
	return risk.NewCorrelationTracker(engine, 3, nil, logger)
}

func TestCorrelationTracker_AllowWhenUnderLimit(t *testing.T) {
	tracker := newTestCorrelationTracker()
	tracker.SetOpenPositions([]string{"market_1", "market_2"})

	result := tracker.CheckMarket("market_3")
	if !result.Allowed {
		t.Errorf("expected allowed, got denied: %s", result.Reason)
	}
}

func TestCorrelationTracker_DenyWhenAtLimit(t *testing.T) {
	tracker := newTestCorrelationTracker()
	tracker.SetOpenPositions([]string{"market_1", "market_2", "market_3"})

	result := tracker.CheckMarket("market_4")
	if result.Allowed {
		t.Error("expected denied when correlated positions at limit")
	}
	if result.Reason != "correlation_limit_exceeded" {
		t.Errorf("expected reason 'correlation_limit_exceeded', got %q", result.Reason)
	}
}

func TestCorrelationTracker_DenyWhenOverLimit(t *testing.T) {
	tracker := newTestCorrelationTracker()
	tracker.SetOpenPositions([]string{"market_1", "market_2", "market_3"})

	result := tracker.CheckMarket("market_4")
	if result.Allowed {
		t.Error("expected denied when correlated positions exceed limit")
	}
	if len(result.CorrelatedWith) == 0 {
		t.Error("expected non-empty correlated_with list")
	}
}

func TestCorrelationTracker_AllowNonCorrelated(t *testing.T) {
	tracker := newTestCorrelationTracker()
	tracker.SetOpenPositions([]string{"market_1", "market_2", "market_3"})

	result := tracker.CheckMarket("market_other")
	if !result.Allowed {
		t.Errorf("expected allowed for non-correlated market, got denied: %s", result.Reason)
	}
}

func TestCorrelationTracker_AddRemovePosition(t *testing.T) {
	tracker := newTestCorrelationTracker()

	tracker.AddOpenPosition("market_1")
	tracker.AddOpenPosition("market_2")

	result := tracker.CheckMarket("market_3")
	if !result.Allowed {
		t.Error("expected allowed with 2 open positions")
	}

	tracker.RemoveOpenPosition("market_1")

	result = tracker.CheckMarket("market_3")
	if !result.Allowed {
		t.Error("expected allowed after removing position")
	}
}

func TestCorrelationTracker_EmptyPositions(t *testing.T) {
	tracker := newTestCorrelationTracker()

	result := tracker.CheckMarket("market_1")
	if !result.Allowed {
		t.Error("expected allowed with no open positions")
	}
}

func TestCorrelationTracker_CorrelatedWithList(t *testing.T) {
	tracker := newTestCorrelationTracker()
	tracker.SetOpenPositions([]string{"market_1", "market_2", "market_3"})

	result := tracker.CheckMarket("market_4")
	if result.Allowed {
		t.Error("expected denied")
	}

	for _, id := range result.CorrelatedWith {
		if id != "market_1" && id != "market_2" && id != "market_3" {
			t.Errorf("unexpected correlated market: %s", id)
		}
	}
}
