package riskmanager

import (
	"testing"

	"github.com/pqap/services/risk-manager/internal/pitboss"
	"github.com/shopspring/decimal"
)

func TestStrategyLimit_ExposureTracking(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	sl := pitboss.NewStrategyLimit(capital, 0.20)

	sl.UpdateExposure("strategy_1", decimal.NewFromFloat(1000))

	exposure := sl.GetExposure("strategy_1")
	if !exposure.Equal(decimal.NewFromFloat(1000)) {
		t.Errorf("expected exposure 1000, got %s", exposure)
	}
}

func TestStrategyLimit_LimitCalculation(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	sl := pitboss.NewStrategyLimit(capital, 0.20)

	expectedLimit := decimal.NewFromFloat(2000)
	if !sl.GetLimit().Equal(expectedLimit) {
		t.Errorf("expected limit 2000, got %s", sl.GetLimit())
	}
}

func TestStrategyLimit_WouldExceed(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	sl := pitboss.NewStrategyLimit(capital, 0.20)

	sl.UpdateExposure("strategy_1", decimal.NewFromFloat(1800))

	if !sl.WouldExceed("strategy_1", decimal.NewFromFloat(300)) {
		t.Error("should exceed limit: 1800 + 300 > 2000")
	}

	if sl.WouldExceed("strategy_1", decimal.NewFromFloat(100)) {
		t.Error("should not exceed limit: 1800 + 100 <= 2000")
	}
}

func TestStrategyLimit_WouldExceedAtBoundary(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	sl := pitboss.NewStrategyLimit(capital, 0.20)

	sl.UpdateExposure("strategy_1", decimal.NewFromFloat(1800))

	if sl.WouldExceed("strategy_1", decimal.NewFromFloat(200)) {
		t.Error("should not exceed limit at exact boundary: 1800 + 200 = 2000")
	}
}

func TestStrategyLimit_AllowWhenNoExposure(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	sl := pitboss.NewStrategyLimit(capital, 0.20)

	if sl.WouldExceed("strategy_1", decimal.NewFromFloat(1000)) {
		t.Error("should not exceed limit with no existing exposure")
	}
}

func TestStrategyLimit_AddExposure(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	sl := pitboss.NewStrategyLimit(capital, 0.20)

	sl.AddExposure("strategy_1", decimal.NewFromFloat(500))
	sl.AddExposure("strategy_1", decimal.NewFromFloat(300))

	exposure := sl.GetExposure("strategy_1")
	if !exposure.Equal(decimal.NewFromFloat(800)) {
		t.Errorf("expected exposure 800, got %s", exposure)
	}
}

func TestStrategyLimit_RemoveExposure(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	sl := pitboss.NewStrategyLimit(capital, 0.20)

	sl.UpdateExposure("strategy_1", decimal.NewFromFloat(1000))
	sl.RemoveExposure("strategy_1", decimal.NewFromFloat(400))

	exposure := sl.GetExposure("strategy_1")
	if !exposure.Equal(decimal.NewFromFloat(600)) {
		t.Errorf("expected exposure 600, got %s", exposure)
	}
}

func TestStrategyLimit_RemoveExposureNeverNegative(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	sl := pitboss.NewStrategyLimit(capital, 0.20)

	sl.UpdateExposure("strategy_1", decimal.NewFromFloat(100))
	sl.RemoveExposure("strategy_1", decimal.NewFromFloat(200))

	exposure := sl.GetExposure("strategy_1")
	if exposure.IsNegative() {
		t.Errorf("exposure should not be negative, got %s", exposure)
	}
	if !exposure.IsZero() {
		t.Errorf("exposure should be zero, got %s", exposure)
	}
}

func TestStrategyLimit_Utilization(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	sl := pitboss.NewStrategyLimit(capital, 0.20)

	sl.UpdateExposure("strategy_1", decimal.NewFromFloat(1000))

	util := sl.GetUtilization("strategy_1")
	if util < 0.49 || util > 0.51 {
		t.Errorf("expected utilization ~0.50, got %f", util)
	}
}

func TestStrategyLimit_GetAllExposures(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	sl := pitboss.NewStrategyLimit(capital, 0.20)

	sl.UpdateExposure("strategy_1", decimal.NewFromFloat(1000))
	sl.UpdateExposure("strategy_2", decimal.NewFromFloat(500))

	all := sl.GetAllExposures()
	if len(all) != 2 {
		t.Errorf("expected 2 strategies, got %d", len(all))
	}
	if !all["strategy_1"].Equal(decimal.NewFromFloat(1000)) {
		t.Errorf("expected strategy_1 exposure 1000, got %s", all["strategy_1"])
	}
}
