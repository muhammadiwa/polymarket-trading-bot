package history_test

import (
	"context"
	"testing"
	"time"

	"github.com/pqap/services/execution-engine/internal/history"
	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type mockTradeRepo struct {
	records []*history.TradeRecord
	insertErr error
}

func (m *mockTradeRepo) Insert(ctx context.Context, record *history.TradeRecord) error {
	if m.insertErr != nil {
		return m.insertErr
	}
	m.records = append(m.records, record)
	return nil
}

func (m *mockTradeRepo) GetByClientOrderID(ctx context.Context, clientOrderID string) (*history.TradeRecord, error) {
	for _, r := range m.records {
		if r.ClientOrderID == clientOrderID {
			return r, nil
		}
	}
	return nil, nil
}

type mockTradePublisher struct {
	events []ports.TradeRecorded
	pubErr error
}

func (m *mockTradePublisher) PublishTradeRecorded(ctx context.Context, event ports.TradeRecorded) error {
	if m.pubErr != nil {
		return m.pubErr
	}
	m.events = append(m.events, event)
	return nil
}

func newTestHandler(repo *mockTradeRepo, pub *mockTradePublisher) *history.Handler {
	log, _ := zap.NewDevelopment()
	return history.NewHandler(repo, pub, log)
}

func TestHandleOrderResult_PLACED(t *testing.T) {
	repo := &mockTradeRepo{}
	pub := &mockTradePublisher{}
	handler := newTestHandler(repo, pub)

	now := time.Now().UTC()
	result := &history.OrderResult{
		ClientOrderID:   "client-1",
		OrderID:         "order-1",
		OpportunityID:   "opp-1",
		MarketID:        "market-1",
		MarketSlug:      "test-market",
		Side:            "YES",
		OrderType:       "GTC",
		Price:           decimal.RequireFromString("0.5500"),
		Quantity:        decimal.RequireFromString("100.00000000"),
		FilledQuantity:  decimal.Zero,
		FillStatus:      history.FillStatusPlaced,
		LatencyMs:       150,
		Fee:             decimal.Zero,
		SignalTimestamp: now,
		OrderTimestamp:  now,
		RiskDecision:    "APPROVED",
		StrategyID:      "default",
	}

	err := handler.HandleOrderResult(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(repo.records))
	}

	record := repo.records[0]
	if record.ClientOrderID != "client-1" {
		t.Errorf("client_order_id = %s, want client-1", record.ClientOrderID)
	}
	if record.FillStatus != history.FillStatusPlaced {
		t.Errorf("fill_status = %s, want PLACED", record.FillStatus)
	}
	if record.StrategyID != "default" {
		t.Errorf("strategy_id = %s, want default", record.StrategyID)
	}
	if record.LatencyMs != 150 {
		t.Errorf("latency_ms = %d, want 150", record.LatencyMs)
	}

	if len(pub.events) != 1 {
		t.Fatalf("expected 1 NATS event, got %d", len(pub.events))
	}
	if pub.events[0].EventType != "TradeRecorded" {
		t.Errorf("event_type = %s, want TradeRecorded", pub.events[0].EventType)
	}
	if pub.events[0].Payload.FillStatus != "PLACED" {
		t.Errorf("payload fill_status = %s, want PLACED", pub.events[0].Payload.FillStatus)
	}
}

