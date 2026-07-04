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
	Side          string          `json:"side"`
	Price         decimal.Decimal `json:"price"`
	FilledQty     decimal.Decimal `json:"filled_qty"`
	LatencyMs     int64           `json:"latency_ms"`
	StrategyID    string          `json:"strategy_id"`
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
	PositionID   string          `json:"position_id"`
	MarketID     string          `json:"market_id"`
	CurrentPrice decimal.Decimal `json:"current_price"`
	Quantity     decimal.Decimal `json:"quantity"`
	StrategyID   string          `json:"strategy_id"`
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

type RiskStateUpdated struct {
	EventID   string                    `json:"event_id"`
	EventType string                    `json:"event_type"`
	Timestamp time.Time                 `json:"timestamp"`
	Source    string                    `json:"source"`
	Payload   RiskStateUpdatedPayload   `json:"payload"`
}

type RiskStateUpdatedPayload struct {
	DailyBudgetRemaining decimal.Decimal `json:"daily_budget_remaining"`
	DailyLoss            decimal.Decimal `json:"daily_loss"`
	Capital              decimal.Decimal `json:"capital"`
	MarketCount          int             `json:"market_count"`
	StrategyCount        int             `json:"strategy_count"`
	EmergencyStop        bool            `json:"emergency_stop"`
}

type RiskDecisionLogged struct {
	EventID   string                      `json:"event_id"`
	EventType string                      `json:"event_type"`
	Timestamp time.Time                   `json:"timestamp"`
	Source    string                      `json:"source"`
	Payload   RiskDecisionLoggedPayload   `json:"payload"`
}

type RiskDecisionLoggedPayload struct {
	DecisionID           string          `json:"decision_id"`
	Decision             string          `json:"decision"`
	Reason               string          `json:"reason"`
	MarketID             *string         `json:"market_id"`
	StrategyID           *string         `json:"strategy_id"`
	TradeSize            decimal.Decimal `json:"trade_size"`
	DailyBudgetRemaining decimal.Decimal `json:"daily_budget_remaining"`
}

type DailyBudgetWarning struct {
	EventID   string                      `json:"event_id"`
	EventType string                      `json:"event_type"`
	Timestamp time.Time                   `json:"timestamp"`
	Source    string                      `json:"source"`
	Payload   DailyBudgetWarningPayload   `json:"payload"`
}

type DailyBudgetWarningPayload struct {
	DailyLoss       decimal.Decimal `json:"daily_loss"`
	DailyLossLimit  decimal.Decimal `json:"daily_loss_limit"`
	Utilization     float64         `json:"utilization"`
	BudgetRemaining decimal.Decimal `json:"budget_remaining"`
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

type EmergencyStop struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Payload   EmergencyStopPayload   `json:"payload"`
}

type EmergencyStopPayload struct {
	Reason          string                 `json:"reason"`
	Drawdown        *decimal.Decimal       `json:"drawdown,omitempty"`
	PeakEquity      *decimal.Decimal       `json:"peak_equity,omitempty"`
	CurrentEquity   *decimal.Decimal       `json:"current_equity,omitempty"`
	DailyPnL        decimal.Decimal        `json:"daily_pnl"`
	OpenOrdersCount int                    `json:"open_orders_count"`
	Context         map[string]interface{} `json:"context,omitempty"`
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
	BypassThrottle bool   `json:"bypass_throttle"`
}

type DrawdownWarning struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Payload   DrawdownWarningPayload `json:"payload"`
}

type DrawdownWarningPayload struct {
	Drawdown      decimal.Decimal `json:"drawdown"`
	DrawdownLimit decimal.Decimal `json:"drawdown_limit"`
	PeakEquity    decimal.Decimal `json:"peak_equity"`
	CurrentEquity decimal.Decimal `json:"current_equity"`
	Utilization   float64         `json:"utilization"`
}

type DrawdownReset struct {
	EventID   string               `json:"event_id"`
	EventType string               `json:"event_type"`
	Timestamp time.Time            `json:"timestamp"`
	Source    string               `json:"source"`
	Payload   DrawdownResetPayload `json:"payload"`
}

type DrawdownResetPayload struct {
	NewPeakEquity decimal.Decimal `json:"new_peak_equity"`
	PreviousPeak  decimal.Decimal `json:"previous_peak"`
}

type TradingResumed struct {
	EventID   string                `json:"event_id"`
	EventType string                `json:"event_type"`
	Timestamp time.Time             `json:"timestamp"`
	Source    string                `json:"source"`
	Payload   TradingResumedPayload `json:"payload"`
}

type TradingResumedPayload struct {
	PreviousReason string    `json:"previous_reason"`
	ResumedBy      string    `json:"resumed_by"`
	ResumedAt      time.Time `json:"resumed_at"`
}

type CancelAllOrders struct {
	EventID   string                  `json:"event_id"`
	EventType string                  `json:"event_type"`
	Timestamp time.Time               `json:"timestamp"`
	Source    string                  `json:"source"`
	Payload   CancelAllOrdersPayload  `json:"payload"`
}

type CancelAllOrdersPayload struct {
	Reason      string `json:"reason"`
	RequestedBy string `json:"requested_by"`
}

type CapitalUpdated struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Payload   CapitalUpdatedPayload  `json:"payload"`
}

type CapitalUpdatedPayload struct {
	TotalCapital decimal.Decimal `json:"total_capital"`
	Reason       string          `json:"reason"`
}

type OrderCancelled struct {
	EventID   string                `json:"event_id"`
	EventType string                `json:"event_type"`
	Timestamp time.Time             `json:"timestamp"`
	Source    string                `json:"source"`
	Payload   OrderCancelledPayload `json:"payload"`
}

type OrderCancelledPayload struct {
	OrderID  string `json:"order_id"`
	Reason   string `json:"reason"`
	Success  bool   `json:"success"`
}

type EventPublisher interface {
	PublishRiskStateUpdated(ctx context.Context, event RiskStateUpdated) error
	PublishRiskDecisionLogged(ctx context.Context, event RiskDecisionLogged) error
	PublishDailyBudgetWarning(ctx context.Context, event DailyBudgetWarning) error
	PublishRiskAlert(ctx context.Context, event RiskAlert) error
	PublishEmergencyStop(ctx context.Context, event EmergencyStop) error
	PublishNotificationRequest(ctx context.Context, event NotificationRequest) error
	PublishDrawdownWarning(ctx context.Context, event DrawdownWarning) error
	PublishDrawdownReset(ctx context.Context, event DrawdownReset) error
	PublishTradingResumed(ctx context.Context, event TradingResumed) error
	PublishCancelAllOrders(ctx context.Context, event CancelAllOrders) error
	Close() error
}

type EventSubscriber interface {
	SubscribeOrderFilled(ctx context.Context, handler func(OrderFilled)) error
	SubscribePositionOpened(ctx context.Context, handler func(PositionOpened)) error
	SubscribePositionClosed(ctx context.Context, handler func(PositionClosed)) error
	SubscribePositionUpdated(ctx context.Context, handler func(PositionUpdated)) error
	SubscribeEmergencyStop(ctx context.Context, handler func(EmergencyStop)) error
	SubscribeCapitalUpdated(ctx context.Context, handler func(CapitalUpdated)) error
	SubscribeRiskAlert(ctx context.Context, handler func(RiskAlert)) error
	Close() error
}
