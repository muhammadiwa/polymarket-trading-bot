package pitboss

import (
	"sync"

	"github.com/shopspring/decimal"
)

type StrategyExposure struct {
	mu           sync.RWMutex
	exposures    map[string]decimal.Decimal
	limit        decimal.Decimal
	capital      decimal.Decimal
}

func NewStrategyLimit(capital decimal.Decimal, strategyLimitPct float64) *StrategyExposure {
	return &StrategyExposure{
		exposures: make(map[string]decimal.Decimal),
		limit:     capital.Mul(decimal.NewFromFloat(strategyLimitPct)),
		capital:   capital,
	}
}

func (s *StrategyExposure) UpdateExposure(strategyID string, exposure decimal.Decimal) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.exposures[strategyID] = exposure
}

func (s *StrategyExposure) AddExposure(strategyID string, amount decimal.Decimal) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.exposures[strategyID] = s.exposures[strategyID].Add(amount)
}

func (s *StrategyExposure) RemoveExposure(strategyID string, amount decimal.Decimal) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current := s.exposures[strategyID]
	newVal := current.Sub(amount)
	if newVal.IsNegative() {
		newVal = decimal.Zero
	}
	s.exposures[strategyID] = newVal
}

func (s *StrategyExposure) GetExposure(strategyID string) decimal.Decimal {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.exposures[strategyID]
}

func (s *StrategyExposure) GetLimit() decimal.Decimal {
	return s.limit
}

func (s *StrategyExposure) WouldExceed(strategyID string, tradeSize decimal.Decimal) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	current := s.exposures[strategyID]
	return current.Add(tradeSize).GreaterThan(s.limit)
}

func (s *StrategyExposure) GetAllExposures() map[string]decimal.Decimal {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]decimal.Decimal, len(s.exposures))
	for k, v := range s.exposures {
		result[k] = v
	}
	return result
}

func (s *StrategyExposure) GetUtilization(strategyID string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.limit.IsZero() {
		return 0
	}
	exposure := s.exposures[strategyID]
	util, _ := exposure.Div(s.limit).Float64()
	return util
}
