package position_manager

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/position-manager/internal/ports"
	"github.com/shopspring/decimal"
)

func TestResolution_YESPosition_YesOutcome(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	ctx := context.Background()

	pos := createTestPosition("market-1", "YES", 0.65, 100)
	repo.CreatePosition(ctx, pos)

	outcome := "YES"
	exitPrice := decimal.NewFromFloat(1.0000)

	if outcome == "YES" && pos.Side == "YES" {
		exitPrice = decimal.NewFromFloat(1.0000)
	}

	pos.RealizedPnL = exitPrice.Sub(pos.EntryPrice).Mul(pos.Quantity)
	pos.CurrentPrice = exitPrice
	pos.Status = ports.StatusSettled
	now := time.Now().UTC()
	pos.SettledAt = &now
	pos.ClosedAt = &now

	if err := repo.MoveToHistory(ctx, pos, exitPrice, "resolution"); err != nil {
		t.Fatalf("failed to move to history: %v", err)
	}

	if len(repo.history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(repo.history))
	}

	h := repo.history[0]
	expectedPnL := decimal.NewFromFloat(35.0)
	if !h.RealizedPnL.Equal(expectedPnL) {
		t.Errorf("expected realized PnL %s, got %s", expectedPnL, h.RealizedPnL)
	}
	if !h.ExitPrice.Equal(decimal.NewFromFloat(1.0)) {
		t.Errorf("expected exit price 1.0, got %s", h.ExitPrice)
	}
	if h.ExitReason != "resolution" {
		t.Errorf("expected exit reason resolution, got %s", h.ExitReason)
	}

	if len(repo.positions) != 0 {
		t.Errorf("expected 0 positions after settlement, got %d", len(repo.positions))
	}

	_ = pub
}

func TestResolution_NOPosition_YesOutcome(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	ctx := context.Background()

	pos := createTestPosition("market-1", "NO", 0.40, 100)
	repo.CreatePosition(ctx, pos)

	outcome := "YES"
	exitPrice := decimal.NewFromFloat(0.0000)

	if outcome == "YES" && pos.Side == "NO" {
		exitPrice = decimal.NewFromFloat(0.0000)
	}

	pos.RealizedPnL = exitPrice.Sub(pos.EntryPrice).Mul(pos.Quantity)
	pos.CurrentPrice = exitPrice
	pos.Status = ports.StatusSettled
	now := time.Now().UTC()
	pos.SettledAt = &now
	pos.ClosedAt = &now

	if err := repo.MoveToHistory(ctx, pos, exitPrice, "resolution"); err != nil {
		t.Fatalf("failed to move to history: %v", err)
	}

	h := repo.history[0]
	expectedPnL := decimal.NewFromFloat(-40.0)
	if !h.RealizedPnL.Equal(expectedPnL) {
		t.Errorf("expected realized PnL %s, got %s", expectedPnL, h.RealizedPnL)
	}

	_ = pub
}

func TestResolution_YESPosition_NoOutcome(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	ctx := context.Background()

	pos := createTestPosition("market-1", "YES", 0.65, 100)
	repo.CreatePosition(ctx, pos)

	outcome := "NO"
	exitPrice := decimal.NewFromFloat(0.0000)

	if outcome == "NO" && pos.Side == "YES" {
		exitPrice = decimal.NewFromFloat(0.0000)
	}

	pos.RealizedPnL = exitPrice.Sub(pos.EntryPrice).Mul(pos.Quantity)
	pos.CurrentPrice = exitPrice
	pos.Status = ports.StatusSettled
	now := time.Now().UTC()
	pos.SettledAt = &now
	pos.ClosedAt = &now

	if err := repo.MoveToHistory(ctx, pos, exitPrice, "resolution"); err != nil {
		t.Fatalf("failed to move to history: %v", err)
	}

	h := repo.history[0]
	expectedPnL := decimal.NewFromFloat(-65.0)
	if !h.RealizedPnL.Equal(expectedPnL) {
		t.Errorf("expected realized PnL %s, got %s", expectedPnL, h.RealizedPnL)
	}

	_ = pub
}

