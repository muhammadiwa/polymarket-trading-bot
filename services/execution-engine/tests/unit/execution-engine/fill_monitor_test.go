package executor_test

import (
	"context"
	"testing"
	"time"

	"github.com/pqap/services/execution-engine/internal/monitor"
	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type mockOrderPort struct {
	status *ports.OrderStatusResponse
	err    error
}

func (m *mockOrderPort) PlaceOrder(req ports.OrderRequest, clientOrderID string) (*ports.OrderResponse, error) {
	return &ports.OrderResponse{
		OrderID:       "order-1",
		ClientOrderID: clientOrderID,
		Status:        "PLACED",
	}, nil
}

func (m *mockOrderPort) CancelOrder(ctx context.Context, orderID string) error {
	return nil
}

func (m *mockOrderPort) GetOrderStatus(orderID string) (*ports.OrderStatusResponse, error) {
	return m.status, m.err
}

type mockEventPublisher struct {
	filledEvents    []ports.OrderFilled
	partialEvents   []ports.OrderPartialFill
	cancelledEvents []ports.OrderCancelled
	failedEvents    []ports.OrderFailed
}

func (m *mockEventPublisher) PublishOrderPlaced(ctx context.Context, event ports.OrderPlaced) error {
	return nil
}

func (m *mockEventPublisher) PublishOrderFilled(ctx context.Context, event ports.OrderFilled) error {
	m.filledEvents = append(m.filledEvents, event)
	return nil
}

func (m *mockEventPublisher) PublishOrderPartialFill(ctx context.Context, event ports.OrderPartialFill) error {
	m.partialEvents = append(m.partialEvents, event)
	return nil
}

func (m *mockEventPublisher) PublishOrderCancelled(ctx context.Context, event ports.OrderCancelled) error {
	m.cancelledEvents = append(m.cancelledEvents, event)
	return nil
}

func (m *mockEventPublisher) PublishOrderFailed(ctx context.Context, event ports.OrderFailed) error {
	m.failedEvents = append(m.failedEvents, event)
	return nil
}

func (m *mockEventPublisher) PublishRiskAlert(ctx context.Context, event ports.RiskAlert) error {
	return nil
}

func (m *mockEventPublisher) PublishAtomicLegFailed(ctx context.Context, event ports.AtomicLegFailed) error {
	return nil
}

func (m *mockEventPublisher) PublishCircuitBreakerTripped(ctx context.Context, event ports.CircuitBreakerTripped) error {
	return nil
}

func (m *mockEventPublisher) PublishCircuitBreakerResumed(ctx context.Context, event ports.CircuitBreakerResumed) error {
	return nil
}

func (m *mockEventPublisher) PublishNotificationRequest(ctx context.Context, event ports.NotificationRequest) error {
	return nil
}

func (m *mockEventPublisher) PublishTradeRecorded(ctx context.Context, event ports.TradeRecorded) error {
	return nil
}

func (m *mockEventPublisher) Close() error {
	return nil
}

func TestFillMonitor_FullFill(t *testing.T) {
	mockOrder := &mockOrderPort{
		status: &ports.OrderStatusResponse{
			OrderID:      "order-1",
			Status:       "FILLED",
			FilledQty:    decimal.NewFromInt(100),
			RemainingQty: decimal.Zero,
			Price:        decimal.RequireFromString("0.50"),
		},
	}

	mockPub := &mockEventPublisher{}

	log, _ := zap.NewDevelopment()
	mon := monitor.NewFillMonitor(mockOrder, mockPub, 10*time.Millisecond, 1*time.Second, log)

	order := &ports.Order{
		ID:            "order-1",
		ClientOrderID: "client-1",
		OpportunityID: "opp-1",
		MarketID:      "market-1",
		Side:          "BUY",
		Price:         decimal.RequireFromString("0.50"),
		Size:          decimal.NewFromInt(100),
		StrategyID:    "default",
		PlacedAt:      time.Now().UTC(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	mon.MonitorOrder(ctx, order)

	if len(mockPub.filledEvents) != 1 {
		t.Errorf("expected 1 filled event, got %d", len(mockPub.filledEvents))
	}
}

func TestFillMonitor_PartialFill(t *testing.T) {
	callCount := 0
	mockOrder := &mockOrderPort{
		status: &ports.OrderStatusResponse{
			OrderID:      "order-1",
			Status:       "PARTIAL_FILL",
			FilledQty:    decimal.NewFromInt(50),
			RemainingQty: decimal.NewFromInt(50),
			Price:        decimal.RequireFromString("0.50"),
		},
	}

	mockPub := &mockEventPublisher{}

	log, _ := zap.NewDevelopment()
	mon := monitor.NewFillMonitor(mockOrder, mockPub, 10*time.Millisecond, 500*time.Millisecond, log)

	order := &ports.Order{
		ID:            "order-1",
		ClientOrderID: "client-1",
		OpportunityID: "opp-1",
		MarketID:      "market-1",
		Side:          "BUY",
		Price:         decimal.RequireFromString("0.50"),
		Size:          decimal.NewFromInt(100),
		StrategyID:    "default",
		PlacedAt:      time.Now().UTC(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_ = callCount
	mon.MonitorOrder(ctx, order)

	if len(mockPub.partialEvents) == 0 {
		t.Error("expected at least 1 partial fill event")
	}
}

func TestFillMonitor_Timeout(t *testing.T) {
	mockOrder := &mockOrderPort{
		status: &ports.OrderStatusResponse{
			OrderID:      "order-1",
			Status:       "PLACED",
			FilledQty:    decimal.Zero,
			RemainingQty: decimal.NewFromInt(100),
			Price:        decimal.RequireFromString("0.50"),
		},
	}

	mockPub := &mockEventPublisher{}

	log, _ := zap.NewDevelopment()
	mon := monitor.NewFillMonitor(mockOrder, mockPub, 10*time.Millisecond, 100*time.Millisecond, log)

	order := &ports.Order{
		ID:            "order-1",
		ClientOrderID: "client-1",
		OpportunityID: "opp-1",
		MarketID:      "market-1",
		Side:          "BUY",
		Price:         decimal.RequireFromString("0.50"),
		Size:          decimal.NewFromInt(100),
		StrategyID:    "default",
		PlacedAt:      time.Now().UTC(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	mon.MonitorOrder(ctx, order)

	if len(mockPub.cancelledEvents) != 1 {
		t.Errorf("expected 1 cancelled event, got %d", len(mockPub.cancelledEvents))
	}
	if mockPub.cancelledEvents[0].Payload.Reason != "timeout" {
		t.Errorf("reason = %s, want timeout", mockPub.cancelledEvents[0].Payload.Reason)
	}
}

func TestNewOrder(t *testing.T) {
	order := monitor.NewOrder("order-1", "client-1", "opp-1", "market-1", "BUY", "default",
		decimal.RequireFromString("0.50"), decimal.NewFromInt(100))

	if order.ID != "order-1" {
		t.Errorf("ID = %s, want order-1", order.ID)
	}
	if order.ClientOrderID != "client-1" {
		t.Errorf("ClientOrderID = %s, want client-1", order.ClientOrderID)
	}
	if order.Status != ports.OrderStatusPlaced {
		t.Errorf("Status = %s, want PLACED", order.Status)
	}
	if !order.FilledQty.IsZero() {
		t.Errorf("FilledQty = %s, want 0", order.FilledQty.String())
	}
	if !order.RemainingQty.Equal(decimal.NewFromInt(100)) {
		t.Errorf("RemainingQty = %s, want 100", order.RemainingQty.String())
	}
}
