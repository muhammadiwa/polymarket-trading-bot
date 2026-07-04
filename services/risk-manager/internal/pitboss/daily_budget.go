package pitboss

import (
	"sync"
	"time"

	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/shopspring/decimal"
)

type DailyBudget struct {
	mu                  sync.RWMutex
	dailyLoss           decimal.Decimal
	dailyLossLimit      decimal.Decimal
	warningThreshold    float64
	currentDate         string
	capital             decimal.Decimal
}

func NewDailyBudget(capital decimal.Decimal, dailyLossLimitPct float64, warningThreshold float64) *DailyBudget {
	return &DailyBudget{
		dailyLoss:        decimal.Zero,
		dailyLossLimit:   capital.Mul(decimal.NewFromFloat(dailyLossLimitPct)),
		warningThreshold: warningThreshold,
		currentDate:      time.Now().UTC().Format("2006-01-02"),
		capital:          capital,
	}
}

func (db *DailyBudget) RecordLoss(loss decimal.Decimal) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.checkDayReset()
	if loss.IsNegative() {
		db.dailyLoss = db.dailyLoss.Add(loss.Abs())
	}
}

func (db *DailyBudget) RecordPnL(pnl decimal.Decimal) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.checkDayReset()
	if pnl.IsNegative() {
		db.dailyLoss = db.dailyLoss.Add(pnl.Abs())
	}
}

func (db *DailyBudget) BudgetRemaining() decimal.Decimal {
	db.mu.RLock()
	defer db.mu.RUnlock()

	remaining := db.dailyLossLimit.Sub(db.dailyLoss)
	if remaining.IsNegative() {
		return decimal.Zero
	}
	return remaining
}

func (db *DailyBudget) DailyLossValue() decimal.Decimal {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.dailyLoss
}

func (db *DailyBudget) DailyLossLimitValue() decimal.Decimal {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.dailyLossLimit
}

func (db *DailyBudget) IsExhausted() bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.dailyLoss.GreaterThanOrEqual(db.dailyLossLimit)
}

func (db *DailyBudget) ShouldWarn() bool {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.dailyLossLimit.IsZero() {
		return false
	}
	utilization := db.dailyLoss.Div(db.dailyLossLimit)
	return utilization.GreaterThanOrEqual(decimal.NewFromFloat(db.warningThreshold))
}

func (db *DailyBudget) Utilization() float64 {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.dailyLossLimit.IsZero() {
		return 0
	}
	util, _ := db.dailyLoss.Div(db.dailyLossLimit).Float64()
	return util
}

func (db *DailyBudget) Reset() {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.dailyLoss = decimal.Zero
	db.currentDate = time.Now().UTC().Format("2006-01-02")
}

func (db *DailyBudget) SetDailyLossFromDB(loss decimal.Decimal) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.checkDayReset()
	db.dailyLoss = loss
}

func (db *DailyBudget) checkDayReset() {
	today := time.Now().UTC().Format("2006-01-02")
	if today != db.currentDate {
		db.dailyLoss = decimal.Zero
		db.currentDate = today
	}
}
