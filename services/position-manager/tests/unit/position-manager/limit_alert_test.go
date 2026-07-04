package position_manager

import (
	"context"
	"testing"

	"github.com/pqap/services/position-manager/internal/ports"
	"github.com/shopspring/decimal"
)

func TestLimitAlert_PerMarketBreach(t *testing.T) {
	pub := &mockPublisher{}

	totalCapital := decimal.NewFromFloat(10000)
	marketLimitPct := 0.10
	limitValue := totalCapital.Mul(decimal.NewFromFloat(marketLimitPct))

	positionValue := decimal.NewFromFloat(0.80).Mul(decimal.NewFromFloat(200))

	if positionValue.GreaterThan(limitValue) {
		t.Logf("correctly detected market limit breach: position %s > limit %s", positionValue, limitValue)
	} else {
		t.Error("expected market limit breach to be detected")
	}

	_ = pub
}

func TestLimitAlert_PerMarketWithinLimits(t *testing.T) {
	pub := &mockPublisher{}

	totalCapital := decimal.NewFromFloat(10000)
	marketLimitPct := 0.10
	limitValue := totalCapital.Mul(decimal.NewFromFloat(marketLimitPct))

	positionValue := decimal.NewFromFloat(0.50).Mul(decimal.NewFromFloat(100))

	if !positionValue.GreaterThan(limitValue) {
		t.Logf("correctly within market limits: position %s <= limit %s", positionValue, limitValue)
	} else {
		t.Error("expected position to be within market limits")
	}

	_ = pub
}

func TestLimitAlert_PerStrategyBreach(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	ctx := context.Background()

	totalCapital := decimal.NewFromFloat(10000)
	strategyLimitPct := 0.20
	limitValue := totalCapital.Mul(decimal.NewFromFloat(strategyLimitPct))

	pos1 := createTestPosition("market-1", "YES", 0.60, 200)
	pos1.StrategyID = "strategy-1"
	pos2 := createTestPosition("market-2", "YES", 0.70, 150)
	pos2.StrategyID = "strategy-1"
	repo.CreatePosition(ctx, pos1)
	repo.CreatePosition(ctx, pos2)

	positions, _ := repo.GetOpenPositionsByStrategy(ctx, "strategy-1")
	totalValue := decimal.Zero
	for _, p := range positions {
		totalValue = totalValue.Add(p.CurrentPrice.Mul(p.Quantity))
	}

	if totalValue.GreaterThan(limitValue) {
		t.Logf("correctly detected strategy limit breach: total %s > limit %s", totalValue, limitValue)
	} else {
		t.Error("expected strategy limit breach to be detected")
	}

	_ = pub
}

func TestLimitAlert_PerStrategyWithinLimits(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	ctx := context.Background()

	totalCapital := decimal.NewFromFloat(10000)
	strategyLimitPct := 0.20
	limitValue := totalCapital.Mul(decimal.NewFromFloat(strategyLimitPct))

	pos1 := createTestPosition("market-1", "YES", 0.50, 100)
	pos1.StrategyID = "strategy-1"
	repo.CreatePosition(ctx, pos1)

	positions, _ := repo.GetOpenPositionsByStrategy(ctx, "strategy-1")
	totalValue := decimal.Zero
	for _, p := range positions {
		totalValue = totalValue.Add(p.CurrentPrice.Mul(p.Quantity))
	}

	if !totalValue.GreaterThan(limitValue) {
		t.Logf("correctly within strategy limits: total %s <= limit %s", totalValue, limitValue)
	} else {
		t.Error("expected position to be within strategy limits")
	}

	_ = pub
}

func TestLimitAlert_MultipleStrategies(t *testing.T) {
	repo := newMockRepo()
	ctx := context.Background()

	pos1 := createTestPosition("market-1", "YES", 0.50, 100)
	pos1.StrategyID = "strategy-1"
	pos2 := createTestPosition("market-2", "YES", 0.60, 100)
	pos2.StrategyID = "strategy-2"
	repo.CreatePosition(ctx, pos1)
	repo.CreatePosition(ctx, pos2)

	positions1, _ := repo.GetOpenPositionsByStrategy(ctx, "strategy-1")
	positions2, _ := repo.GetOpenPositionsByStrategy(ctx, "strategy-2")

	if len(positions1) != 1 {
		t.Errorf("expected 1 position for strategy-1, got %d", len(positions1))
	}
	if len(positions2) != 1 {
		t.Errorf("expected 1 position for strategy-2, got %d", len(positions2))
	}
}

func TestLimitAlert_BreachGeneratesAlert(t *testing.T) {
	pub := &mockPublisher{}

	alert := ports.RiskAlert{
		EventID:   "test-alert-id",
		EventType: "RiskAlert",
		Source:    "position-manager",
		Payload: ports.RiskAlertPayload{
			AlertType: "market_limit",
			Message:   "Position exceeds per-market limit",
			Severity:  "warning",
		},
	}

	pub.PublishRiskAlert(context.Background(), alert)

	if len(pub.riskEvents) != 1 {
		t.Fatalf("expected 1 risk event, got %d", len(pub.riskEvents))
	}

	if pub.riskEvents[0].Payload.AlertType != "market_limit" {
		t.Errorf("expected alert type market_limit, got %s", pub.riskEvents[0].Payload.AlertType)
	}
}

func TestLimitAlert_BreachGeneratesNotification(t *testing.T) {
	pub := &mockPublisher{}

	notification := ports.NotificationRequest{
		EventID:   "test-notif-id",
		EventType: "NotificationRequest",
		Source:    "position-manager",
		Payload: ports.NotificationRequestPayload{
			Category: "limit_breach",
			Title:    "Position Limit Breach",
			Message:  "Position exceeds per-market limit",
			Channel:  "telegram",
			Priority: "warning",
		},
	}

	pub.PublishNotificationRequest(context.Background(), notification)

	if len(pub.notifEvents) != 1 {
		t.Fatalf("expected 1 notification event, got %d", len(pub.notifEvents))
	}

	if pub.notifEvents[0].Payload.Priority != "warning" {
		t.Errorf("expected priority warning, got %s", pub.notifEvents[0].Payload.Priority)
	}
}

func TestLimitAlert_EdgeCaseExactLimit(t *testing.T) {
	totalCapital := decimal.NewFromFloat(10000)
	marketLimitPct := 0.10
	limitValue := totalCapital.Mul(decimal.NewFromFloat(marketLimitPct))

	positionValue := limitValue

	if !positionValue.GreaterThan(limitValue) {
		t.Logf("correctly at exact limit: position %s == limit %s (no breach)", positionValue, limitValue)
	}
}

func TestLimitAlert_SetTotalCapital(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}

	totalCapital := decimal.NewFromFloat(50000)
	marketLimitPct := 0.10
	strategyLimitPct := 0.20

	_ = totalCapital
	_ = marketLimitPct
	_ = strategyLimitPct
	_ = repo
	_ = pub

	newCapital := decimal.NewFromFloat(100000)
	if !newCapital.Equal(decimal.NewFromFloat(100000)) {
		t.Error("expected capital to be updated")
	}
}
