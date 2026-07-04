package executor_test

import (
	"testing"

	"github.com/pqap/services/execution-engine/internal/executor"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func TestPartialTracker_RecordFill(t *testing.T) {
	log, _ := zap.NewDevelopment()
	tracker := executor.NewPartialTracker(log)

	record := executor.PartialFillRecord{
		PairID:        "pair-1",
		Leg:           "YES",
		FilledQty:     decimal.NewFromInt(50),
		RemainingQty:  decimal.NewFromInt(50),
		FillPrice:     decimal.NewFromFloat(0.55),
		OrderID:       "order-1",
		ClientOrderID: "client-1",
		MarketID:      "market-1",
		StrategyID:    "test-strategy",
	}

	tracker.RecordPartialFill(record)

	fills := tracker.GetPartialFills("pair-1")
	if len(fills) != 1 {
		t.Fatalf("expected 1 partial fill, got %d", len(fills))
	}

	if fills[0].Leg != "YES" {
		t.Errorf("leg = %s, want YES", fills[0].Leg)
	}

	if !fills[0].FilledQty.Equal(decimal.NewFromInt(50)) {
		t.Errorf("filled qty = %s, want 50", fills[0].FilledQty.String())
	}
}

func TestPartialTracker_MultipleFills(t *testing.T) {
	log, _ := zap.NewDevelopment()
	tracker := executor.NewPartialTracker(log)

	tracker.RecordPartialFill(executor.PartialFillRecord{
		PairID:    "pair-1",
		Leg:       "YES",
		FilledQty: decimal.NewFromInt(30),
		FillPrice: decimal.NewFromFloat(0.55),
	})

	tracker.RecordPartialFill(executor.PartialFillRecord{
		PairID:    "pair-1",
		Leg:       "NO",
		FilledQty: decimal.NewFromInt(20),
		FillPrice: decimal.NewFromFloat(0.40),
	})

	fills := tracker.GetPartialFills("pair-1")
	if len(fills) != 2 {
		t.Fatalf("expected 2 partial fills, got %d", len(fills))
	}
}

func TestPartialTracker_Reconcile(t *testing.T) {
	log, _ := zap.NewDevelopment()
	tracker := executor.NewPartialTracker(log)

	tracker.RecordPartialFill(executor.PartialFillRecord{
		PairID:    "pair-1",
		Leg:       "YES",
		FilledQty: decimal.NewFromInt(50),
		FillPrice: decimal.NewFromFloat(0.55),
	})

	if !tracker.HasPartialFills("pair-1") {
		t.Error("expected partial fills to exist")
	}

	tracker.Reconcile("pair-1")

	if tracker.HasPartialFills("pair-1") {
		t.Error("expected partial fills to be reconciled")
	}

	fills := tracker.GetPartialFills("pair-1")
	if len(fills) != 0 {
		t.Errorf("expected 0 fills after reconcile, got %d", len(fills))
	}
}

func TestPartialTracker_HasPartialFills(t *testing.T) {
	log, _ := zap.NewDevelopment()
	tracker := executor.NewPartialTracker(log)

	if tracker.HasPartialFills("pair-1") {
		t.Error("expected no partial fills for non-existent pair")
	}

	tracker.RecordPartialFill(executor.PartialFillRecord{
		PairID:    "pair-1",
		Leg:       "YES",
		FilledQty: decimal.NewFromInt(50),
		FillPrice: decimal.NewFromFloat(0.55),
	})

	if !tracker.HasPartialFills("pair-1") {
		t.Error("expected partial fills to exist")
	}
}
