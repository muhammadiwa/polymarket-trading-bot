package history

import (
	"time"

	"github.com/shopspring/decimal"
)

type FillStatus string

const (
	FillStatusPending     FillStatus = "PENDING"
	FillStatusPlaced      FillStatus = "PLACED"
	FillStatusFilled      FillStatus = "FILLED"
	FillStatusPartialFill FillStatus = "PARTIAL_FILL"
	FillStatusCancelled   FillStatus = "CANCELLED"
	FillStatusFailed      FillStatus = "FAILED"
	FillStatusExpired     FillStatus = "EXPIRED"
)

type TradeRecord struct {
	ID              string          `json:"id"`
	ClientOrderID   string          `json:"client_order_id"`
	StrategyID      string          `json:"strategy_id"`
	MarketID        string          `json:"market_id"`
	MarketSlug      string          `json:"market_slug"`
	Side            string          `json:"side"`
	OrderType       string          `json:"order_type"`
	Price           decimal.Decimal `json:"price"`
	Quantity        decimal.Decimal `json:"quantity"`
	FilledQuantity  decimal.Decimal `json:"filled_quantity"`
	FillStatus      FillStatus      `json:"fill_status"`
	LatencyMs       int             `json:"latency_ms"`
	PnL             decimal.Decimal `json:"pnl"`
	Fee             decimal.Decimal `json:"fee"`
	SlippagePct     decimal.Decimal `json:"slippage_pct"`
	SignalTimestamp time.Time       `json:"signal_timestamp"`
	OrderTimestamp  time.Time       `json:"order_timestamp"`
	FillTimestamp   *time.Time      `json:"fill_timestamp"`
	OpportunityID   *string         `json:"opportunity_id"`
	RiskDecision    string          `json:"risk_decision"`
	FailureReason   *string         `json:"failure_reason"`
	AccountID       *string         `json:"account_id"`
	CreatedAt       time.Time       `json:"created_at"`
}

type OrderResult struct {
	ClientOrderID   string
	OrderID         string
	OpportunityID   string
	MarketID        string
	MarketSlug      string
	Side            string
	OrderType       string
	Price           decimal.Decimal
	SignalPrice     decimal.Decimal
	Quantity        decimal.Decimal
	FilledQuantity  decimal.Decimal
	FillStatus      FillStatus
	LatencyMs       int
	Fee             decimal.Decimal
	SignalTimestamp time.Time
	OrderTimestamp  time.Time
	FillTimestamp   *time.Time
	RiskDecision    string
	FailureReason   string
	StrategyID      string
	AccountID       *string
}

type TradeRecordedPayload struct {
	TradeID        string          `json:"trade_id"`
	ClientOrderID  string          `json:"client_order_id"`
	StrategyID     string          `json:"strategy_id"`
	MarketID       string          `json:"market_id"`
	Side           string          `json:"side"`
	Price          decimal.Decimal `json:"price"`
	FilledQuantity decimal.Decimal `json:"filled_quantity"`
	FillStatus     string          `json:"fill_status"`
	PnL            decimal.Decimal `json:"pnl"`
	LatencyMs      int             `json:"latency_ms"`
}
