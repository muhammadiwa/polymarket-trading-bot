package adapters_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/shopspring/decimal"
)

type testPublisher struct {
	events []interface{}
}

func (p *testPublisher) PublishOrderPlaced(ctx context.Context, event ports.OrderPlaced) error {
	p.events = append(p.events, event)
	return nil
}

func (p *testPublisher) PublishOrderFilled(ctx context.Context, event ports.OrderFilled) error {
	p.events = append(p.events, event)
	return nil
}

func (p *testPublisher) PublishOrderPartialFill(ctx context.Context, event ports.OrderPartialFill) error {
	p.events = append(p.events, event)
	return nil
}

func (p *testPublisher) PublishOrderCancelled(ctx context.Context, event ports.OrderCancelled) error {
	p.events = append(p.events, event)
	return nil
}

func (p *testPublisher) PublishOrderFailed(ctx context.Context, event ports.OrderFailed) error {
	p.events = append(p.events, event)
	return nil
}

func (p *testPublisher) PublishRiskAlert(ctx context.Context, event ports.RiskAlert) error {
	p.events = append(p.events, event)
	return nil
}

func (p *testPublisher) PublishAtomicLegFailed(ctx context.Context, event ports.AtomicLegFailed) error {
	p.events = append(p.events, event)
	return nil
}

func (p *testPublisher) PublishCircuitBreakerTripped(ctx context.Context, event ports.CircuitBreakerTripped) error {
	p.events = append(p.events, event)
	return nil
}

func (p *testPublisher) PublishCircuitBreakerResumed(ctx context.Context, event ports.CircuitBreakerResumed) error {
	p.events = append(p.events, event)
	return nil
}

func (p *testPublisher) PublishNotificationRequest(ctx context.Context, event ports.NotificationRequest) error {
	p.events = append(p.events, event)
	return nil
}

func (p *testPublisher) PublishTradeRecorded(ctx context.Context, event ports.TradeRecorded) error {
	p.events = append(p.events, event)
	return nil
}

func (p *testPublisher) Close() error {
	return nil
}

func TestEventSchema_OrderPlaced(t *testing.T) {
	event := ports.OrderPlaced{
		EventID:   uuid.New().String(),
		EventType: "OrderPlaced",
		Timestamp: time.Now().UTC(),
		Source:    "execution-engine",
		Payload: ports.OrderPlacedPayload{
			OrderID:       uuid.New().String(),
			ClientOrderID: uuid.New().String(),
			OpportunityID: uuid.New().String(),
			MarketID:      "market-1",
			Side:          "BUY",
			Price:         decimal.RequireFromString("0.50"),
			Size:          decimal.NewFromInt(100),
			StrategyID:    "default",
		},
	}

	if event.EventID == "" {
		t.Error("event_id should not be empty")
	}
	if event.EventType != "OrderPlaced" {
		t.Errorf("event_type = %s, want OrderPlaced", event.EventType)
	}
	if event.Source != "execution-engine" {
		t.Errorf("source = %s, want execution-engine", event.Source)
	}
	if event.Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}
	if event.Timestamp.Location() != time.UTC {
		t.Error("timestamp should be UTC")
	}
}

func TestEventSchema_OrderFilled(t *testing.T) {
	event := ports.OrderFilled{
		EventID:   uuid.New().String(),
		EventType: "OrderFilled",
		Timestamp: time.Now().UTC(),
		Source:    "execution-engine",
		Payload: ports.OrderFilledPayload{
			OrderID:       uuid.New().String(),
			ClientOrderID: uuid.New().String(),
			OpportunityID: uuid.New().String(),
			MarketID:      "market-1",
			Side:          "BUY",
			Price:         decimal.RequireFromString("0.50"),
			FilledQty:     decimal.NewFromInt(100),
			LatencyMs:     150,
			StrategyID:    "default",
		},
	}

	if event.EventID == "" {
		t.Error("event_id should not be empty")
	}
	if event.EventType != "OrderFilled" {
		t.Errorf("event_type = %s, want OrderFilled", event.EventType)
	}
	if event.Source != "execution-engine" {
		t.Errorf("source = %s, want execution-engine", event.Source)
	}
}

func TestEventSchema_OrderFailed(t *testing.T) {
	event := ports.OrderFailed{
		EventID:   uuid.New().String(),
		EventType: "OrderFailed",
		Timestamp: time.Now().UTC(),
		Source:    "execution-engine",
		Payload: ports.OrderFailedPayload{
			OrderID:       uuid.New().String(),
			ClientOrderID: uuid.New().String(),
			OpportunityID: uuid.New().String(),
			MarketID:      "market-1",
			Reason:        "risk_denied",
			ErrorDetail:   "daily budget exhausted",
			StrategyID:    "default",
		},
	}

	if event.EventID == "" {
		t.Error("event_id should not be empty")
	}
	if event.EventType != "OrderFailed" {
		t.Errorf("event_type = %s, want OrderFailed", event.EventType)
	}
	if event.Payload.Reason != "risk_denied" {
		t.Errorf("reason = %s, want risk_denied", event.Payload.Reason)
	}
}

func TestEventSchema_RiskAlert(t *testing.T) {
	event := ports.RiskAlert{
		EventID:   uuid.New().String(),
		EventType: "RiskAlert",
		Timestamp: time.Now().UTC(),
		Source:    "execution-engine",
		Payload: ports.RiskAlertPayload{
			AlertType: "circuit_breaker_tripped",
			Message:   "CLOB API circuit breaker tripped",
			Severity:  "critical",
		},
	}

	if event.EventID == "" {
		t.Error("event_id should not be empty")
	}
	if event.Payload.Severity != "critical" {
		t.Errorf("severity = %s, want critical", event.Payload.Severity)
	}
}
