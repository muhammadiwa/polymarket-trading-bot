package riskmanager

import (
	"testing"
	"time"

	"github.com/pqap/services/risk-manager/internal/pitboss"
	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func TestStateBuilder_BuildState(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	logger, _ := zap.NewDevelopment()

	dailyBudget := pitboss.NewDailyBudget(capital, 0.02, 0.80)
	marketLimits := pitboss.NewMarketLimit(capital, 0.10)
	strategyLimits := pitboss.NewStrategyLimit(capital, 0.20)

	dailyBudget.RecordLoss(decimal.NewFromFloat(-50))
	marketLimits.UpdateExposure("market_1", decimal.NewFromFloat(500))
	strategyLimits.UpdateExposure("strategy_1", decimal.NewFromFloat(1000))

	sb := pitboss.NewStateBuilder(dailyBudget, marketLimits, strategyLimits, capital, logger)

	state := sb.BuildState()

	if !state.DailyBudgetRemaining.Equal(decimal.NewFromFloat(150)) {
		t.Errorf("expected daily budget remaining 150, got %s", state.DailyBudgetRemaining)
	}
	if !state.DailyLoss.Equal(decimal.NewFromFloat(50)) {
		t.Errorf("expected daily loss 50, got %s", state.DailyLoss)
	}
	if !state.DailyLossLimit.Equal(decimal.NewFromFloat(200)) {
		t.Errorf("expected daily loss limit 200, got %s", state.DailyLossLimit)
	}
	if !state.Capital.Equal(decimal.NewFromFloat(10000)) {
		t.Errorf("expected capital 10000, got %s", state.Capital)
	}
	if len(state.MarketLimits) != 1 {
		t.Errorf("expected 1 market limit, got %d", len(state.MarketLimits))
	}
	if len(state.StrategyLimits) != 1 {
		t.Errorf("expected 1 strategy limit, got %d", len(state.StrategyLimits))
	}
	if state.EmergencyStop {
		t.Error("expected emergency stop false")
	}
	if state.UpdatedAt.IsZero() {
		t.Error("expected non-zero updated_at")
	}
}

func TestStateBuilder_EmergencyStop(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	logger, _ := zap.NewDevelopment()

	dailyBudget := pitboss.NewDailyBudget(capital, 0.02, 0.80)
	marketLimits := pitboss.NewMarketLimit(capital, 0.10)
	strategyLimits := pitboss.NewStrategyLimit(capital, 0.20)

	sb := pitboss.NewStateBuilder(dailyBudget, marketLimits, strategyLimits, capital, logger)

	if sb.IsEmergencyStop() {
		t.Error("expected emergency stop false initially")
	}

	sb.SetEmergencyStop(true)
	if !sb.IsEmergencyStop() {
		t.Error("expected emergency stop true after setting")
	}

	state := sb.BuildState()
	if !state.EmergencyStop {
		t.Error("expected emergency stop true in state")
	}
}

func TestStateBuilder_MarketLimitEntries(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	logger, _ := zap.NewDevelopment()

	dailyBudget := pitboss.NewDailyBudget(capital, 0.02, 0.80)
	marketLimits := pitboss.NewMarketLimit(capital, 0.10)
	strategyLimits := pitboss.NewStrategyLimit(capital, 0.20)

	marketLimits.UpdateExposure("market_1", decimal.NewFromFloat(500))
	marketLimits.UpdateExposure("market_2", decimal.NewFromFloat(300))

	sb := pitboss.NewStateBuilder(dailyBudget, marketLimits, strategyLimits, capital, logger)

	state := sb.BuildState()

	entry1, ok := state.MarketLimits["market_1"]
	if !ok {
		t.Fatal("expected market_1 in limits")
	}
	if !entry1.Exposure.Equal(decimal.NewFromFloat(500)) {
		t.Errorf("expected market_1 exposure 500, got %s", entry1.Exposure)
	}
	if !entry1.Limit.Equal(decimal.NewFromFloat(1000)) {
		t.Errorf("expected market_1 limit 1000, got %s", entry1.Limit)
	}
	if entry1.Utilization < 0.49 || entry1.Utilization > 0.51 {
		t.Errorf("expected market_1 utilization ~0.50, got %f", entry1.Utilization)
	}
}

func TestRedisWriter_WriteAndReadState(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Redis integration test in short mode")
	}
}

func TestReconstructor_Reconstruct(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping PostgreSQL integration test in short mode")
	}
}
