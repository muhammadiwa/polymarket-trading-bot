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

type mockOrderPortAtomic struct {
	placeErr   error
	cancelErr  error
	statusErr  error
	orderResp  *ports.OrderResponse
	statusResp *ports.OrderStatusResponse
	placeDelay time.Duration
}

func (m *mockOrderPortAtomic) PlaceOrder(req ports.OrderRequest, clientOrderID string) (*ports.OrderResponse, error) {
	if m.placeDelay > 0 {
		time.Sleep(m.placeDelay)
	}
	if m.placeErr != nil {
		return nil, m.placeErr
	}
	return m.orderResp, nil
}

func (m *mockOrderPortAtomic) CancelOrder(ctx context.Context, orderID string) error {
	return m.cancelErr
}

func (m *mockOrderPortAtomic) GetOrderStatus(orderID string) (*ports.OrderStatusResponse, error) {
	return m.statusResp, m.statusErr
}

type mockRiskPortAtomic struct {
	decision *ports.RiskDecision
	err      error
}

func (m *mockRiskPortAtomic) CheckRisk(ctx context.Context, marketID, strategyID string, orderSize decimal.Decimal) (*ports.RiskDecision, error) {
	return m.decision, m.err
}

type mockPublisherAtomic struct {
	placedEvents    []ports.OrderPlaced
	filledEvents    []ports.OrderFilled
	failedEvents    []ports.OrderFailed
	cancelledEvents []ports.OrderCancelled
	atomicFailed    []ports.AtomicLegFailed
	notifications   []ports.NotificationRequest
}

func (m *mockPublisherAtomic) PublishOrderPlaced(ctx context.Context, event ports.OrderPlaced) error {
	m.placedEvents = append(m.placedEvents, event)
	return nil
}

func (m *mockPublisherAtomic) PublishOrderFilled(ctx context.Context, event ports.OrderFilled) error {
	m.filledEvents = append(m.filledEvents, event)
	return nil
}

func (m *mockPublisherAtomic) PublishOrderPartialFill(ctx context.Context, event ports.OrderPartialFill) error {
	return nil
}

func (m *mockPublisherAtomic) PublishOrderCancelled(ctx context.Context, event ports.OrderCancelled) error {
	m.cancelledEvents = append(m.cancelledEvents, event)
	return nil
}

func (m *mockPublisherAtomic) PublishOrderFailed(ctx context.Context, event ports.OrderFailed) error {
	m.failedEvents = append(m.failedEvents, event)
	return nil
}

func (m *mockPublisherAtomic) PublishRiskAlert(ctx context.Context, event ports.RiskAlert) error {
	return nil
}

func (m *mockPublisherAtomic) PublishAtomicLegFailed(ctx context.Context, event ports.AtomicLegFailed) error {
	m.atomicFailed = append(m.atomicFailed, event)
	return nil
}

func (m *mockPublisherAtomic) PublishCircuitBreakerTripped(ctx context.Context, event ports.CircuitBreakerTripped) error {
	return nil
}

func (m *mockPublisherAtomic) PublishCircuitBreakerResumed(ctx context.Context, event ports.CircuitBreakerResumed) error {
	return nil
}

func (m *mockPublisherAtomic) PublishNotificationRequest(ctx context.Context, event ports.NotificationRequest) error {
	m.notifications = append(m.notifications, event)
	return nil
}

func (m *mockPublisherAtomic) PublishTradeRecorded(ctx context.Context, event ports.TradeRecorded) error {
	return nil
}

func (m *mockPublisherAtomic) Close() error {
	return nil
}

type mockRiskEventRepoAtomic struct {
	events []ports.RiskEvent
}

func (m *mockRiskEventRepoAtomic) InsertRiskEvent(ctx context.Context, event ports.RiskEvent) error {
	m.events = append(m.events, event)
	return nil
}

type mockAtomicLogger struct {
	records []executor.AtomicPairRecord
}

func (m *mockAtomicLogger) LogAtomicPair(ctx context.Context, record executor.AtomicPairRecord) error {
	m.records = append(m.records, record)
	return nil
}

func newTestAtomicExecutor(
	orderPort ports.OrderPort,
	riskPort ports.RiskPort,
	publisher ports.EventPublisher,
) *executor.AtomicExecutor {
	log, _ := zap.NewDevelopment()
	mockRiskRepo := &mockRiskEventRepoAtomic{}
	mockPrice := &mockMarketPricePort{price: decimal.RequireFromString("0.55")}
	mockAtomicLog := &mockAtomicLogger{}

	return executor.NewAtomicExecutor(
		orderPort,
		riskPort,
		publisher,
		mockRiskRepo,
		mockPrice,
		0.01,
		"GTC",
		500*time.Millisecond,
		1000*time.Millisecond,
		mockAtomicLog,
		log,
	)
}

