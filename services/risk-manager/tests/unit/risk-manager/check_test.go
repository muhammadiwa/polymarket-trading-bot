package riskmanager

import (
	"testing"

	"github.com/pqap/services/risk-manager/internal/pitboss"
	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/shopspring/decimal"
)

func TestChecker_CheckAllow(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	cap := capital
	dailyBudget := pitboss.NewDailyBudget(cap, 0.02, 0.80)
	marketLimits := pitboss.NewMarketLimit(cap, 0.10)
	strategyLimits := pitboss.NewStrategyLimit(cap, 0.20)
	logger, _ := newTestLogger()
	stateBuilder := pitboss.NewStateBuilder(dailyBudget, marketLimits, strategyLimits, cap, logger)
	riskLogger := pitboss.NewLogger(nil, logger)
	pb := pitboss.NewPitBoss(dailyBudget, marketLimits, strategyLimits, stateBuilder, riskLogger, nil, cap, logger)

	checker := pitboss.NewChecker(pb)

	req := ports.RiskCheckRequest{
		MarketID:   "market_1",
		StrategyID: "strategy_1",
		TradeSize:  decimal.NewFromFloat(500),
		Side:       "YES",
	}

	resp := checker.Check(req)
	if resp.Decision != "ALLOW" {
		t.Errorf("expected ALLOW, got %s", resp.Decision)
	}
}

func TestChecker_CheckDenyDailyLimit(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	cap := capital
	dailyBudget := pitboss.NewDailyBudget(cap, 0.02, 0.80)
	marketLimits := pitboss.NewMarketLimit(cap, 0.10)
	strategyLimits := pitboss.NewStrategyLimit(cap, 0.20)
	logger, _ := newTestLogger()
	stateBuilder := pitboss.NewStateBuilder(dailyBudget, marketLimits, strategyLimits, cap, logger)
	riskLogger := pitboss.NewLogger(nil, logger)
	pb := pitboss.NewPitBoss(dailyBudget, marketLimits, strategyLimits, stateBuilder, riskLogger, nil, cap, logger)

	checker := pitboss.NewChecker(pb)

	for i := 0; i < 10; i++ {
		pb.DailyBudget().RecordLoss(decimal.NewFromFloat(200))
	}

	req := ports.RiskCheckRequest{
		MarketID:   "market_1",
		StrategyID: "strategy_1",
		TradeSize:  decimal.NewFromFloat(100),
		Side:       "YES",
	}

	resp := checker.Check(req)
	if resp.Decision != "DENY" {
		t.Errorf("expected DENY, got %s", resp.Decision)
	}
	if resp.Reason != "daily_limit" {
		t.Errorf("expected reason 'daily_limit', got %s", resp.Reason)
	}
}

func TestChecker_CheckFromState(t *testing.T) {
	checker := pitboss.NewChecker(nil)

	state := &ports.PitBossState{
		DailyBudgetRemaining: decimal.NewFromFloat(100),
		DailyLoss:            decimal.NewFromFloat(100),
		DailyLossLimit:       decimal.NewFromFloat(200),
		Capital:              decimal.NewFromFloat(10000),
		MarketLimits: map[string]ports.LimitEntry{
			"market_1": {
				Exposure:    decimal.NewFromFloat(500),
				Limit:       decimal.NewFromFloat(1000),
				Utilization: 0.5,
			},
		},
		StrategyLimits: map[string]ports.LimitEntry{
			"strategy_1": {
				Exposure:    decimal.NewFromFloat(1000),
				Limit:       decimal.NewFromFloat(2000),
				Utilization: 0.5,
			},
		},
		EmergencyStop: false,
	}

	req := ports.RiskCheckRequest{
		MarketID:   "market_1",
		StrategyID: "strategy_1",
		TradeSize:  decimal.NewFromFloat(100),
		Side:       "YES",
	}

	resp := checker.CheckFromState(state, req)
	if resp.Decision != "ALLOW" {
		t.Errorf("expected ALLOW, got %s", resp.Decision)
	}
}

func TestChecker_CheckFromStateEmergencyStop(t *testing.T) {
	checker := pitboss.NewChecker(nil)

	state := &ports.PitBossState{
		DailyBudgetRemaining: decimal.NewFromFloat(100),
		EmergencyStop:        true,
	}

	req := ports.RiskCheckRequest{
		MarketID:   "market_1",
		StrategyID: "strategy_1",
		TradeSize:  decimal.NewFromFloat(100),
		Side:       "YES",
	}

	resp := checker.CheckFromState(state, req)
	if resp.Decision != "DENY" {
		t.Errorf("expected DENY, got %s", resp.Decision)
	}
	if resp.Reason != "emergency_stop" {
		t.Errorf("expected reason 'emergency_stop', got %s", resp.Reason)
	}
}

func TestChecker_CheckFromStateMarketLimit(t *testing.T) {
	checker := pitboss.NewChecker(nil)

	state := &ports.PitBossState{
		DailyBudgetRemaining: decimal.NewFromFloat(100),
		MarketLimits: map[string]ports.LimitEntry{
			"market_1": {
				Exposure: decimal.NewFromFloat(900),
				Limit:    decimal.NewFromFloat(1000),
			},
		},
	}

	req := ports.RiskCheckRequest{
		MarketID:   "market_1",
		StrategyID: "strategy_1",
		TradeSize:  decimal.NewFromFloat(200),
		Side:       "YES",
	}

	resp := checker.CheckFromState(state, req)
	if resp.Decision != "DENY" {
		t.Errorf("expected DENY, got %s", resp.Decision)
	}
	if resp.Reason != "market_limit" {
		t.Errorf("expected reason 'market_limit', got %s", resp.Reason)
	}
}
