package riskmanager

import (
	"testing"

	"github.com/pqap/services/risk-manager/internal/pitboss"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func TestReconstructor_NewReconstructor(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	capital := decimal.NewFromFloat(10000)

	dailyBudget := pitboss.NewDailyBudget(capital, 0.02, 0.80)
	marketLimits := pitboss.NewMarketLimit(capital, 0.10)
	strategyLimits := pitboss.NewStrategyLimit(capital, 0.20)
	stateBuilder := pitboss.NewStateBuilder(dailyBudget, marketLimits, strategyLimits, capital, logger)

	r := pitboss.NewReconstructor(nil, dailyBudget, marketLimits, strategyLimits, stateBuilder, logger)
	if r == nil {
		t.Error("expected non-nil reconstructor")
	}
}

func TestReconstructor_ReconstructWithNilRepo(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	capital := decimal.NewFromFloat(10000)

	dailyBudget := pitboss.NewDailyBudget(capital, 0.02, 0.80)
	marketLimits := pitboss.NewMarketLimit(capital, 0.10)
	strategyLimits := pitboss.NewStrategyLimit(capital, 0.20)
	stateBuilder := pitboss.NewStateBuilder(dailyBudget, marketLimits, strategyLimits, capital, logger)

	r := pitboss.NewReconstructor(nil, dailyBudget, marketLimits, strategyLimits, stateBuilder, logger)
	if r == nil {
		t.Error("expected non-nil reconstructor")
	}
}
