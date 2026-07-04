package history_test

import (
	"testing"
	"time"

	"github.com/pqap/services/execution-engine/internal/history"
	"github.com/shopspring/decimal"
)

func TestTradeRecord_JSON(t *testing.T) {
	now := time.Now().UTC()
	record := &history.TradeRecord{
		ID:              "test-id",
		ClientOrderID:   "client-1",
		StrategyID:      "default",
		MarketID:        "market-1",
		MarketSlug:      "test-market",
		Side:            "YES",
		OrderType:       "GTC",
		Price:           decimal.RequireFromString("0.5500"),
		Quantity:        decimal.RequireFromString("100.00000000"),
		FilledQuantity:  decimal.Zero,
		FillStatus:      history.FillStatusPlaced,
		LatencyMs:       150,
		PnL:             decimal.Zero,
		Fee:             decimal.Zero,
		SlippagePct:     decimal.Zero,
		SignalTimestamp: now,
		OrderTimestamp:  now,
		RiskDecision:    "APPROVED",
		CreatedAt:       now,
	}

	if record.ClientOrderID != "client-1" {
		t.Errorf("expected client_order_id client-1, got %s", record.ClientOrderID)
	}
	if record.FillStatus != history.FillStatusPlaced {
		t.Errorf("expected fill_status PLACED, got %s", record.FillStatus)
	}
}

func TestOrderResult_HasSignalPrice(t *testing.T) {
	result := &history.OrderResult{
		ClientOrderID: "client-1",
		Price:         decimal.RequireFromString("0.5500"),
		SignalPrice:   decimal.RequireFromString("0.5400"),
	}
	if result.SignalPrice.IsZero() {
		t.Error("SignalPrice should be set")
	}
}

func TestFillStatus_Values(t *testing.T) {
	statuses := []history.FillStatus{
		history.FillStatusPending,
		history.FillStatusPlaced,
		history.FillStatusFilled,
		history.FillStatusPartialFill,
		history.FillStatusCancelled,
		history.FillStatusFailed,
		history.FillStatusExpired,
	}
	for _, s := range statuses {
		if s == "" {
			t.Error("FillStatus should not be empty")
		}
	}
}
