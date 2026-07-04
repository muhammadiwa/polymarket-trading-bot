package executor

import (
	"github.com/shopspring/decimal"
)

type SlippageResult struct {
	Passed bool
	Delta  decimal.Decimal
}

type SlippageProtector struct {
	tolerance decimal.Decimal
}

func NewSlippageProtector(tolerance float64) *SlippageProtector {
	return &SlippageProtector{
		tolerance: decimal.NewFromFloat(tolerance),
	}
}

func (sp *SlippageProtector) Check(opportunityPrice, currentPrice decimal.Decimal) *SlippageResult {
	if opportunityPrice.IsZero() {
		return &SlippageResult{Passed: false, Delta: decimal.Zero}
	}

	delta := currentPrice.Sub(opportunityPrice).Abs().Div(opportunityPrice)

	return &SlippageResult{
		Passed: delta.LessThanOrEqual(sp.tolerance),
		Delta:  delta,
	}
}
