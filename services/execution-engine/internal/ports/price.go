package ports

import (
	"context"

	"github.com/shopspring/decimal"
)

type MarketPricePort interface {
	GetCurrentPrice(ctx context.Context, marketID string) (decimal.Decimal, error)
	GetLiquidityDepth(ctx context.Context, marketID string) decimal.Decimal
}
