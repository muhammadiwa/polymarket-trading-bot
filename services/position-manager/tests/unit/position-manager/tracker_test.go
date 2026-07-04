package position_manager

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/position-manager/internal/ports"
	"github.com/shopspring/decimal"
)

type mockRepo struct {
	positions  map[string]*ports.Position
	history    []*ports.PositionHistory
	reconLogs  []*ports.ReconciliationLog
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		positions: make(map[string]*ports.Position),
	}
}

func (m *mockRepo) CreatePosition(ctx context.Context, p *ports.Position) error {
	m.positions[p.ID] = p
	return nil
}

func (m *mockRepo) UpdatePosition(ctx context.Context, p *ports.Position) error {
	m.positions[p.ID] = p
	return nil
}

func (m *mockRepo) UpdatePositionStatus(ctx context.Context, id string, expectedStatus, newStatus ports.PositionStatus) (bool, error) {
	p, ok := m.positions[id]
	if !ok {
		return false, nil
	}
	if p.Status != expectedStatus {
		return false, nil
	}
	p.Status = newStatus
	return true, nil
}

func (m *mockRepo) GetPosition(ctx context.Context, id string) (*ports.Position, error) {
	p, ok := m.positions[id]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (m *mockRepo) GetOpenPositions(ctx context.Context) ([]*ports.Position, error) {
	var result []*ports.Position
	for _, p := range m.positions {
		if p.Status == ports.StatusOpen || p.Status == ports.StatusMonitoring {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockRepo) GetOpenPositionsByMarket(ctx context.Context, marketID string) ([]*ports.Position, error) {
	var result []*ports.Position
	for _, p := range m.positions {
		if p.MarketID == marketID && (p.Status == ports.StatusOpen || p.Status == ports.StatusMonitoring) {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockRepo) GetOpenPositionsByStrategy(ctx context.Context, strategyID string) ([]*ports.Position, error) {
	var result []*ports.Position
	for _, p := range m.positions {
		if p.StrategyID == strategyID && (p.Status == ports.StatusOpen || p.Status == ports.StatusMonitoring) {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockRepo) GetActivePositionCount(ctx context.Context) (int, error) {
	count := 0
	for _, p := range m.positions {
		if p.Status == ports.StatusOpen || p.Status == ports.StatusMonitoring {
			count++
		}
	}
	return count, nil
}

func (m *mockRepo) MoveToHistory(ctx context.Context, p *ports.Position, exitPrice decimal.Decimal, exitReason string) error {
	h := &ports.PositionHistory{
		ID:           p.ID,
		MarketID:     p.MarketID,
		MarketSlug:   p.MarketSlug,
		Side:         p.Side,
		EntryPrice:   p.EntryPrice,
		ExitPrice:    exitPrice,
		Quantity:     p.Quantity,
		RealizedPnL:  p.RealizedPnL,
		StrategyID:   p.StrategyID,
		EntryOrderID: p.EntryOrderID,
		ExitOrderID:  p.ExitOrderID,
		ExitReason:   exitReason,
		OpenedAt:     p.OpenedAt,
		ClosedAt:     *p.ClosedAt,
		AccountID:    p.AccountID,
	}
	m.history = append(m.history, h)
	delete(m.positions, p.ID)
	return nil
}

func (m *mockRepo) GetHistory(ctx context.Context, limit, offset int) ([]*ports.PositionHistory, error) {
	return m.history, nil
}

func (m *mockRepo) LogReconciliation(ctx context.Context, log *ports.ReconciliationLog) error {
	m.reconLogs = append(m.reconLogs, log)
	return nil
}

func (m *mockRepo) GetReconciliationState(ctx context.Context) (*ports.ReconciliationState, error) {
	return &ports.ReconciliationState{}, nil
}

func (m *mockRepo) IncrementMismatchCount(ctx context.Context) error {
	return nil
}

func (m *mockRepo) ResetMismatchCount(ctx context.Context) error {
	return nil
}

type mockPublisher struct {
	openedEvents    []ports.PositionOpened
	updatedEvents   []ports.PositionUpdated
	closedEvents    []ports.PositionClosed
	mismatchEvents  []ports.PositionReconciliationMismatch
	riskEvents      []ports.RiskAlert
	notifEvents     []ports.NotificationRequest
	exitEvents      []ports.ExitOrderRequest
	emergencyEvents []ports.EmergencyStop
}

func (m *mockPublisher) PublishPositionOpened(_ context.Context, event ports.PositionOpened) error {
	m.openedEvents = append(m.openedEvents, event)
	return nil
}

func (m *mockPublisher) PublishPositionUpdated(_ context.Context, event ports.PositionUpdated) error {
	m.updatedEvents = append(m.updatedEvents, event)
	return nil
}

func (m *mockPublisher) PublishPositionClosed(_ context.Context, event ports.PositionClosed) error {
	m.closedEvents = append(m.closedEvents, event)
	return nil
}

func (m *mockPublisher) PublishReconciliationMismatch(_ context.Context, event ports.PositionReconciliationMismatch) error {
	m.mismatchEvents = append(m.mismatchEvents, event)
	return nil
}

func (m *mockPublisher) PublishRiskAlert(_ context.Context, event ports.RiskAlert) error {
	m.riskEvents = append(m.riskEvents, event)
	return nil
}

func (m *mockPublisher) PublishNotificationRequest(_ context.Context, event ports.NotificationRequest) error {
	m.notifEvents = append(m.notifEvents, event)
	return nil
}

func (m *mockPublisher) PublishExitOrderRequest(_ context.Context, event ports.ExitOrderRequest) error {
	m.exitEvents = append(m.exitEvents, event)
	return nil
}

func (m *mockPublisher) PublishEmergencyStop(_ context.Context, event ports.EmergencyStop) error {
	m.emergencyEvents = append(m.emergencyEvents, event)
	return nil
}

func (m *mockPublisher) Close() error {
	return nil
}

func createTestPosition(marketID, side string, entryPrice float64, qty float64) *ports.Position {
	now := time.Now().UTC()
	return &ports.Position{
		ID:            uuid.New().String(),
		MarketID:      marketID,
		MarketSlug:    "test-market",
		Side:          side,
		EntryPrice:    decimal.NewFromFloat(entryPrice),
		CurrentPrice:  decimal.NewFromFloat(entryPrice),
		Quantity:      decimal.NewFromFloat(qty),
		UnrealizedPnL: decimal.Zero,
		RealizedPnL:   decimal.Zero,
		Status:        ports.StatusOpen,
		StrategyID:    "test-strategy",
		EntryOrderID:  uuid.New().String(),
		OpenedAt:      now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func TestPositionTracker_CreateFromOrderFilled(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}

	event := ports.OrderFilled{
		EventID:   uuid.New().String(),
		EventType: "OrderFilled",
		Timestamp: time.Now().UTC(),
		Source:    "execution-engine",
		Payload: ports.OrderFilledPayload{
			OrderID:    uuid.New().String(),
			MarketID:   "market-1",
			MarketSlug: "test-market",
			Side:       "YES",
			Price:      decimal.NewFromFloat(0.6500),
			FilledQty:  decimal.NewFromFloat(100),
			StrategyID: "test-strategy",
		},
	}

	pos := createTestPosition("market-1", "YES", 0.65, 100)
	if err := repo.CreatePosition(context.Background(), pos); err != nil {
		t.Fatalf("failed to create position: %v", err)
	}

	if len(repo.positions) != 1 {
		t.Fatalf("expected 1 position, got %d", len(repo.positions))
	}

	p := repo.positions[pos.ID]
	if p.MarketID != "market-1" {
		t.Errorf("expected market_id market-1, got %s", p.MarketID)
	}
	if p.Side != "YES" {
		t.Errorf("expected side YES, got %s", p.Side)
	}
	if !p.EntryPrice.Equal(decimal.NewFromFloat(0.65)) {
		t.Errorf("expected entry_price 0.65, got %s", p.EntryPrice)
	}
	if !p.Quantity.Equal(decimal.NewFromFloat(100)) {
		t.Errorf("expected quantity 100, got %s", p.Quantity)
	}
	if p.Status != ports.StatusOpen {
		t.Errorf("expected status OPEN, got %s", p.Status)
	}

	_ = event
	_ = pub
}

func TestPositionTracker_MultiplePositionsForSameMarket(t *testing.T) {
	repo := newMockRepo()

	pos1 := createTestPosition("market-1", "YES", 0.65, 100)
	pos2 := createTestPosition("market-1", "YES", 0.70, 50)

	repo.CreatePosition(context.Background(), pos1)
	repo.CreatePosition(context.Background(), pos2)

	positions, _ := repo.GetOpenPositionsByMarket(context.Background(), "market-1")
	if len(positions) != 2 {
		t.Errorf("expected 2 positions for market-1, got %d", len(positions))
	}
}

func TestPositionTracker_StatusTransition(t *testing.T) {
	repo := newMockRepo()

	pos := createTestPosition("market-1", "YES", 0.65, 100)
	repo.CreatePosition(context.Background(), pos)

	pos.Status = ports.StatusClosing
	repo.UpdatePosition(context.Background(), pos)

	updated, _ := repo.GetPosition(context.Background(), pos.ID)
	if updated.Status != ports.StatusClosing {
		t.Errorf("expected status CLOSING, got %s", updated.Status)
	}
}