func TestResolution_NOPosition_NoOutcome(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	ctx := context.Background()

	pos := createTestPosition("market-1", "NO", 0.40, 100)
	repo.CreatePosition(ctx, pos)

	outcome := "NO"
	exitPrice := decimal.NewFromFloat(1.0000)

	if outcome == "NO" && pos.Side == "NO" {
		exitPrice = decimal.NewFromFloat(1.0000)
	}

	pos.RealizedPnL = exitPrice.Sub(pos.EntryPrice).Mul(pos.Quantity)
	pos.CurrentPrice = exitPrice
	pos.Status = ports.StatusSettled
	now := time.Now().UTC()
	pos.SettledAt = &now
	pos.ClosedAt = &now

	if err := repo.MoveToHistory(ctx, pos, exitPrice, "resolution"); err != nil {
		t.Fatalf("failed to move to history: %v", err)
	}

	h := repo.history[0]
	expectedPnL := decimal.NewFromFloat(60.0)
	if !h.RealizedPnL.Equal(expectedPnL) {
		t.Errorf("expected realized PnL %s, got %s", expectedPnL, h.RealizedPnL)
	}

	_ = pub
}

func TestResolution_BothSidesSettled(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	ctx := context.Background()

	yesPos := createTestPosition("market-1", "YES", 0.65, 100)
	noPos := createTestPosition("market-1", "NO", 0.35, 100)
	repo.CreatePosition(ctx, yesPos)
	repo.CreatePosition(ctx, noPos)

	outcome := "YES"

	yesExitPrice := decimal.NewFromFloat(1.0000)
	noExitPrice := decimal.NewFromFloat(0.0000)

	if outcome != "YES" {
		yesExitPrice = decimal.NewFromFloat(0.0000)
		noExitPrice = decimal.NewFromFloat(1.0000)
	}

	yesPos.RealizedPnL = yesExitPrice.Sub(yesPos.EntryPrice).Mul(yesPos.Quantity)
	noPos.RealizedPnL = noExitPrice.Sub(noPos.EntryPrice).Mul(noPos.Quantity)

	now := time.Now().UTC()
	yesPos.Status = ports.StatusSettled
	yesPos.SettledAt = &now
	yesPos.ClosedAt = &now
	noPos.Status = ports.StatusSettled
	noPos.SettledAt = &now
	noPos.ClosedAt = &now

	repo.MoveToHistory(ctx, yesPos, yesExitPrice, "resolution")
	repo.MoveToHistory(ctx, noPos, noExitPrice, "resolution")

	if len(repo.history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(repo.history))
	}

	if len(repo.positions) != 0 {
		t.Errorf("expected 0 positions after settlement, got %d", len(repo.positions))
	}

	for _, h := range repo.history {
		if h.Side == "YES" {
			expectedPnL := decimal.NewFromFloat(35.0)
			if !h.RealizedPnL.Equal(expectedPnL) {
				t.Errorf("YES position: expected PnL %s, got %s", expectedPnL, h.RealizedPnL)
			}
		} else {
			expectedPnL := decimal.NewFromFloat(-35.0)
			if !h.RealizedPnL.Equal(expectedPnL) {
				t.Errorf("NO position: expected PnL %s, got %s", expectedPnL, h.RealizedPnL)
			}
		}
	}

	_ = pub
}

func TestResolution_NoOpenPositions(t *testing.T) {
	repo := newMockRepo()
	ctx := context.Background()

	positions, _ := repo.GetOpenPositionsByMarket(ctx, "market-1")
	if len(positions) != 0 {
		t.Errorf("expected 0 positions for unresolved market, got %d", len(positions))
	}
}

func TestResolution_MultiplePositionsSameSide(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	ctx := context.Background()

	pos1 := createTestPosition("market-1", "YES", 0.60, 100)
	pos2 := createTestPosition("market-1", "YES", 0.70, 50)
	repo.CreatePosition(ctx, pos1)
	repo.CreatePosition(ctx, pos2)

	exitPrice := decimal.NewFromFloat(1.0000)

	for _, pos := range []*ports.Position{pos1, pos2} {
		pos.RealizedPnL = exitPrice.Sub(pos.EntryPrice).Mul(pos.Quantity)
		pos.CurrentPrice = exitPrice
		pos.Status = ports.StatusSettled
		now := time.Now().UTC()
		pos.SettledAt = &now
		pos.ClosedAt = &now
		repo.MoveToHistory(ctx, pos, exitPrice, "resolution")
	}

	if len(repo.history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(repo.history))
	}

	_ = uuid.New()
	_ = pub
}
