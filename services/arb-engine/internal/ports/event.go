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

	// Cross-market fields (optional)
	RelatedMarketID  string  `json:"related_market_id,omitempty"`
	RelationshipType string  `json:"relationship_type,omitempty"`
	NearResolution   bool    `json:"near_resolution,omitempty"`
	ConfidenceFactor float64 `json:"confidence_factor,omitempty"`
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

	// Cross-market fields (optional)
	RelatedMarketID  string  `json:"related_market_id,omitempty"`
	RelationshipType string  `json:"relationship_type,omitempty"`
	NearResolution   bool    `json:"near_resolution,omitempty"`
	ConfidenceFactor float64 `json:"confidence_factor,omitempty"`
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

type MarketRelationship struct {
	ID              string  `json:"id"`
	MarketAID       string  `json:"market_a_id"`
	MarketBID       string  `json:"market_b_id"`
	RelationshipType string `json:"relationship_type"`
	Confidence      float64 `json:"confidence"`
}

type RelationshipRepository interface {
	GetRelationships(ctx context.Context) ([]MarketRelationship, error)
	GetRelatedMarkets(ctx context.Context, marketID string) ([]MarketRelationship, error)
	UpsertRelationship(ctx context.Context, rel MarketRelationship) error
	DeleteRelationship(ctx context.Context, id string) error
	Close() error
}
