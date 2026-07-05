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
	correlationExceeded    map[string]bool
	batasiWinPaused        bool
	winStreakCurrent       int
	winStreakThreshold     int
	metabolicAlert         bool
	strategyWeights        map[string]decimal.Decimal // #2: per-strategy capital allocation weights
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
		dailyBudget:         dailyBudget,
		marketLimits:        marketLimits,
		strategyLimits:      strategyLimits,
		capital:             capital,
		peakEquity:          capital,
		currentEquity:       capital,
		correlationExceeded: make(map[string]bool),
		logger:              logger,
	}
}

// #20: SetEmergencyStop delegates to SetEmergencyStopWithReason for complete state
func (sb *StateBuilder) SetEmergencyStop(val bool) {
	if val {
		sb.SetEmergencyStopWithReason(val, "")
	} else {
		sb.SetEmergencyStopWithReason(val, "")
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

func (sb *StateBuilder) SetCorrelationExceeded(groupID string, exceeded bool) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	if sb.correlationExceeded == nil {
		sb.correlationExceeded = make(map[string]bool)
	}
	sb.correlationExceeded[groupID] = exceeded
}

func (sb *StateBuilder) SetBatasiWinPaused(paused bool) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.batasiWinPaused = paused
}

func (sb *StateBuilder) SetWinStreakState(current, threshold int) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.winStreakCurrent = current
	sb.winStreakThreshold = threshold
}

func (sb *StateBuilder) SetMetabolicAlert(alert bool) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.metabolicAlert = alert
}

// #2: SetStrategyWeights sets per-strategy capital allocation weights
func (sb *StateBuilder) SetStrategyWeights(weights map[string]decimal.Decimal) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.strategyWeights = weights
}

// #15: BuildState holds a single RLock for entire snapshot instead of per-field locks.
// #4: Returns defensive copy of CorrelationExceeded map.
func (sb *StateBuilder) BuildState() ports.PitBossState {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

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

	// #4: Defensive copy of correlationExceeded map
	corrCopy := make(map[string]bool, len(sb.correlationExceeded))
	for k, v := range sb.correlationExceeded {
		corrCopy[k] = v
	}

	// #2: Defensive copy of strategy weights
	weightsCopy := make(map[string]decimal.Decimal, len(sb.strategyWeights))
	for k, v := range sb.strategyWeights {
		weightsCopy[k] = v
	}

	var emergencyTimestamp *time.Time
	if sb.emergencyStopTimestamp != nil {
		ts := *sb.emergencyStopTimestamp
		emergencyTimestamp = &ts
	}

	return ports.PitBossState{
		DailyBudgetRemaining:   sb.dailyBudget.BudgetRemaining(),
		DailyLoss:              sb.dailyBudget.DailyLossValue(),
		DailyLossLimit:         sb.dailyBudget.DailyLossLimitValue(),
		Capital:                sb.capital,
		MarketLimits:           marketLimits,
		StrategyLimits:         strategyLimits,
		StrategyWeights:        weightsCopy,
		EmergencyStop:          sb.emergencyStop.Load(),
		EmergencyStopReason:    sb.emergencyStopReason,
		EmergencyStopTimestamp: emergencyTimestamp,
		PeakEquity:             sb.peakEquity,
		CurrentEquity:          sb.currentEquity,
		Drawdown:               sb.drawdown,
		DrawdownLimit:          sb.drawdownLimit,
		CorrelationExceeded:    corrCopy,
		BatasiWinPaused:        sb.batasiWinPaused,
		WinStreakCurrent:       sb.winStreakCurrent,
		WinStreakThreshold:     sb.winStreakThreshold,
		MetabolicAlert:         sb.metabolicAlert,
		UpdatedAt:              time.Now().UTC(),
	}
}
