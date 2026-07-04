package executor_test

import (
	"context"
	"testing"
	"time"

	"github.com/pqap/services/execution-engine/internal/history"
	"github.com/pqap/services/execution-engine/internal/executor"
	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type mockOrderPortExec struct {
	placeErr   error
	cancelErr  error
	statusErr  error
	orderResp  *ports.OrderResponse
	statusResp *ports.OrderStatusResponse
}

func (m *mockOrderPortExec) PlaceOrder(req ports.OrderRequest, clientOrderID string) (*ports.OrderResponse, error) {
	if m.placeErr != nil {
		return nil, m.placeErr
	}
	return m.orderResp, nil
}

func (m *mockOrderPortExec) CancelOrder(ctx context.Context, orderID string) error {
	return m.cancelErr
}

func (m *mockOrderPortExec) GetOrderStatus(orderID string) (*ports.OrderStatusResponse, error) {
	return m.statusResp, m.statusErr
}

type mockRiskPortExec struct {
	decision *ports.RiskDecision
	err      error
}

func (m *mockRiskPortExec) CheckRisk(ctx context.Context, marketID, strategyID string, orderSize decimal.Decimal) (*ports.RiskDecision, error) {
	return m.decision, m.err
}

type mockPublisherExec struct {
	placedEvents []ports.OrderPlaced
	filledEvents []ports.OrderFilled
	failedEvents []ports.OrderFailed
}

func (m *mockPublisherExec) PublishOrderPlaced(ctx context.Context, event ports.OrderPlaced) error {
	m.placedEvents = append(m.placedEvents, event)
	return nil
}

func (m *mockPublisherExec) PublishOrderFilled(ctx context.Context, event ports.OrderFilled) error {
	m.filledEvents = append(m.filledEvents, event)
	return nil
}

func (m *mockPublisherExec) PublishOrderPartialFill(ctx context.Context, event ports.OrderPartialFill) error {
	return nil
}

func (m *mockPublisherExec) PublishOrderCancelled(ctx context.Context, event ports.OrderCancelled) error {
	return nil
}

func (m *mockPublisherExec) PublishOrderFailed(ctx context.Context, event ports.OrderFailed) error {
	m.failedEvents = append(m.failedEvents, event)
	return nil
}

func (m *mockPublisherExec) PublishRiskAlert(ctx context.Context, event ports.RiskAlert) error {
	return nil
}

func (m *mockPublisherExec) PublishAtomicLegFailed(ctx context.Context, event ports.AtomicLegFailed) error {
	return nil
}

func (m *mockPublisherExec) PublishCircuitBreakerTripped(ctx context.Context, event ports.CircuitBreakerTripped) error {
	return nil
}

func (m *mockPublisherExec) PublishCircuitBreakerResumed(ctx context.Context, event ports.CircuitBreakerResumed) error {
	return nil
}

func (m *mockPublisherExec) PublishNotificationRequest(ctx context.Context, event ports.NotificationRequest) error {
	return nil
}

func (m *mockPublisherExec) PublishTradeRecorded(ctx context.Context, event ports.TradeRecorded) error {
	return nil
}

func (m *mockPublisherExec) Close() error {
	return nil
}

type mockMarketPricePort struct {
	price decimal.Decimal
	err   error
}

func (m *mockMarketPricePort) GetCurrentPrice(ctx context.Context, marketID string) (decimal.Decimal, error) {
	return m.price, m.err
}

type mockRiskEventRepoExec struct {
	events []ports.RiskEvent
}

func (m *mockRiskEventRepoExec) InsertRiskEvent(ctx context.Context, event ports.RiskEvent) error {
	m.events = append(m.events, event)
	return nil
}

type mockOrderLoggerExec struct {
	orders []*ports.Order
}

func (m *mockOrderLoggerExec) LogOrder(ctx context.Context, order *ports.Order) error {
	m.orders = append(m.orders, order)
	return nil
}

type mockFillMonitorExec struct {
	orders []*ports.Order
}

func (m *mockFillMonitorExec) StartMonitoring(ctx context.Context, order *ports.Order) {
	m.orders = append(m.orders, order)
}

type mockTradeHistoryHandler struct {
	results []*history.OrderResult
}

func (m *mockTradeHistoryHandler) HandleOrderResult(ctx context.Context, result *history.OrderResult) error {
	m.results = append(m.results, result)
	return nil
}

func newTestExecutor(
	orderPort ports.OrderPort,
	riskPort ports.RiskPort,
	publisher ports.EventPublisher,
) *executor.Executor {
	log, _ := zap.NewDevelopment()
	mockPrice := &mockMarketPricePort{price: decimal.RequireFromString("0.55")}
	mockRiskRepo := &mockRiskEventRepoExec{}
	mockLogger := &mockOrderLoggerExec{}
	mockFillMon := &mockFillMonitorExec{}
	mockTradeHandler := &mockTradeHistoryHandler{}

	return executor.NewExecutor(
		orderPort,
		riskPort,
		publisher,
		mockPrice,
		0.01,
		"GTC",
		0,
		100*time.Millisecond,
		5*time.Second,
		10,
		mockLogger,
		mockFillMon,
		mockRiskRepo,
		mockTradeHandler,
		log,
	)
}

func TestExecutor_SuccessfulExecution(t *testing.T) {
	mockOrder := &mockOrderPortExec{
		orderResp: &ports.OrderResponse{
			OrderID:       "order-1",
			ClientOrderID: "client-1",
			Status:        "PLACED",
		},
	}

	mockRisk := &mockRiskPortExec{
		decision: &ports.RiskDecision{
			Allowed: true,
			Reason:  "",
		},
	}

	mockPub := &mockPublisherExec{}

	exec := newTestExecutor(mockOrder, mockRisk, mockPub)

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
		},
	}

	err := exec.Execute(context.Background(), opp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockPub.placedEvents) != 1 {
		t.Errorf("expected 1 placed event, got %d", len(mockPub.placedEvents))
	}
}

func TestExecutor_RiskDenied(t *testing.T) {
	mockOrder := &mockOrderPortExec{}

	mockRisk := &mockRiskPortExec{
		decision: &ports.RiskDecision{
			Allowed: false,
			Reason:  "daily_budget_exhausted",
		},
	}

	mockPub := &mockPublisherExec{}

	exec := newTestExecutor(mockOrder, mockRisk, mockPub)

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
		},
	}

	err := exec.Execute(context.Background(), opp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if len(mockPub.failedEvents) != 1 {
		t.Errorf("expected 1 failed event, got %d", len(mockPub.failedEvents))
	}

	if mockPub.failedEvents[0].Payload.Reason != "risk_denied" {
		t.Errorf("reason = %s, want risk_denied", mockPub.failedEvents[0].Payload.Reason)
	}
}

func TestExecutor_APIError(t *testing.T) {
	mockOrder := &mockOrderPortExec{
		placeErr: context.DeadlineExceeded,
	}

	mockRisk := &mockRiskPortExec{
		decision: &ports.RiskDecision{
			Allowed: true,
			Reason:  "",
		},
	}

	mockPub := &mockPublisherExec{}

	exec := newTestExecutor(mockOrder, mockRisk, mockPub)

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
		},
	}

	err := exec.Execute(context.Background(), opp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if len(mockPub.failedEvents) != 1 {
		t.Errorf("expected 1 failed event, got %d", len(mockPub.failedEvents))
	}

	if mockPub.failedEvents[0].Payload.Reason != "api_error" {
		t.Errorf("reason = %s, want api_error", mockPub.failedEvents[0].Payload.Reason)
	}
}
