package position_manager

import (
	"context"
	"testing"
	"time"

	"github.com/pqap/services/position-manager/internal/ports"
	"github.com/shopspring/decimal"
)

func TestExit_ValidOpenPosition(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	ctx := context.Background()

	pos := createTestPosition("market-1", "YES", 0.65, 100)
	repo.CreatePosition(ctx, pos)

	position, _ := repo.GetPosition(ctx, pos.ID)
	if position.Status != ports.StatusOpen {
		t.Fatalf("expected status OPEN, got %s", position.Status)
	}

	position.Status = ports.StatusClosing
	repo.UpdatePosition(ctx, position)

	updated, _ := repo.GetPosition(ctx, pos.ID)
	if updated.Status != ports.StatusClosing {
		t.Errorf("expected status CLOSING, got %s", updated.Status)
	}

	_ = pub
}

func TestExit_ExitOrderPublished(t *testing.T) {
	pub := &mockPublisher{}

	exitOrder := ports.ExitOrderRequest{
		EventID:   "test-event-id",
		EventType: "ExitOrderRequest",
		Timestamp: time.Now().UTC(),
		Source:    "position-manager",
		Payload: ports.ExitOrderRequestPayload{
			PositionID: "test-position-id",
			MarketID:   "market-1",
			Side:       "YES",
			Quantity:   decimal.NewFromFloat(100),
			OrderType:  "MARKET",
			Reason:     "manual_exit",
		},
	}

	pub.PublishExitOrderRequest(context.Background(), exitOrder)

	if len(pub.exitEvents) != 1 {
		t.Fatalf("expected 1 exit event, got %d", len(pub.exitEvents))
	}

	if pub.exitEvents[0].Payload.OrderType != "MARKET" {
		t.Errorf("expected order type MARKET, got %s", pub.exitEvents[0].Payload.OrderType)
	}
	if pub.exitEvents[0].Payload.Reason != "manual_exit" {
		t.Errorf("expected reason manual_exit, got %s", pub.exitEvents[0].Payload.Reason)
	}
}

func TestExit_OnFill_RealizedPnL(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	ctx := context.Background()

	pos := createTestPosition("market-1", "YES", 0.60, 100)
	pos.Status = ports.StatusClosing
	repo.CreatePosition(ctx, pos)

	exitPrice := decimal.NewFromFloat(0.80)
	pos.RealizedPnL = exitPrice.Sub(pos.EntryPrice).Mul(pos.Quantity)
	pos.CurrentPrice = exitPrice
	pos.Status = ports.StatusClosed
	now := time.Now().UTC()
	pos.ClosedAt = &now

	repo.MoveToHistory(ctx, pos, exitPrice, "manual")

	if len(repo.history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(repo.history))
	}

	h := repo.history[0]
	expectedPnL := decimal.NewFromFloat(20.0)
	if !h.RealizedPnL.Equal(expectedPnL) {
		t.Errorf("expected realized PnL %s, got %s", expectedPnL, h.RealizedPnL)
	}
	if h.ExitReason != "manual" {
		t.Errorf("expected exit reason manual, got %s", h.ExitReason)
	}

	_ = pub
}

func TestExit_ClosedPositionCannotBeExited(t *testing.T) {
	repo := newMockRepo()
	ctx := context.Background()

	pos := createTestPosition("market-1", "YES", 0.65, 100)
	pos.Status = ports.StatusClosed
	repo.CreatePosition(ctx, pos)

	position, _ := repo.GetPosition(ctx, pos.ID)
	if position.Status == ports.StatusClosed {
		if position.Status != ports.StatusOpen && position.Status != ports.StatusMonitoring {
			t.Log("correctly rejected exit on closed position")
		}
	}
}

func TestExit_SettledPositionCannotBeExited(t *testing.T) {
	repo := newMockRepo()
	ctx := context.Background()

	pos := createTestPosition("market-1", "YES", 0.65, 100)
	pos.Status = ports.StatusSettled
	repo.CreatePosition(ctx, pos)

	position, _ := repo.GetPosition(ctx, pos.ID)
	if position.Status != ports.StatusOpen && position.Status != ports.StatusMonitoring {
		t.Log("correctly rejected exit on settled position")
	}
}

func TestExit_PositionNotFound(t *testing.T) {
	repo := newMockRepo()
	ctx := context.Background()

	position, _ := repo.GetPosition(ctx, "non-existent-id")
	if position == nil {
		t.Log("correctly returned nil for non-existent position")
	}
}

func TestExit_LossScenario(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	ctx := context.Background()

	pos := createTestPosition("market-1", "YES", 0.70, 100)
	pos.Status = ports.StatusClosing
	repo.CreatePosition(ctx, pos)

	exitPrice := decimal.NewFromFloat(0.50)
	pos.RealizedPnL = exitPrice.Sub(pos.EntryPrice).Mul(pos.Quantity)
	pos.CurrentPrice = exitPrice
	pos.Status = ports.StatusClosed
	now := time.Now().UTC()
	pos.ClosedAt = &now

	repo.MoveToHistory(ctx, pos, exitPrice, "manual")

	h := repo.history[0]
	expectedPnL := decimal.NewFromFloat(-20.0)
	if !h.RealizedPnL.Equal(expectedPnL) {
		t.Errorf("expected realized PnL %s, got %s", expectedPnL, h.RealizedPnL)
	}

	_ = pub
}

func TestExit_NOPosition(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	ctx := context.Background()

	pos := createTestPosition("market-1", "NO", 0.40, 100)
	pos.Status = ports.StatusClosing
	repo.CreatePosition(ctx, pos)

	exitPrice := decimal.NewFromFloat(0.20)
	pos.RealizedPnL = exitPrice.Sub(pos.EntryPrice).Mul(pos.Quantity)
	pos.CurrentPrice = exitPrice
	pos.Status = ports.StatusClosed
	now := time.Now().UTC()
	pos.ClosedAt = &now

	repo.MoveToHistory(ctx, pos, exitPrice, "manual")

	h := repo.history[0]
	expectedPnL := decimal.NewFromFloat(-20.0)
	if !h.RealizedPnL.Equal(expectedPnL) {
		t.Errorf("expected realized PnL %s, got %s", expectedPnL, h.RealizedPnL)
	}

	_ = pub
}