func TestHandleOrderResult_FAILED(t *testing.T) {
	repo := &mockTradeRepo{}
	pub := &mockTradePublisher{}
	handler := newTestHandler(repo, pub)

	now := time.Now().UTC()
	result := &history.OrderResult{
		ClientOrderID:   "client-2",
		MarketID:        "market-1",
		MarketSlug:      "test-market",
		Side:            "YES",
		OrderType:       "GTC",
		Price:           decimal.RequireFromString("0.5500"),
		Quantity:        decimal.Zero,
		FilledQuantity:  decimal.Zero,
		FillStatus:      history.FillStatusFailed,
		LatencyMs:       0,
		SignalTimestamp: now,
		OrderTimestamp:  now,
		RiskDecision:    "risk_denied",
		FailureReason:   "daily_budget_exhausted",
		StrategyID:      "default",
	}

	err := handler.HandleOrderResult(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(repo.records))
	}

	record := repo.records[0]
	if record.FillStatus != history.FillStatusFailed {
		t.Errorf("fill_status = %s, want FAILED", record.FillStatus)
	}
	if record.FailureReason == nil || *record.FailureReason != "daily_budget_exhausted" {
		t.Errorf("failure_reason = %v, want daily_budget_exhausted", record.FailureReason)
	}

	if len(pub.events) != 1 {
		t.Fatalf("expected 1 NATS event, got %d", len(pub.events))
	}
	if pub.events[0].Payload.FillStatus != "FAILED" {
		t.Errorf("payload fill_status = %s, want FAILED", pub.events[0].Payload.FillStatus)
	}
}

func TestHandleOrderResult_Idempotent(t *testing.T) {
	repo := &mockTradeRepo{}
	pub := &mockTradePublisher{}
	handler := newTestHandler(repo, pub)

	now := time.Now().UTC()
	result := &history.OrderResult{
		ClientOrderID:  "client-dup",
		MarketID:       "market-1",
		MarketSlug:     "test-market",
		Side:           "YES",
		OrderType:      "GTC",
		Price:          decimal.RequireFromString("0.5000"),
		Quantity:       decimal.RequireFromString("100.00000000"),
		FilledQuantity: decimal.Zero,
		FillStatus:     history.FillStatusPlaced,
		LatencyMs:      100,
		SignalTimestamp: now,
		OrderTimestamp:  now,
		RiskDecision:   "APPROVED",
		StrategyID:     "default",
	}

	err := handler.HandleOrderResult(context.Background(), result)
	if err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	err = handler.HandleOrderResult(context.Background(), result)
	if err != nil {
		t.Fatalf("second insert should be idempotent, got: %v", err)
	}
}

func TestHandleOrderResult_RepoError(t *testing.T) {
	repo := &mockTradeRepo{insertErr: context.DeadlineExceeded}
	pub := &mockTradePublisher{}
	handler := newTestHandler(repo, pub)

	now := time.Now().UTC()
	result := &history.OrderResult{
		ClientOrderID:  "client-err",
		MarketID:       "market-1",
		MarketSlug:     "test-market",
		Side:           "YES",
		OrderType:      "GTC",
		Price:          decimal.RequireFromString("0.5000"),
		Quantity:       decimal.RequireFromString("100.00000000"),
		FilledQuantity: decimal.Zero,
		FillStatus:     history.FillStatusPlaced,
		LatencyMs:      100,
		SignalTimestamp: now,
		OrderTimestamp:  now,
		RiskDecision:   "APPROVED",
		StrategyID:     "default",
	}

	err := handler.HandleOrderResult(context.Background(), result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if len(pub.events) != 0 {
		t.Errorf("should not publish event on repo error, got %d events", len(pub.events))
	}
}

func TestHandleOrderResult_NilPublisher(t *testing.T) {
	repo := &mockTradeRepo{}
	handler := newTestHandler(repo, nil)

	now := time.Now().UTC()
	result := &history.OrderResult{
		ClientOrderID:  "client-nopub",
		MarketID:       "market-1",
		MarketSlug:     "test-market",
		Side:           "YES",
		OrderType:      "GTC",
		Price:          decimal.RequireFromString("0.5000"),
		Quantity:       decimal.RequireFromString("100.00000000"),
		FilledQuantity: decimal.Zero,
		FillStatus:     history.FillStatusPlaced,
		LatencyMs:      100,
		SignalTimestamp: now,
		OrderTimestamp:  now,
		RiskDecision:   "APPROVED",
		StrategyID:     "default",
	}

	err := handler.HandleOrderResult(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(repo.records))
	}
}
