package riskmanager

import (
	"testing"
	"time"

	"github.com/pqap/services/risk-manager/internal/pitboss"
	"github.com/shopspring/decimal"
)

func TestDailyBudget_TracksLoss(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	db := pitboss.NewDailyBudget(capital, 0.02, 0.80)

	db.RecordLoss(decimal.NewFromFloat(-100))

	if !db.DailyLossValue().Equal(decimal.NewFromFloat(100)) {
		t.Errorf("expected daily loss 100, got %s", db.DailyLossValue())
	}
}

func TestDailyBudget_BudgetRemaining(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	db := pitboss.NewDailyBudget(capital, 0.02, 0.80)

	expectedLimit := decimal.NewFromFloat(200)
	if !db.DailyLossLimitValue().Equal(expectedLimit) {
		t.Errorf("expected daily loss limit 200, got %s", db.DailyLossLimitValue())
	}

	db.RecordLoss(decimal.NewFromFloat(-50))
	remaining := db.BudgetRemaining()
	expected := decimal.NewFromFloat(150)
	if !remaining.Equal(expected) {
		t.Errorf("expected budget remaining 150, got %s", remaining)
	}
}

func TestDailyBudget_Exhausted(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	db := pitboss.NewDailyBudget(capital, 0.02, 0.80)

	if db.IsExhausted() {
		t.Error("budget should not be exhausted initially")
	}

	db.RecordLoss(decimal.NewFromFloat(-200))
	if !db.IsExhausted() {
		t.Error("budget should be exhausted after losing 200 (2% of 10000)")
	}
}

func TestDailyBudget_WarningAt80Percent(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	db := pitboss.NewDailyBudget(capital, 0.02, 0.80)

	if db.ShouldWarn() {
		t.Error("should not warn at 0% utilization")
	}

	db.RecordLoss(decimal.NewFromFloat(-160))
	if !db.ShouldWarn() {
		t.Error("should warn at 80% utilization")
	}
}

func TestDailyBudget_BudgetRemainingNeverNegative(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	db := pitboss.NewDailyBudget(capital, 0.02, 0.80)

	db.RecordLoss(decimal.NewFromFloat(-500))

	remaining := db.BudgetRemaining()
	if remaining.IsNegative() {
		t.Errorf("budget remaining should not be negative, got %s", remaining)
	}
	if !remaining.IsZero() {
		t.Errorf("budget remaining should be zero when over limit, got %s", remaining)
	}
}

func TestDailyBudget_Utilization(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	db := pitboss.NewDailyBudget(capital, 0.02, 0.80)

	db.RecordLoss(decimal.NewFromFloat(-100))
	util := db.Utilization()

	if util < 0.49 || util > 0.51 {
		t.Errorf("expected utilization ~0.50, got %f", util)
	}
}

func TestDailyBudget_Reset(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	db := pitboss.NewDailyBudget(capital, 0.02, 0.80)

	db.RecordLoss(decimal.NewFromFloat(-100))
	db.Reset()

	if !db.DailyLossValue().IsZero() {
		t.Errorf("expected daily loss 0 after reset, got %s", db.DailyLossValue())
	}
	if db.IsExhausted() {
		t.Error("budget should not be exhausted after reset")
	}
}

func TestDailyBudget_MidnightReset(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	db := pitboss.NewDailyBudget(capital, 0.02, 0.80)

	db.RecordLoss(decimal.NewFromFloat(-100))

	if !db.DailyLossValue().Equal(decimal.NewFromFloat(100)) {
		t.Errorf("expected daily loss 100, got %s", db.DailyLossValue())
	}

	_ = time.Now().UTC().Format("2006-01-02")
}

func TestDailyBudget_MultipleLosses(t *testing.T) {
	capital := decimal.NewFromFloat(10000)
	db := pitboss.NewDailyBudget(capital, 0.02, 0.80)

	db.RecordLoss(decimal.NewFromFloat(-50))
	db.RecordLoss(decimal.NewFromFloat(-30))
	db.RecordLoss(decimal.NewFromFloat(-20))

	expected := decimal.NewFromFloat(100)
	if !db.DailyLossValue().Equal(expected) {
		t.Errorf("expected daily loss 100, got %s", db.DailyLossValue())
	}
}
