package riskmanager

import (
	"testing"

	"github.com/pqap/services/risk-manager/internal/pitboss"
	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func newTestStateBuilder() *pitboss.StateBuilder {
	capital := decimal.NewFromFloat(10000)
	logger, _ := zap.NewDevelopment()
	dailyBudget := pitboss.NewDailyBudget(capital, 0.02, 0.80)
	marketLimits := pitboss.NewMarketLimit(capital, 0.10)
	strategyLimits := pitboss.NewStrategyLimit(capital, 0.20)
	return pitboss.NewStateBuilder(dailyBudget, marketLimits, strategyLimits, capital, logger)
}

func TestPitBossState_CorrelationExceeded(t *testing.T) {
	sb := newTestStateBuilder()

	sb.SetCorrelationExceeded("group_1", true)
	state := sb.BuildState()

	if len(state.CorrelationExceeded) == 0 {
		t.Error("expected correlation_exceeded to have entries")
	}
	if !state.CorrelationExceeded["group_1"] {
		t.Error("expected group_1 to be exceeded")
	}
}

func TestPitBossState_CorrelationNotExceeded(t *testing.T) {
	sb := newTestStateBuilder()

	sb.SetCorrelationExceeded("group_1", false)
	state := sb.BuildState()

	if state.CorrelationExceeded["group_1"] {
		t.Error("expected group_1 to not be exceeded")
	}
}

func TestPitBossState_BatasiWinPaused(t *testing.T) {
	sb := newTestStateBuilder()

	sb.SetBatasiWinPaused(true)
	state := sb.BuildState()

	if !state.BatasiWinPaused {
		t.Error("expected batasi_win_paused to be true")
	}
}

func TestPitBossState_BatasiWinNotPaused(t *testing.T) {
	sb := newTestStateBuilder()

	sb.SetBatasiWinPaused(false)
	state := sb.BuildState()

	if state.BatasiWinPaused {
		t.Error("expected batasi_win_paused to be false")
	}
}

func TestPitBossState_MetabolicAlert(t *testing.T) {
	sb := newTestStateBuilder()

	sb.SetMetabolicAlert(true)
	state := sb.BuildState()

	if !state.MetabolicAlert {
		t.Error("expected metabolic_alert to be true")
	}
}

func TestPitBossState_MetabolicNoAlert(t *testing.T) {
	sb := newTestStateBuilder()

	sb.SetMetabolicAlert(false)
	state := sb.BuildState()

	if state.MetabolicAlert {
		t.Error("expected metabolic_alert to be false")
	}
}

func TestPitBossState_AllNewFields(t *testing.T) {
	sb := newTestStateBuilder()

	sb.SetCorrelationExceeded("election_group", true)
	sb.SetBatasiWinPaused(true)
	sb.SetMetabolicAlert(true)

	state := sb.BuildState()

	if !state.CorrelationExceeded["election_group"] {
		t.Error("expected election_group correlation exceeded")
	}
	if !state.BatasiWinPaused {
		t.Error("expected batasi_win_paused")
	}
	if !state.MetabolicAlert {
		t.Error("expected metabolic_alert")
	}
}

func TestChecker_CheckFromState_DenyBatasiWinPaused(t *testing.T) {
	pb := newTestPitBoss(10000)
	checker := pitboss.NewChecker(pb)

	state := &ports.PitBossState{
		EmergencyStop:       false,
		BatasiWinPaused:     true,
		DailyBudgetRemaining: decimal.NewFromFloat(1000),
	}

	req := ports.RiskCheckRequest{
		MarketID:   "market_1",
		StrategyID: "strategy_1",
		TradeSize:  decimal.NewFromFloat(100),
	}

	result := checker.CheckFromState(state, req)
	if result.Decision != "DENY" {
		t.Errorf("expected DENY, got %s", result.Decision)
	}
	if result.Reason != "batasi_win_paused" {
		t.Errorf("expected reason 'batasi_win_paused', got %s", result.Reason)
	}
}

func TestChecker_CheckFromState_DenyCorrelationExceeded(t *testing.T) {
	pb := newTestPitBoss(10000)
	checker := pitboss.NewChecker(pb)

	state := &ports.PitBossState{
		EmergencyStop:       false,
		BatasiWinPaused:     false,
		DailyBudgetRemaining: decimal.NewFromFloat(1000),
		CorrelationExceeded: map[string]bool{"group_1": true},
	}

	req := ports.RiskCheckRequest{
		MarketID:   "market_1",
		StrategyID: "strategy_1",
		TradeSize:  decimal.NewFromFloat(100),
	}

	result := checker.CheckFromState(state, req)
	if result.Decision != "DENY" {
		t.Errorf("expected DENY, got %s", result.Decision)
	}
	if result.Reason != "correlation_limit_exceeded" {
		t.Errorf("expected reason 'correlation_limit_exceeded', got %s", result.Reason)
	}
}

func TestChecker_CheckFromState_AllowWhenNoNewFlags(t *testing.T) {
	pb := newTestPitBoss(10000)
	checker := pitboss.NewChecker(pb)

	state := &ports.PitBossState{
		EmergencyStop:       false,
		BatasiWinPaused:     false,
		DailyBudgetRemaining: decimal.NewFromFloat(1000),
		CorrelationExceeded: map[string]bool{},
		MarketLimits:        map[string]ports.LimitEntry{},
		StrategyLimits:      map[string]ports.LimitEntry{},
	}

	req := ports.RiskCheckRequest{
		MarketID:   "market_1",
		StrategyID: "strategy_1",
		TradeSize:  decimal.NewFromFloat(100),
	}

	result := checker.CheckFromState(state, req)
	if result.Decision != "ALLOW" {
		t.Errorf("expected ALLOW, got %s (reason: %s)", result.Decision, result.Reason)
	}
}
