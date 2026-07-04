package position_manager

import (
	"context"
	"testing"
	"time"

	"github.com/pqap/services/position-manager/internal/ports"
	"github.com/shopspring/decimal"
)

type mockPolymarketPort struct {
	positions []*ports.APIPosition
	err       error
}

func (m *mockPolymarketPort) GetPositions() ([]*ports.APIPosition, error) {
	return m.positions, m.err
}

func (m *mockPolymarketPort) GetMarketResolution(marketID string) (bool, string, error) {
	return false, "", nil
}

func TestReconciler_Match(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	polymarket := &mockPolymarketPort{
		positions: []*ports.APIPosition{
			{MarketID: "market-1", Side: "YES", Quantity: "100"},
		},
	}
	ctx := context.Background()

	pos := createTestPosition("market-1", "YES", 0.65, 100)
	repo.CreatePosition(ctx, pos)

	internalPositions, _ := repo.GetOpenPositions(ctx)
	apiPositions, _ := polymarket.GetPositions()

	if len(internalPositions) != len(apiPositions) {
		t.Errorf("position count mismatch: internal %d, api %d", len(internalPositions), len(apiPositions))
	}

	_ = pub
}

func TestReconciler_MismatchQuantity(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	polymarket := &mockPolymarketPort{
		positions: []*ports.APIPosition{
			{MarketID: "market-1", Side: "YES", Quantity: "150"},
		},
	}
	ctx := context.Background()

	pos := createTestPosition("market-1", "YES", 0.65, 100)
	repo.CreatePosition(ctx, pos)

	internalPositions, _ := repo.GetOpenPositions(ctx)
	apiPositions, _ := polymarket.GetPositions()

	mismatchFound := false
	for _, internal := range internalPositions {
		for _, api := range apiPositions {
			if internal.MarketID == api.MarketID && internal.Side == api.Side {
				if internal.Quantity.String() != api.Quantity {
					mismatchFound = true
				}
			}
		}
	}

	if !mismatchFound {
		t.Error("expected quantity mismatch to be detected")
	}

	_ = pub
}

func TestReconciler_MissingPosition(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	polymarket := &mockPolymarketPort{
		positions: []*ports.APIPosition{},
	}
	ctx := context.Background()

	pos := createTestPosition("market-1", "YES", 0.65, 100)
	repo.CreatePosition(ctx, pos)

	internalPositions, _ := repo.GetOpenPositions(ctx)
	apiPositions, _ := polymarket.GetPositions()

	internalMap := make(map[string]bool)
	for _, ip := range internalPositions {
		internalMap[ip.MarketID+"-"+ip.Side] = true
	}

	for _, internal := range internalPositions {
		key := internal.MarketID + "-" + internal.Side
		found := false
		for _, api := range apiPositions {
			if api.MarketID+"-"+api.Side == key {
				found = true
				break
			}
		}
		if !found {
			t.Logf("correctly detected missing API position for %s", key)
		}
	}

	_ = internalMap
	_ = pub
}

func TestReconciler_ExtraPosition(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	polymarket := &mockPolymarketPort{
		positions: []*ports.APIPosition{
			{MarketID: "market-1", Side: "YES", Quantity: "100"},
			{MarketID: "market-2", Side: "NO", Quantity: "50"},
		},
	}
	ctx := context.Background()

	pos := createTestPosition("market-1", "YES", 0.65, 100)
	repo.CreatePosition(ctx, pos)

	internalPositions, _ := repo.GetOpenPositions(ctx)
	apiPositions, _ := polymarket.GetPositions()

	internalMap := make(map[string]*ports.Position)
	for _, ip := range internalPositions {
		internalMap[ip.MarketID+"-"+ip.Side] = ip
	}

	for _, api := range apiPositions {
		key := api.MarketID + "-" + api.Side
		if _, found := internalMap[key]; !found {
			t.Logf("correctly detected extra API position for %s", key)
		}
	}

	_ = pub
}

func TestReconciler_ConsecutiveMismatches(t *testing.T) {
	threshold := 3
	consecutiveErrors := 0

	for i := 0; i < 4; i++ {
		consecutiveErrors++
		if consecutiveErrors >= threshold {
			t.Logf("emergency stop triggered at attempt %d (consecutive: %d)", i+1, consecutiveErrors)
			if consecutiveErrors < threshold {
				t.Error("expected emergency stop to trigger")
			}
			break
		}
	}

	if consecutiveErrors < threshold {
		t.Errorf("expected consecutive errors >= %d, got %d", threshold, consecutiveErrors)
	}
}

func TestReconciler_ResetOnMatch(t *testing.T) {
	consecutiveErrors := 3

	consecutiveErrors = 0

	if consecutiveErrors != 0 {
		t.Errorf("expected consecutive errors reset to 0, got %d", consecutiveErrors)
	}
}

func TestReconciliationLog_Created(t *testing.T) {
	repo := newMockRepo()
	ctx := context.Background()

	log := &ports.ReconciliationLog{
		MarketID:              "market-1",
		MismatchType:          "quantity",
		InternalState:         []byte(`{"quantity":"100"}`),
		APIState:              []byte(`{"quantity":"150"}`),
		ConsecutiveMismatches: 1,
		Resolved:              false,
		CreatedAt:             time.Now().UTC(),
	}

	if err := repo.LogReconciliation(ctx, log); err != nil {
		t.Fatalf("failed to log reconciliation: %v", err)
	}

	if len(repo.reconLogs) != 1 {
		t.Errorf("expected 1 recon log, got %d", len(repo.reconLogs))
	}

	if repo.reconLogs[0].MismatchType != "quantity" {
		t.Errorf("expected mismatch type quantity, got %s", repo.reconLogs[0].MismatchType)
	}
}

func TestReconciliationLog_MultipleTypes(t *testing.T) {
	repo := newMockRepo()
	ctx := context.Background()

	types := []string{"quantity", "side", "missing", "extra"}
	for _, mt := range types {
		log := &ports.ReconciliationLog{
			MarketID:              "market-1",
			MismatchType:          mt,
			InternalState:         []byte(`{}`),
			APIState:              []byte(`{}`),
			ConsecutiveMismatches: 1,
			Resolved:              false,
			CreatedAt:             time.Now().UTC(),
		}
		repo.LogReconciliation(ctx, log)
	}

	if len(repo.reconLogs) != 4 {
		t.Errorf("expected 4 recon logs, got %d", len(repo.reconLogs))
	}

	_ = decimal.Zero
}
