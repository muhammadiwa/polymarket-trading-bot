package logger_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/arb-engine/internal/logger"
	"github.com/pqap/services/arb-engine/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type mockRepo struct {
	inserted []ports.Opportunity
}

func (m *mockRepo) Insert(_ context.Context, opp ports.Opportunity) error {
	m.inserted = append(m.inserted, opp)
	return nil
}

func (m *mockRepo) GetHistoricalFillRate(_ context.Context, _ string, _ int) (float64, int, error) {
	return 0.5, 0, nil
}

func (m *mockRepo) Close() error { return nil }

func TestLogOpportunity(t *testing.T) {
	repo := &mockRepo{}
	log := zap.NewNop()
	oppLogger := logger.NewOpportunityLogger(repo, log)

	opp := ports.Opportunity{
		ID:              uuid.New().String(),
		MarketID:        "test-market",
		YESPrice:        decimal.RequireFromString("0.55"),
		NOPrice:         decimal.RequireFromString("0.40"),
		Spread:          decimal.RequireFromString("0.05"),
		Liquidity:       decimal.RequireFromString("0.8"),
		FillProbability: decimal.RequireFromString("0.7"),
		Score:           decimal.RequireFromString("0.028"),
		FilterReason:    "",
		DetectedAt:      time.Now().UTC(),
		LatencyMs:       5,
	}

	err := oppLogger.Log(context.Background(), opp)
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	if len(repo.inserted) != 1 {
		t.Fatalf("expected 1 inserted opportunity, got %d", len(repo.inserted))
	}
	if repo.inserted[0].ID != opp.ID {
		t.Errorf("inserted ID = %s, want %s", repo.inserted[0].ID, opp.ID)
	}
	if repo.inserted[0].MarketID != opp.MarketID {
		t.Errorf("inserted MarketID = %s, want %s", repo.inserted[0].MarketID, opp.MarketID)
	}
}

func TestLogFilteredOpportunity(t *testing.T) {
	repo := &mockRepo{}
	log := zap.NewNop()
	oppLogger := logger.NewOpportunityLogger(repo, log)

	opp := ports.Opportunity{
		ID:           uuid.New().String(),
		MarketID:     "test-market",
		YESPrice:     decimal.RequireFromString("0.50"),
		NOPrice:      decimal.RequireFromString("0.495"),
		Spread:       decimal.RequireFromString("0.005"),
		Score:        decimal.RequireFromString("0.002"),
		FilterReason: "below_threshold",
		DetectedAt:   time.Now().UTC(),
		LatencyMs:    2,
	}

	err := oppLogger.Log(context.Background(), opp)
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	if repo.inserted[0].FilterReason != "below_threshold" {
		t.Errorf("FilterReason = %s, want below_threshold", repo.inserted[0].FilterReason)
	}
}
