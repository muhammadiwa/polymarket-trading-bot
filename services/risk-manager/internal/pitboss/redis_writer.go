package pitboss

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type StateBuilder struct {
	dailyBudget            *DailyBudget
	marketLimits           *MarketExposure
	strategyLimits         *StrategyExposure
	emergencyStop          atomic.Bool
	mu                     sync.RWMutex // #5, #7: protects emergency + drawdown fields
	emergencyStopReason    string
	emergencyStopTimestamp *time.Time
	peakEquity             decimal.Decimal
	currentEquity          decimal.Decimal
	drawdown               decimal.Decimal
	drawdownLimit          decimal.Decimal
	capital                decimal.Decimal
	logger                 *zap.Logger
}

func NewStateBuilder(
	dailyBudget *DailyBudget,
	marketLimits *MarketExposure,
	strategyLimits *StrategyExposure,
	capital decimal.Decimal,
	logger *zap.Logger,
) *StateBuilder {
	return &StateBuilder{
		dailyBudget:    dailyBudget,
		marketLimits:   marketLimits,
		strategyLimits: strategyLimits,
		capital:        capital,
		peakEquity:     capital,
		currentEquity:  capital,
		logger:         logger,
	}
}

func (sb *StateBuilder) SetEmergencyStop(val bool) {
	sb.emergencyStop.Store(val)
	sb.mu.Lock() // #5: protect emergency fields
	defer sb.mu.Unlock()
	if !val {
		sb.emergencyStopReason = ""
		sb.emergencyStopTimestamp = nil
	}
}

func (sb *StateBuilder) SetEmergencyStopWithReason(val bool, reason string) {
	sb.emergencyStop.Store(val)
	sb.mu.Lock() // #5: protect emergency fields
	defer sb.mu.Unlock()
	if val {
		sb.emergencyStopReason = reason
		now := time.Now().UTC()
		sb.emergencyStopTimestamp = &now
	} else {
		sb.emergencyStopReason = ""
		sb.emergencyStopTimestamp = nil
	}
}

func (sb *StateBuilder) IsEmergencyStop() bool {
	return sb.emergencyStop.Load()
}

func (sb *StateBuilder) SetDrawdownState(peak, current, drawdown, limit decimal.Decimal) {
	sb.mu.Lock() // #7: protect drawdown fields
	defer sb.mu.Unlock()
	sb.peakEquity = peak
	sb.currentEquity = current
	sb.drawdown = drawdown
	sb.drawdownLimit = limit
}

func (sb *StateBuilder) GetDrawdownState() (peak, current, drawdown, limit decimal.Decimal) {
	sb.mu.RLock() // #7: read lock for drawdown fields
	defer sb.mu.RUnlock()
	return sb.peakEquity, sb.currentEquity, sb.drawdown, sb.drawdownLimit
}

func (sb *StateBuilder) BuildState() ports.PitBossState {
	marketLimits := make(map[string]ports.LimitEntry)
	for marketID, exposure := range sb.marketLimits.GetAllExposures() {
		limit := sb.marketLimits.GetLimit()
		utilization := float64(0)
		if !limit.IsZero() {
			utilization, _ = exposure.Div(limit).Float64()
		}
		marketLimits[marketID] = ports.LimitEntry{
			Exposure:    exposure,
			Limit:       limit,
			Utilization: utilization,
		}
	}

	strategyLimits := make(map[string]ports.LimitEntry)
	for strategyID, exposure := range sb.strategyLimits.GetAllExposures() {
		limit := sb.strategyLimits.GetLimit()
		utilization := float64(0)
		if !limit.IsZero() {
			utilization, _ = exposure.Div(limit).Float64()
		}
		strategyLimits[strategyID] = ports.LimitEntry{
			Exposure:    exposure,
			Limit:       limit,
			Utilization: utilization,
		}
	}

	return ports.PitBossState{
		DailyBudgetRemaining:   sb.dailyBudget.BudgetRemaining(),
		DailyLoss:              sb.dailyBudget.DailyLossValue(),
		DailyLossLimit:         sb.dailyBudget.DailyLossLimitValue(),
		Capital:                sb.capital,
		MarketLimits:           marketLimits,
		StrategyLimits:         strategyLimits,
		EmergencyStop:          sb.emergencyStop.Load(),
		EmergencyStopReason:    func() string { sb.mu.RLock(); defer sb.mu.RUnlock(); return sb.emergencyStopReason }(),
		EmergencyStopTimestamp: func() *time.Time { sb.mu.RLock(); defer sb.mu.RUnlock(); return sb.emergencyStopTimestamp }(),
		PeakEquity:             func() decimal.Decimal { sb.mu.RLock(); defer sb.mu.RUnlock(); return sb.peakEquity }(),
		CurrentEquity:          func() decimal.Decimal { sb.mu.RLock(); defer sb.mu.RUnlock(); return sb.currentEquity }(),
		Drawdown:               func() decimal.Decimal { sb.mu.RLock(); defer sb.mu.RUnlock(); return sb.drawdown }(),
		DrawdownLimit:          func() decimal.Decimal { sb.mu.RLock(); defer sb.mu.RUnlock(); return sb.drawdownLimit }(),
		UpdatedAt:              time.Now().UTC(),
	}
}