func TestAtomicExecutor_BothLegsSuccess(t *testing.T) {
	mockOrder := &mockOrderPortAtomic{
		orderResp: &ports.OrderResponse{
			OrderID:       "order-1",
			ClientOrderID: "client-1",
			Status:        "PLACED",
		},
	}

	mockRisk := &mockRiskPortAtomic{
		decision: &ports.RiskDecision{
			Allowed: true,
			Reason:  "",
		},
	}

	mockPub := &mockPublisherAtomic{}

	exec := newTestAtomicExecutor(mockOrder, mockRisk, mockPub)

	opp := ports.OpportunityDetected{
		EventID:   "event-1",
		EventType: "OpportunityDetected",
		Timestamp: time.Now().UTC(),
		Source:    "arb-engine",
		Payload: ports.OpportunityPayload{
			OpportunityID: "opp-1",
			MarketID:      "market-1",
			YESPrice:      decimal.RequireFromString("0.55"),
			NOPrice:       decimal.RequireFromString("0.40"),
			Spread:        decimal.RequireFromString("0.05"),
			Score:         decimal.RequireFromString("0.8"),
			StrategyID:    "test-strategy",
		},
	}

	err := exec.ExecuteAtomic(context.Background(), opp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAtomicExecutor_RiskDenied(t *testing.T) {
	mockOrder := &mockOrderPortAtomic{}

	mockRisk := &mockRiskPortAtomic{
		decision: &ports.RiskDecision{
			Allowed: false,
			Reason:  "daily_budget_exhausted",
		},
	}

	mockPub := &mockPublisherAtomic{}

	exec := newTestAtomicExecutor(mockOrder, mockRisk, mockPub)

	opp := ports.OpportunityDetected{
		EventID:   "event-1",
		EventType: "OpportunityDetected",
		Timestamp: time.Now().UTC(),
		Source:    "arb-engine",
		Payload: ports.OpportunityPayload{
			OpportunityID: "opp-1",
			MarketID:      "market-1",
			YESPrice:      decimal.RequireFromString("0.55"),
			NOPrice:       decimal.RequireFromString("0.40"),
			StrategyID:    "test-strategy",
		},
	}

	err := exec.ExecuteAtomic(context.Background(), opp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if len(mockPub.failedEvents) != 1 {
		t.Errorf("expected 1 failed event, got %d", len(mockPub.failedEvents))
	}
}

func TestAtomicExecutor_Timeout(t *testing.T) {
	mockOrder := &mockOrderPortAtomic{
		placeDelay: 600 * time.Millisecond,
		orderResp: &ports.OrderResponse{
			OrderID:       "order-1",
			ClientOrderID: "client-1",
			Status:        "PLACED",
		},
	}

	mockRisk := &mockRiskPortAtomic{
		decision: &ports.RiskDecision{
			Allowed: true,
			Reason:  "",
		},
	}

	mockPub := &mockPublisherAtomic{}

	log, _ := zap.NewDevelopment()
	mockRiskRepo := &mockRiskEventRepoAtomic{}
	mockPrice := &mockMarketPricePort{price: decimal.RequireFromString("0.55")}
	mockAtomicLog := &mockAtomicLogger{}

	exec := executor.NewAtomicExecutor(
		mockOrder,
		mockRisk,
		mockPub,
		mockRiskRepo,
		mockPrice,
		0.01,
		"GTC",
		100*time.Millisecond,
		1000*time.Millisecond,
		mockAtomicLog,
		log,
	)

	opp := ports.OpportunityDetected{
		EventID:   "event-1",
		EventType: "OpportunityDetected",
		Timestamp: time.Now().UTC(),
		Source:    "arb-engine",
		Payload: ports.OpportunityPayload{
			OpportunityID: "opp-1",
			MarketID:      "market-1",
			YESPrice:      decimal.RequireFromString("0.55"),
			NOPrice:       decimal.RequireFromString("0.40"),
			StrategyID:    "test-strategy",
		},
	}

	err := exec.ExecuteAtomic(context.Background(), opp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAtomicExecutor_APIError(t *testing.T) {
	mockOrder := &mockOrderPortAtomic{
		placeErr: context.DeadlineExceeded,
	}

	mockRisk := &mockRiskPortAtomic{
		decision: &ports.RiskDecision{
			Allowed: true,
			Reason:  "",
		},
	}

	mockPub := &mockPublisherAtomic{}

	exec := newTestAtomicExecutor(mockOrder, mockRisk, mockPub)

	opp := ports.OpportunityDetected{
		EventID:   "event-1",
		EventType: "OpportunityDetected",
		Timestamp: time.Now().UTC(),
		Source:    "arb-engine",
		Payload: ports.OpportunityPayload{
			OpportunityID: "opp-1",
			MarketID:      "market-1",
			YESPrice:      decimal.RequireFromString("0.55"),
			NOPrice:       decimal.RequireFromString("0.40"),
			StrategyID:    "test-strategy",
		},
	}

	err := exec.ExecuteAtomic(context.Background(), opp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
