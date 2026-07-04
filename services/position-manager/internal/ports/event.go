package ports

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

type OrderFilled struct {
	EventID   string             `json:"event_id"`
	EventType string             `json:"event_type"`
	Timestamp time.Time          `json:"timestamp"`
	Source    string             `json:"source"`
	Payload   OrderFilledPayload `json:"payload"`
}

type OrderFilledPayload struct {
	OrderID       string          `json:"order_id"`
	ClientOrderID string          `json:"client_order_id"`
	OpportunityID string          `json:"opportunity_id"`
	MarketID      string          `json:"market_id"`
	MarketSlug    string          `json:"market_slug"`
	Side          string          `json:"side"`
	Price         decimal.Decimal `json:"price"`
	FilledQty     decimal.Decimal `json:"filled_qty"`
	LatencyMs     int64           `json:"latency_ms"`
	StrategyID    string          `json:"strategy_id"`
}

type MarketPriceUpdated struct {
	EventID   string                  `json:"event_id"`
	EventType string                  `json:"event_type"`
	Timestamp time.Time               `json:"timestamp"`
	Source    string                  `json:"source"`
	Payload   MarketPriceUpdatedPayload `json:"payload"`
}

type MarketPriceUpdatedPayload struct {
	MarketID    string          `json:"market_id"`
	YESPrice    decimal.Decimal `json:"yes_price"`
	NOPrice     decimal.Decimal `json:"no_price"`
	Timestamp   time.Time       `json:"timestamp"`
}

type MarketResolved struct {
	EventID   string               `json:"event_id"`
	EventType string               `json:"event_type"`
	Timestamp time.Time            `json:"timestamp"`
	Source    string               `json:"source"`
	Payload   MarketResolvedPayload `json:"payload"`
}

type MarketResolvedPayload struct {
	MarketID string `json:"market_id"`
	Outcome  string `json:"outcome"`
}

type PositionOpened struct {
	EventID   string                `json:"event_id"`
	EventType string                `json:"event_type"`
	Timestamp time.Time             `json:"timestamp"`
	Source    string                `json:"source"`
	Payload   PositionOpenedPayload `json:"payload"`
}

type PositionOpenedPayload struct {
	PositionID   string          `json:"position_id"`
	MarketID     string          `json:"market_id"`
	MarketSlug   string          `json:"market_slug"`
	Side         string          `json:"side"`
	EntryPrice   decimal.Decimal `json:"entry_price"`
	Quantity     decimal.Decimal `json:"quantity"`
	StrategyID   string          `json:"strategy_id"`
	EntryOrderID string          `json:"entry_order_id"`
	AccountID    *string         `json:"account_id"`
}

type PositionUpdated struct {
	EventID   string                  `json:"event_id"`
	EventType string                  `json:"event_type"`
	Timestamp time.Time               `json:"timestamp"`
	Source    string                  `json:"source"`
	Payload   PositionUpdatedPayload  `json:"payload"`
}

