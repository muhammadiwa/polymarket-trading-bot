package ports

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

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

type OrderPlaced struct {
	EventID   string             `json:"event_id"`
	EventType string             `json:"event_type"`
	Timestamp time.Time          `json:"timestamp"`
	Source    string             `json:"source"`
	Payload   OrderPlacedPayload `json:"payload"`
}

type OrderPlacedPayload struct {
	OrderID        string          `json:"order_id"`
	ClientOrderID  string          `json:"client_order_id"`
	OpportunityID  string          `json:"opportunity_id"`
	MarketID       string          `json:"market_id"`
	Side           string          `json:"side"`
	Price          decimal.Decimal `json:"price"`
	CurrentPrice   decimal.Decimal `json:"current_price"`
	Size           decimal.Decimal `json:"size"`
	StrategyID     string          `json:"strategy_id"`
}

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

type OrderPartialFill struct {
	EventID   string                   `json:"event_id"`
	EventType string                   `json:"event_type"`
	Timestamp time.Time                `json:"timestamp"`
	Source    string                   `json:"source"`
	Payload   OrderPartialFillPayload  `json:"payload"`
}

type OrderPartialFillPayload struct {
	OrderID       string          `json:"order_id"`
	ClientOrderID string          `json:"client_order_id"`
	OpportunityID string          `json:"opportunity_id"`
	MarketID      string          `json:"market_id"`
	Side          string          `json:"side"`
	Price         decimal.Decimal `json:"price"`
	FilledQty     decimal.Decimal `json:"filled_qty"`
	RemainingQty  decimal.Decimal `json:"remaining_qty"`
	StrategyID    string          `json:"strategy_id"`
}

type OrderCancelled struct {
	EventID   string               `json:"event_id"`
	EventType string               `json:"event_type"`
	Timestamp time.Time            `json:"timestamp"`
	Source    string               `json:"source"`
	Payload   OrderCancelledPayload `json:"payload"`
}

type OrderCancelledPayload struct {
	OrderID       string `json:"order_id"`
	ClientOrderID string `json:"client_order_id"`
	OpportunityID string `json:"opportunity_id"`
	MarketID      string `json:"market_id"`
	Reason        string `json:"reason"`
	StrategyID    string `json:"strategy_id"`
}

type OrderFailed struct {
	EventID   string             `json:"event_id"`
	EventType string             `json:"event_type"`
	Timestamp time.Time          `json:"timestamp"`
	Source    string             `json:"source"`
	Payload   OrderFailedPayload `json:"payload"`
}

type OrderFailedPayload struct {
	OrderID       string `json:"order_id"`
	ClientOrderID string `json:"client_order_id"`
	OpportunityID string `json:"opportunity_id"`
	MarketID      string `json:"market_id"`
	Reason        string `json:"reason"`
	ErrorDetail   string `json:"error_detail"`
	StrategyID    string `json:"strategy_id"`
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

type AtomicLegFailed struct {
	EventID   string                  `json:"event_id"`
	EventType string                  `json:"event_type"`
	Timestamp time.Time               `json:"timestamp"`
	Source    string                  `json:"source"`
	Payload   AtomicLegFailedPayload  `json:"payload"`
}

type AtomicLegFailedPayload struct {
	PairID               string          `json:"pair_id"`
	OpportunityID        string          `json:"opportunity_id"`
	MarketID             string          `json:"market_id"`
	FailedLeg            string          `json:"failed_leg"`
	FailedOrderID        string          `json:"failed_order_id"`
	FailureReason        string          `json:"failure_reason"`
	SuccessfulLeg        string          `json:"successful_leg"`
	SuccessfulOrderID    string          `json:"successful_order_id"`
	SuccessfulFilledQty  decimal.Decimal `json:"successful_filled_qty"`
	CancelledLeg         string          `json:"cancelled_leg"`
	CancelledOrderID     string          `json:"cancelled_order_id"`
	StrategyID           string          `json:"strategy_id"`
}

type CircuitBreakerTripped struct {
	EventID   string                       `json:"event_id"`
	EventType string                       `json:"event_type"`
	Timestamp time.Time                    `json:"timestamp"`
	Source    string                       `json:"source"`
	Payload   CircuitBreakerTrippedPayload `json:"payload"`
}

type CircuitBreakerTrippedPayload struct {
	ConsecutiveErrors int       `json:"consecutive_errors"`
	LastError         string    `json:"last_error"`
	LastErrorTime     time.Time `json:"last_error_time"`
	CooldownSeconds   int       `json:"cooldown_seconds"`
	Message           string    `json:"message"`
}

type CircuitBreakerResumed struct {
	EventID   string                      `json:"event_id"`
	EventType string                      `json:"event_type"`
	Timestamp time.Time                   `json:"timestamp"`
	Source    string                      `json:"source"`
	Payload   CircuitBreakerResumedPayload `json:"payload"`
}

type CircuitBreakerResumedPayload struct {
	Reason   string `json:"reason"`
	UserID   string `json:"user_id"`
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
	Severity       string `json:"severity"`
	Channel        string `json:"channel"`
	Priority       string `json:"priority"`
	BypassThrottle bool   `json:"bypass_throttle"`
}

type TradeRecorded struct {
	EventID   string                `json:"event_id"`
	EventType string                `json:"event_type"`
	Timestamp time.Time             `json:"timestamp"`
	Source    string                `json:"source"`
	Payload   TradeRecordedPayload  `json:"payload"`
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

type EventPublisher interface {
	PublishOrderPlaced(ctx context.Context, event OrderPlaced) error
	PublishOrderFilled(ctx context.Context, event OrderFilled) error
	PublishOrderPartialFill(ctx context.Context, event OrderPartialFill) error
	PublishOrderCancelled(ctx context.Context, event OrderCancelled) error
	PublishOrderFailed(ctx context.Context, event OrderFailed) error
	PublishRiskAlert(ctx context.Context, event RiskAlert) error
	PublishAtomicLegFailed(ctx context.Context, event AtomicLegFailed) error
	PublishCircuitBreakerTripped(ctx context.Context, event CircuitBreakerTripped) error
	PublishCircuitBreakerResumed(ctx context.Context, event CircuitBreakerResumed) error
	PublishNotificationRequest(ctx context.Context, event NotificationRequest) error
	PublishTradeRecorded(ctx context.Context, event TradeRecorded) error
	Close() error
}

type EventSubscriber interface {
	Subscribe(ctx context.Context, handler func(OpportunityDetected)) error
	Close() error
}
