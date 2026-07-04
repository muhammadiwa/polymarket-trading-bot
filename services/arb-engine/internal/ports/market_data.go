package ports

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

type MarketPriceUpdated struct {
	MarketID        string          `json:"market_id"`
	YESPrice        decimal.Decimal `json:"yes_price"`
	NOPrice         decimal.Decimal `json:"no_price"`
	Spread          decimal.Decimal `json:"spread"`
	Volume          decimal.Decimal `json:"volume_24h"`
	LiquidityDepth  decimal.Decimal `json:"liquidity_depth"`
	IsStale         bool            `json:"is_stale"`
	Timestamp       time.Time       `json:"last_updated"`
}

type MarketDataPort interface {
	Subscribe(ctx context.Context, handler func(MarketPriceUpdated)) error
	Close() error
}
