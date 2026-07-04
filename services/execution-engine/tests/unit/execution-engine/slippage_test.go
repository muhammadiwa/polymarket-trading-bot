package executor_test

import (
	"testing"

	"github.com/pqap/services/execution-engine/internal/executor"
	"github.com/shopspring/decimal"
)

func TestSlippageProtector_WithinTolerance(t *testing.T) {
	protector := executor.NewSlippageProtector(0.01)

	tests := []struct {
		name            string
		opportunityPrice string
		currentPrice     string
		wantPassed       bool
	}{
		{
			name:            "no slippage",
			opportunityPrice: "0.50",
			currentPrice:     "0.50",
			wantPassed:       true,
		},
		{
			name:            "within tolerance",
			opportunityPrice: "0.50",
			currentPrice:     "0.504",
			wantPassed:       true,
		},
		{
			name:            "exactly at tolerance",
			opportunityPrice: "0.50",
			currentPrice:     "0.505",
			wantPassed:       true,
		},
		{
			name:            "beyond tolerance",
			opportunityPrice: "0.50",
			currentPrice:     "0.51",
			wantPassed:       false,
		},
		{
			name:            "price decrease within tolerance",
			opportunityPrice: "0.50",
			currentPrice:     "0.496",
			wantPassed:       true,
		},
		{
			name:            "price decrease beyond tolerance",
			opportunityPrice: "0.50",
			currentPrice:     "0.49",
			wantPassed:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oppPrice := decimal.RequireFromString(tt.opportunityPrice)
			curPrice := decimal.RequireFromString(tt.currentPrice)

			result := protector.Check(oppPrice, curPrice)

			if result.Passed != tt.wantPassed {
				t.Errorf("Passed = %v, want %v (delta: %s)", result.Passed, tt.wantPassed, result.Delta.String())
			}
		})
	}
}

func TestSlippageProtector_ZeroPrice(t *testing.T) {
	protector := executor.NewSlippageProtector(0.01)

	result := protector.Check(decimal.Zero, decimal.NewFromFloat(0.50))

	if result.Passed {
		t.Error("expected fail for zero opportunity price")
	}
}

func TestSlippageProtector_CustomTolerance(t *testing.T) {
	protector := executor.NewSlippageProtector(0.05)

	oppPrice := decimal.RequireFromString("0.50")
	curPrice := decimal.RequireFromString("0.52")

	result := protector.Check(oppPrice, curPrice)

	if !result.Passed {
		t.Errorf("expected pass with 5%% tolerance, got fail (delta: %s)", result.Delta.String())
	}
}
