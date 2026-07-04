package ports

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

type OrderStatus string

const (
	OrderStatusPending     OrderStatus = "PENDING"
	OrderStatusPlaced      OrderStatus = "PLACED"
	OrderStatusPartialFill OrderStatus = "PARTIAL_FILL"
	OrderStatusFilled      OrderStatus = "FILLED"
	OrderStatusCancelled   OrderStatus = "CANCELLED"
	OrderStatusFailed      OrderStatus = "FAILED"
)

type Order struct {
	ID              string          `json:"id"`
	ClientOrderID   string          `json:"client_order_id"`
	OpportunityID   string          `json:"opportunity_id"`
	MarketID        string          `json:"market_id"`
	MarketSlug      string          `json:"market_slug"`
	Side            string          `json:"side"`
	Price           decimal.Decimal `json:"price"`
	Size            decimal.Decimal `json:"size"`
	FilledQty       decimal.Decimal `json:"filled_qty"`
	RemainingQty    decimal.Decimal `json:"remaining_qty"`
	Status          OrderStatus     `json:"status"`
	TimeInForce     string          `json:"time_in_force"`
	LatencyMs       int64           `json:"latency_ms"`
	RiskCheckResult string          `json:"risk_check_result"`
	SlippageCheck   string          `json:"slippage_check"`
	ErrorReason     string          `json:"error_reason"`
	StrategyID      string          `json:"strategy_id"`
	AccountID       *string         `json:"account_id"`
	PlacedAt        time.Time       `json:"placed_at"`
	FilledAt        *time.Time      `json:"filled_at"`
}

type OrderRequest struct {
	OpportunityID string          `json:"opportunity_id"`
	MarketID      string          `json:"market_id"`
	Side          string          `json:"side"`
	Price         decimal.Decimal `json:"price"`
	Size          decimal.Decimal `json:"size"`
	TimeInForce   string          `json:"time_in_force"`
	StrategyID    string          `json:"strategy_id"`
}

type OrderResponse struct {
	OrderID       string          `json:"order_id"`
	ClientOrderID string          `json:"client_order_id"`
	Status        string          `json:"status"`
	FilledQty     decimal.Decimal `json:"filled_qty"`
	RemainingQty  decimal.Decimal `json:"remaining_qty"`
	Price         decimal.Decimal `json:"price"`
}

type OrderStatusResponse struct {
	OrderID      string          `json:"order_id"`
	Status       string          `json:"status"`
	FilledQty    decimal.Decimal `json:"filled_qty"`
	RemainingQty decimal.Decimal `json:"remaining_qty"`
	Price        decimal.Decimal `json:"price"`
}

type OrderPort interface {
	PlaceOrder(req OrderRequest, clientOrderID string) (*OrderResponse, error)
	CancelOrder(ctx context.Context, orderID string) error
	GetOrderStatus(orderID string) (*OrderStatusResponse, error)
}
