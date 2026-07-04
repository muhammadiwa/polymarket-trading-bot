package pitboss

import (
	"sync"

	"github.com/shopspring/decimal"
)

type MarketExposure struct {
	mu           sync.RWMutex
	exposures    map[string]decimal.Decimal
	limit        decimal.Decimal
	capital      decimal.Decimal
}

func NewMarketLimit(capital decimal.Decimal, marketLimitPct float64) *MarketExposure {
	return &MarketExposure{
		exposures: make(map[string]decimal.Decimal),
		limit:     capital.Mul(decimal.NewFromFloat(marketLimitPct)),
		capital:   capital,
	}
}

func (m *MarketExposure) UpdateExposure(marketID string, exposure decimal.Decimal) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exposures[marketID] = exposure
}

func (m *MarketExposure) AddExposure(marketID string, amount decimal.Decimal) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exposures[marketID] = m.exposures[marketID].Add(amount)
}

func (m *MarketExposure) RemoveExposure(marketID string, amount decimal.Decimal) {
	m.mu.Lock()
	defer m.mu.Unlock()

	current := m.exposures[marketID]
	newVal := current.Sub(amount)
	if newVal.IsNegative() {
		newVal = decimal.Zero
	}
	m.exposures[marketID] = newVal
}

func (m *MarketExposure) GetExposure(marketID string) decimal.Decimal {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.exposures[marketID]
}

func (m *MarketExposure) GetLimit() decimal.Decimal {
	return m.limit
}

func (m *MarketExposure) WouldExceed(marketID string, tradeSize decimal.Decimal) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	current := m.exposures[marketID]
	return current.Add(tradeSize).GreaterThan(m.limit)
}

func (m *MarketExposure) GetAllExposures() map[string]decimal.Decimal {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]decimal.Decimal, len(m.exposures))
	for k, v := range m.exposures {
		result[k] = v
	}
	return result
}

func (m *MarketExposure) GetUtilization(marketID string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.limit.IsZero() {
		return 0
	}
	exposure := m.exposures[marketID]
	util, _ := exposure.Div(m.limit).Float64()
	return util
}
