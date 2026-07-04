package executor_test

import (
	"context"
	"testing"
	"time"

	"github.com/pqap/services/execution-engine/internal/executor"
	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type mockOrderPortLeg struct {
	cancelErr error
	cancelled []string
}

func (m *mockOrderPortLeg) PlaceOrder(req ports.OrderRequest, clientOrderID string) (*ports.OrderResponse, error) {
	return &ports.OrderResponse{
		OrderID:       "order-1",
		ClientOrderID: clientOrderID,
		Status:        "PLACED",
	}, nil
}

func (m *mockOrderPortLeg) CancelOrder(ctx context.Context, orderID string) error {
	m.cancelled = append(m.cancelled, orderID)
	return m.cancelErr
}

func (m *mockOrderPortLeg) GetOrderStatus(orderID string) (*ports.OrderStatusResponse, error) {
	return &ports.OrderStatusResponse{
		OrderID:      orderID,
		Status:       "PLACED",
		FilledQty:    decimal.Zero,
		RemainingQty: decimal.NewFromInt(100),
		Price:        decimal.NewFromFloat(0.55),
	}, nil
}

type mockPublisherLeg struct {
	atomicFailed []ports.AtomicLegFailed
}

func (m *mockPublisherLeg) PublishOrderPlaced(ctx context.Context, event ports.OrderPlaced) error {
	return nil
}

func (m *mockPublisherLeg) PublishOrderFilled(ctx context.Context, event ports.OrderFilled) error {
	return nil
}

func (m *mockPublisherLeg) PublishOrderPartialFill(ctx context.Context, event ports.OrderPartialFill) error {
	return nil
}

func (m *mockPublisherLeg) PublishOrderCancelled(ctx context.Context, event ports.OrderCancelled) error {
	return nil
}

func (m *mockPublisherLeg) PublishOrderFailed(ctx context.Context, event ports.OrderFailed) error {
	return nil
}

func (m *mockPublisherLeg) PublishRiskAlert(ctx context.Context, event ports.RiskAlert) error {
	return nil
}

func (m *mockPublisherLeg) PublishAtomicLegFailed(ctx context.Context, event ports.AtomicLegFailed) error {
	m.atomicFailed = append(m.atomicFailed, event)
	return nil
}

func (m *mockPublisherLeg) PublishCircuitBreakerTripped(ctx context.Context, event ports.CircuitBreakerTripped) error {
	return nil
}

func (m *mockPublisherLeg) PublishCircuitBreakerResumed(ctx context.Context, event ports.CircuitBreakerResumed) error {
	return nil
}

func (m *mockPublisherLeg) PublishNotificationRequest(ctx context.Context, event ports.NotificationRequest) error {
	return nil
}

func (m *mockPublisherLeg) PublishTradeRecorded(ctx context.Context, event ports.TradeRecorded) error {
	return nil
}

func (m *mockPublisherLeg) Close() error {
	return nil
}

func TestLegHandler_CancelSuccess(t *testing.T) {
	mockOrder := &mockOrderPortLeg{}
	mockPub := &mockPublisherLeg{}
	log, _ := zap.NewDevelopment()

	handler := executor.NewLegHandler(mockOrder, mockPub, 1*time.Second, log)

	handler.HandleLegFailure(context.Background(), &executor.LegFailureContext{
		PairID:           "pair-1",
		OpportunityID:    "opp-1",
		MarketID:         "market-1",
		FailedLeg:        "YES",
		CancelledLeg:     "NO",
		CancelledOrderID: "order-no-1",
		StrategyID:       "test-strategy",
	})

	if len(mockOrder.cancelled) != 1 {
		t.Errorf("expected 1 cancelled order, got %d", len(mockOrder.cancelled))
	}

	if mockOrder.cancelled[0] != "order-no-1" {
		t.Errorf("cancelled order = %s, want order-no-1", mockOrder.cancelled[0])
	}

	if len(mockPub.atomicFailed) != 1 {
		t.Errorf("expected 1 AtomicLegFailed event, got %d", len(mockPub.atomicFailed))
	}

	if mockPub.atomicFailed[0].Payload.FailedLeg != "YES" {
		t.Errorf("failed leg = %s, want YES", mockPub.atomicFailed[0].Payload.FailedLeg)
	}
}

func TestLegHandler_CancelError(t *testing.T) {
	mockOrder := &mockOrderPortLeg{
		cancelErr: context.DeadlineExceeded,
	}
	mockPub := &mockPublisherLeg{}
	log, _ := zap.NewDevelopment()

	handler := executor.NewLegHandler(mockOrder, mockPub, 1*time.Second, log)

	handler.HandleLegFailure(context.Background(), &executor.LegFailureContext{
		PairID:           "pair-1",
		OpportunityID:    "opp-1",
		MarketID:         "market-1",
		FailedLeg:        "NO",
		CancelledLeg:     "YES",
		CancelledOrderID: "order-yes-1",
		StrategyID:       "test-strategy",
	})

	if len(mockOrder.cancelled) != 1 {
		t.Errorf("expected 1 cancelled order, got %d", len(mockOrder.cancelled))
	}

	if len(mockPub.atomicFailed) != 1 {
		t.Errorf("expected 1 AtomicLegFailed event, got %d", len(mockPub.atomicFailed))
	}
}

func TestLegHandler_PartialFill(t *testing.T) {
	mockOrder := &mockOrderPortLeg{}
	mockPub := &mockPublisherLeg{}
	log, _ := zap.NewDevelopment()

	handler := executor.NewLegHandler(mockOrder, mockPub, 1*time.Second, log)

	handler.HandleLegFailure(context.Background(), &executor.LegFailureContext{
		PairID:           "pair-1",
		OpportunityID:    "opp-1",
		MarketID:         "market-1",
		FailedLeg:        "YES",
		CancelledLeg:     "NO",
		CancelledOrderID: "order-no-1",
		StrategyID:       "test-strategy",
	})

	if len(mockPub.atomicFailed) != 1 {
		t.Errorf("expected 1 AtomicLegFailed event, got %d", len(mockPub.atomicFailed))
	}

	if mockPub.atomicFailed[0].Payload.CancelledLeg != "NO" {
		t.Errorf("cancelled leg = %s, want NO", mockPub.atomicFailed[0].Payload.CancelledLeg)
	}
}

func TestLegHandler_CancelLatency(t *testing.T) {
	mockOrder := &mockOrderPortLeg{}
	mockPub := &mockPublisherLeg{}
	log, _ := zap.NewDevelopment()

	handler := executor.NewLegHandler(mockOrder, mockPub, 1*time.Second, log)

	start := time.Now()
	handler.HandleLegFailure(context.Background(), &executor.LegFailureContext{
		PairID:           "pair-1",
		OpportunityID:    "opp-1",
		MarketID:         "market-1",
		FailedLeg:        "YES",
		CancelledLeg:     "NO",
		CancelledOrderID: "order-no-1",
		StrategyID:       "test-strategy",
	})
	elapsed := time.Since(start)

	if elapsed > 1*time.Second {
		t.Errorf("cancel latency %v exceeded 1s target", elapsed)
	}
}
