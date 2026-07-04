package riskmanager

import (
	"testing"

	"github.com/pqap/services/risk-manager/internal/pitboss"
	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func newTestPitBoss(capital float64) *pitboss.PitBoss {
	cap := decimal.NewFromFloat(capital)
	dailyBudget := pitboss.NewDailyBudget(cap, 0.02, 0.80)
	marketLimits := pitboss.NewMarketLimit(cap, 0.10)
	strategyLimits := pitboss.NewStrategyLimit(cap, 0.20)
	logger, _ := zap.NewDevelopment()
	stateBuilder := pitboss.NewStateBuilder(dailyBudget, marketLimits, strategyLimits, cap, logger)
	riskLogger := pitboss.NewLogger(nil, logger)
	drawdownTracker := pitboss.NewDrawdownTracker(cap, 0.10, 0.80, nil, logger)
	return pitboss.NewPitBoss(dailyBudget, marketLimits, strategyLimits, stateBuilder, riskLogger, nil, cap, drawdownTracker, logger)
}

func TestPitBoss_AllowWhenWithinLimits(t *testing.T) {
	pb := newTestPitBoss(10000)

	req := ports.RiskCheckRequest{
		MarketID:   "market_1",
		StrategyID: "strategy_1",
		TradeSize:  decimal.NewFromFloat(500),
		Side:       "YES",
	}

	decision := pb.Evaluate(req)
	if decision.Decision != "ALLOW" {
		t.Errorf("expected ALLOW, got %s (reason: %s)", decision.Decision, decision.Reason)
	}
	if decision.Reason != "approved" {
		t.Errorf("expected reason 'approved', got %s", decision.Reason)
	}
}

func TestPitBoss_DenyWhenDailyBudgetExhausted(t *testing.T) {
	pb := newTestPitBoss(10000)

	for i := 0; i < 10; i++ {
		pb.DailyBudget().RecordLoss(decimal.NewFromFloat(200))
	}

	req := ports.RiskCheckRequest{
		MarketID:   "market_1",
		StrategyID: "strategy_1",
		TradeSize:  decimal.NewFromFloat(100),
		Side:       "YES",
	}

	decision := pb.Evaluate(req)
	if decision.Decision != "DENY" {
		t.Errorf("expected DENY, got %s", decision.Decision)
	}
	if decision.Reason != "daily_limit" {
		t.Errorf("expected reason 'daily_limit', got %s", decision.Reason)
	}
}

func TestPitBoss_DenyWhenMarketLimitExceeded(t *testing.T) {
	pb := newTestPitBoss(10000)

	pb.MarketLimits().UpdateExposure("market_1", decimal.NewFromFloat(900))

	req := ports.RiskCheckRequest{
		MarketID:   "market_1",
		StrategyID: "strategy_1",
		TradeSize:  decimal.NewFromFloat(200),
		Side:       "YES",
	}

	decision := pb.Evaluate(req)
	if decision.Decision != "DENY" {
		t.Errorf("expected DENY, got %s", decision.Decision)
	}
	if decision.Reason != "market_limit" {
		t.Errorf("expected reason 'market_limit', got %s", decision.Reason)
	}
}

func TestPitBoss_DenyWhenStrategyLimitExceeded(t *testing.T) {
	pb := newTestPitBoss(10000)

	pb.StrategyLimits().UpdateExposure("strategy_1", decimal.NewFromFloat(1800))

	req := ports.RiskCheckRequest{
		MarketID:   "market_1",
		StrategyID: "strategy_1",
		TradeSize:  decimal.NewFromFloat(300),
		Side:       "YES",
	}

	decision := pb.Evaluate(req)
	if decision.Decision != "DENY" {
		t.Errorf("expected DENY, got %s", decision.Decision)
	}
	if decision.Reason != "strategy_limit" {
		t.Errorf("expected reason 'strategy_limit', got %s", decision.Reason)
	}
}

func TestPitBoss_DenyWhenEmergencyStopActive(t *testing.T) {
	pb := newTestPitBoss(10000)

	pb.SetEmergencyStop(true)

	req := ports.RiskCheckRequest{
		MarketID:   "market_1",
		StrategyID: "strategy_1",
		TradeSize:  decimal.NewFromFloat(100),
		Side:       "YES",
	}

	decision := pb.Evaluate(req)
	if decision.Decision != "DENY" {
		t.Errorf("expected DENY, got %s", decision.Decision)
	}
	if decision.Reason != "emergency_stop" {
		t.Errorf("expected reason 'emergency_stop', got %s", decision.Reason)
	}
}

func TestPitBoss_AllDecisionIncludesCorrectFields(t *testing.T) {
	pb := newTestPitBoss(10000)

	req := ports.RiskCheckRequest{
		MarketID:   "market_1",
		StrategyID: "strategy_1",
		TradeSize:  decimal.NewFromFloat(500),
		Side:       "YES",
	}

	decision := pb.Evaluate(req)

	if decision.EventID == "" {
		t.Error("expected non-empty event_id")
	}
	if decision.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if decision.Capital.IsZero() {
		t.Error("expected non-zero capital")
	}
	if decision.MarketID == nil || *decision.MarketID != "market_1" {
		t.Error("expected market_id to be 'market_1'")
	}
	if decision.StrategyID == nil || *decision.StrategyID != "strategy_1" {
		t.Error("expected strategy_id to be 'strategy_1'")
	}
}

func TestPitBoss_DenyDecisionIncludesExposureAndLimit(t *testing.T) {
	pb := newTestPitBoss(10000)

	pb.MarketLimits().UpdateExposure("market_1", decimal.NewFromFloat(900))

	req := ports.RiskCheckRequest{
		MarketID:   "market_1",
		StrategyID: "strategy_1",
		TradeSize:  decimal.NewFromFloat(200),
		Side:       "YES",
	}

	decision := pb.Evaluate(req)

	if decision.CurrentExposure.IsZero() {
		t.Error("expected non-zero current_exposure for market_limit denial")
	}
	if decision.LimitValue.IsZero() {
		t.Error("expected non-zero limit_value for market_limit denial")
	}
}
