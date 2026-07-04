package ports

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

type Opportunity struct {
	ID              string          `json:"id"`
	MarketID        string          `json:"market_id"`
	YESPrice        decimal.Decimal `json:"yes_price"`
	NOPrice         decimal.Decimal `json:"no_price"`
	Spread          decimal.Decimal `json:"spread"`
	Liquidity       decimal.Decimal `json:"liquidity"`
	FillProbability decimal.Decimal `json:"fill_probability"`
	Score           decimal.Decimal `json:"score"`
	FilterReason    string          `json:"filter_reason"`
	DetectedAt      time.Time       `json:"detected_at"`
	LatencyMs       int64           `json:"latency_ms"`
}

type OpportunityDetected struct {
	EventID   string             `json:"event_id"`
	EventType string             `json:"event_type"`
	Timestamp time.Time          `json:"timestamp"`
	Source    string             `json:"source"`
	Payload   OpportunityPayload `json:"payload"`
}

type OpportunityPayload struct {
	OpportunityID   string          `json:"opportunity_id"`
	MarketID        string          `json:"market_id"`
	YESPrice        decimal.Decimal `json:"yes_price"`
	NOPrice         decimal.Decimal `json:"no_price"`
	Spread          decimal.Decimal `json:"spread"`
	Score           decimal.Decimal `json:"score"`
	FillProbability decimal.Decimal `json:"fill_probability"`
	Liquidity       decimal.Decimal `json:"liquidity"`
	StrategyID      string          `json:"strategy_id"`
}

type EventPort interface {
	PublishOpportunityDetected(ctx context.Context, event OpportunityDetected) error
	Close() error
}

type OpportunityLogger interface {
	Log(ctx context.Context, opp Opportunity) error
	GetHistoricalFillRate(ctx context.Context, marketID string, days int) (decimal.Decimal, int, error)
	Close() error
}
