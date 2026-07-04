package riskmanager

import (
	"testing"

	"github.com/pqap/services/risk-manager/internal/pitboss"
	"github.com/shopspring/decimal"
)

func TestMarketLimit_ExposureTracking(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	ml := pitboss.NewMarketLimit(capital, 0.10)

	ml.UpdateExposure("market_1", decimal.NewFromFloat(500))

	exposure := ml.GetExposure("market_1")
	if !exposure.Equal(decimal.NewFromFloat(500)) {
		t.Errorf("expected exposure 500, got %s", exposure)
	}
}

func TestMarketLimit_LimitCalculation(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	ml := pitboss.NewMarketLimit(capital, 0.10)

	expectedLimit := decimal.NewFromFloat(1000)
	if !ml.GetLimit().Equal(expectedLimit) {
		t.Errorf("expected limit 1000, got %s", ml.GetLimit())
	}
}

func TestMarketLimit_WouldExceed(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	ml := pitboss.NewMarketLimit(capital, 0.10)

	ml.UpdateExposure("market_1", decimal.NewFromFloat(900))

	if !ml.WouldExceed("market_1", decimal.NewFromFloat(200)) {
		t.Error("should exceed limit: 900 + 200 > 1000")
	}

	if ml.WouldExceed("market_1", decimal.NewFromFloat(50)) {
		t.Error("should not exceed limit: 900 + 50 <= 1000")
	}
}

func TestMarketLimit_WouldExceedAtBoundary(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	ml := pitboss.NewMarketLimit(capital, 0.10)

	ml.UpdateExposure("market_1", decimal.NewFromFloat(900))

	if ml.WouldExceed("market_1", decimal.NewFromFloat(100)) {
		t.Error("should not exceed limit at exact boundary: 900 + 100 = 1000")
	}
}

func TestMarketLimit_AllowWhenNoExposure(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	ml := pitboss.NewMarketLimit(capital, 0.10)

	if ml.WouldExceed("market_1", decimal.NewFromFloat(500)) {
		t.Error("should not exceed limit with no existing exposure")
	}
}

func TestMarketLimit_AddExposure(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	ml := pitboss.NewMarketLimit(capital, 0.10)

	ml.AddExposure("market_1", decimal.NewFromFloat(300))
	ml.AddExposure("market_1", decimal.NewFromFloat(200))

	exposure := ml.GetExposure("market_1")
	if !exposure.Equal(decimal.NewFromFloat(500)) {
		t.Errorf("expected exposure 500, got %s", exposure)
	}
}

func TestMarketLimit_RemoveExposure(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	ml := pitboss.NewMarketLimit(capital, 0.10)

	ml.UpdateExposure("market_1", decimal.NewFromFloat(500))
	ml.RemoveExposure("market_1", decimal.NewFromFloat(200))

	exposure := ml.GetExposure("market_1")
	if !exposure.Equal(decimal.NewFromFloat(300)) {
		t.Errorf("expected exposure 300, got %s", exposure)
	}
}

func TestMarketLimit_RemoveExposureNeverNegative(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	ml := pitboss.NewMarketLimit(capital, 0.10)

	ml.UpdateExposure("market_1", decimal.NewFromFloat(100))
	ml.RemoveExposure("market_1", decimal.NewFromFloat(200))

	exposure := ml.GetExposure("market_1")
	if exposure.IsNegative() {
		t.Errorf("exposure should not be negative, got %s", exposure)
	}
	if !exposure.IsZero() {
		t.Errorf("exposure should be zero, got %s", exposure)
	}
}

func TestMarketLimit_Utilization(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	ml := pitboss.NewMarketLimit(capital, 0.10)

	ml.UpdateExposure("market_1", decimal.NewFromFloat(500))

	util := ml.GetUtilization("market_1")
	if util < 0.49 || util > 0.51 {
		t.Errorf("expected utilization ~0.50, got %f", util)
	}
}

func TestMarketLimit_GetAllExposures(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	ml := pitboss.NewMarketLimit(capital, 0.10)

	ml.UpdateExposure("market_1", decimal.NewFromFloat(500))
	ml.UpdateExposure("market_2", decimal.NewFromFloat(300))

	all := ml.GetAllExposures()
	if len(all) != 2 {
		t.Errorf("expected 2 markets, got %d", len(all))
	}
	if !all["market_1"].Equal(decimal.NewFromFloat(500)) {
		t.Errorf("expected market_1 exposure 500, got %s", all["market_1"])
	}
}
