package logger_test

import (
	"context"
	"testing"
	"time"

	"github.com/pqap/services/execution-engine/internal/logger"
	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type mockTradeRepo struct {
	records []logger.TradeRecord
}

func (m *mockTradeRepo) InsertTrade(ctx context.Context, record logger.TradeRecord) error {
	m.records = append(m.records, record)
	return nil
}

func (m *mockTradeRepo) Close() error {
	return nil
}

func TestOrderLogger_LogOrder(t *testing.T) {
	mockRepo := &mockTradeRepo{}
	log, _ := zap.NewDevelopment()
	orderLogger := logger.NewOrderLogger(mockRepo, log)

	order := &ports.Order{
		ID:            "order-1",
		ClientOrderID: "client-1",
		OpportunityID: "opp-1",
		MarketID:      "market-1",
		Side:          "BUY",
		Price:         decimal.RequireFromString("0.50"),
		Size:          decimal.NewFromInt(100),
		FilledQty:     decimal.NewFromInt(100),
		Status:        ports.OrderStatusFilled,
		StrategyID:    "default",
		LatencyMs:     150,
		RiskCheckResult: "ALLOW",
		SlippageCheck: "PASS",
		PlacedAt:      time.Now().UTC(),
	}

	err := orderLogger.LogOrder(context.Background(), order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockRepo.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(mockRepo.records))
	}

	record := mockRepo.records[0]
	if record.OrderID != "order-1" {
		t.Errorf("OrderID = %s, want order-1", record.OrderID)
	}
	if record.ClientOrderID != "client-1" {
		t.Errorf("ClientOrderID = %s, want client-1", record.ClientOrderID)
	}
	if record.MarketID != "market-1" {
		t.Errorf("MarketID = %s, want market-1", record.MarketID)
	}
	if record.Side != "BUY" {
		t.Errorf("Side = %s, want BUY", record.Side)
	}
	if !record.Price.Equal(decimal.RequireFromString("0.50")) {
		t.Errorf("Price = %s, want 0.50", record.Price.String())
	}
	if record.FillStatus != "FILLED" {
		t.Errorf("FillStatus = %s, want FILLED", record.FillStatus)
	}
	if record.LatencyMs != 150 {
		t.Errorf("LatencyMs = %d, want 150", record.LatencyMs)
	}
}

func TestOrderLogger_WithPnL(t *testing.T) {
	mockRepo := &mockTradeRepo{}
	log, _ := zap.NewDevelopment()
	orderLogger := logger.NewOrderLogger(mockRepo, log)

	pnl := decimal.RequireFromString("10.50")
	order := &ports.Order{
		ID:            "order-2",
		ClientOrderID: "client-2",
		OpportunityID: "opp-2",
		MarketID:      "market-2",
		Side:          "SELL",
		Price:         decimal.RequireFromString("0.60"),
		Size:          decimal.NewFromInt(50),
		FilledQty:     decimal.NewFromInt(50),
		Status:        ports.OrderStatusFilled,
		StrategyID:    "default",
		LatencyMs:     200,
		RiskCheckResult: "ALLOW",
		SlippageCheck: "PASS",
		PlacedAt:      time.Now().UTC(),
	}

	order.PnL = &pnl

	err := orderLogger.LogOrder(context.Background(), order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockRepo.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(mockRepo.records))
	}

	record := mockRepo.records[0]
	if record.PnL == nil {
		t.Fatal("expected PnL to be set")
	}
	if !record.PnL.Equal(pnl) {
		t.Errorf("PnL = %s, want 10.50", record.PnL.String())
	}
}

func TestOrderLogger_WithAccountID(t *testing.T) {
	mockRepo := &mockTradeRepo{}
	log, _ := zap.NewDevelopment()
	orderLogger := logger.NewOrderLogger(mockRepo, log)

	accountID := "account-1"
	order := &ports.Order{
		ID:            "order-3",
		ClientOrderID: "client-3",
		OpportunityID: "opp-3",
		MarketID:      "market-3",
		Side:          "BUY",
		Price:         decimal.RequireFromString("0.45"),
		Size:          decimal.NewFromInt(200),
		FilledQty:     decimal.Zero,
		Status:        ports.OrderStatusPlaced,
		StrategyID:    "strategy-1",
		LatencyMs:     100,
		RiskCheckResult: "ALLOW",
		SlippageCheck: "PASS",
		AccountID:     &accountID,
		PlacedAt:      time.Now().UTC(),
	}

	err := orderLogger.LogOrder(context.Background(), order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	record := mockRepo.records[0]
	if record.AccountID == nil {
		t.Fatal("expected AccountID to be set")
	}
	if *record.AccountID != "account-1" {
		t.Errorf("AccountID = %s, want account-1", *record.AccountID)
	}
}
