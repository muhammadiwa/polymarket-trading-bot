package ports

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

type RiskEvent struct {
	ID          string
	MarketID    string
	StrategyID  string
	OrderSize   decimal.Decimal
	Allowed     bool
	Reason      string
	LatencyMs   int64
	CreatedAt   time.Time
}

type RiskEventRepository interface {
	InsertRiskEvent(ctx context.Context, event RiskEvent) error
}