type PositionUpdatedPayload struct {
	PositionID    string          `json:"position_id"`
	MarketID      string          `json:"market_id"`
	CurrentPrice  decimal.Decimal `json:"current_price"`
	UnrealizedPnL decimal.Decimal `json:"unrealized_pnl"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type PositionClosed struct {
	EventID   string                `json:"event_id"`
	EventType string                `json:"event_type"`
	Timestamp time.Time             `json:"timestamp"`
	Source    string                `json:"source"`
	Payload   PositionClosedPayload `json:"payload"`
}

type PositionClosedPayload struct {
	PositionID  string          `json:"position_id"`
	MarketID    string          `json:"market_id"`
	Side        string          `json:"side"`
	EntryPrice  decimal.Decimal `json:"entry_price"`
	ExitPrice   decimal.Decimal `json:"exit_price"`
	Quantity    decimal.Decimal `json:"quantity"`
	RealizedPnL decimal.Decimal `json:"realized_pnl"`
	ExitReason  string          `json:"exit_reason"`
	StrategyID  string          `json:"strategy_id"`
	AccountID   *string         `json:"account_id"`
}

type PositionReconciliationMismatch struct {
	EventID   string                                `json:"event_id"`
	EventType string                                `json:"event_type"`
	Timestamp time.Time                             `json:"timestamp"`
	Source    string                                `json:"source"`
	Payload   PositionReconciliationMismatchPayload  `json:"payload"`
}

type PositionReconciliationMismatchPayload struct {
	PositionID            string `json:"position_id"`
	InternalQuantity      string `json:"internal_quantity"`
	APIQuantity           string `json:"api_quantity"`
	InternalSide          string `json:"internal_side"`
	APISide               string `json:"api_side"`
	ConsecutiveMismatches int    `json:"consecutive_mismatches"`
	MismatchType          string `json:"mismatch_type"`
}

type RiskAlert struct {
	EventID   string           `json:"event_id"`
	EventType string           `json:"event_type"`
	Timestamp time.Time        `json:"timestamp"`
	Source    string           `json:"source"`
	Payload   RiskAlertPayload `json:"payload"`
}

type RiskAlertPayload struct {
	AlertType string `json:"alert_type"`
	Message   string `json:"message"`
	Severity  string `json:"severity"`
}

type NotificationRequest struct {
	EventID   string                    `json:"event_id"`
	EventType string                    `json:"event_type"`
	Timestamp time.Time                 `json:"timestamp"`
	Source    string                    `json:"source"`
	Payload   NotificationRequestPayload `json:"payload"`
}

type NotificationRequestPayload struct {
	Category       string `json:"category"`
	Title          string `json:"title"`
	Message        string `json:"message"`
	Channel        string `json:"channel"`
	Priority       string `json:"priority"`
	Severity       string `json:"severity"`
	BypassThrottle bool   `json:"bypass_throttle"`
}

type ExitOrderRequest struct {
	EventID   string                `json:"event_id"`
	EventType string                `json:"event_type"`
	Timestamp time.Time             `json:"timestamp"`
	Source    string                `json:"source"`
	Payload   ExitOrderRequestPayload `json:"payload"`
}

type ExitOrderRequestPayload struct {
	PositionID string          `json:"position_id"`
	MarketID   string          `json:"market_id"`
	Side       string          `json:"side"`
	Quantity   decimal.Decimal `json:"quantity"`
	OrderType  string          `json:"order_type"`
	Reason     string          `json:"reason"`
}

type EmergencyStop struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Payload   EmergencyStopPayload   `json:"payload"`
}

type EmergencyStopPayload struct {
	Reason                string                 `json:"reason"`
	Drawdown              *decimal.Decimal       `json:"drawdown,omitempty"`
	PeakEquity            *decimal.Decimal       `json:"peak_equity,omitempty"`
	CurrentEquity         *decimal.Decimal       `json:"current_equity,omitempty"`
	DailyPnL              decimal.Decimal        `json:"daily_pnl"`
	OpenOrdersCount       int                    `json:"open_orders_count"`
	ConsecutiveMismatches int                    `json:"consecutive_mismatches"`
	Context               map[string]interface{} `json:"context,omitempty"`
}

type EventPublisher interface {
	PublishPositionOpened(ctx context.Context, event PositionOpened) error
	PublishPositionUpdated(ctx context.Context, event PositionUpdated) error
	PublishPositionClosed(ctx context.Context, event PositionClosed) error
	PublishReconciliationMismatch(ctx context.Context, event PositionReconciliationMismatch) error
	PublishRiskAlert(ctx context.Context, event RiskAlert) error
	PublishNotificationRequest(ctx context.Context, event NotificationRequest) error
	PublishExitOrderRequest(ctx context.Context, event ExitOrderRequest) error
	PublishEmergencyStop(ctx context.Context, event EmergencyStop) error
	Close() error
}
