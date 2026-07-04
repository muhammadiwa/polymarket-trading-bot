package ports

import (
	"context"

	"github.com/shopspring/decimal"
)

type RiskDecision struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason"`
}

type RiskPort interface {
	CheckRisk(ctx context.Context, marketID, strategyID string, orderSize decimal.Decimal) (*RiskDecision, error)
}
