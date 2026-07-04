package ports

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

type RiskDecision struct {
	EventID              string                 `json:"event_id"`
	Timestamp            time.Time              `json:"timestamp"`
	Decision             string                 `json:"decision"`
	Reason               string                 `json:"reason"`
	MarketID             *string                `json:"market_id"`
	StrategyID           *string                `json:"strategy_id"`
	TradeSize            decimal.Decimal        `json:"trade_size"`
	CurrentExposure      decimal.Decimal        `json:"current_exposure"`
	LimitValue           decimal.Decimal        `json:"limit_value"`
	DailyBudgetRemaining decimal.Decimal        `json:"daily_budget_remaining"`
	Capital              decimal.Decimal        `json:"capital"`
	Context              map[string]interface{} `json:"context"`
	AccountID            *string                `json:"account_id"`
}

type PitBossState struct {
	DailyBudgetRemaining  decimal.Decimal       `json:"daily_budget_remaining"`
	DailyLoss             decimal.Decimal       `json:"daily_loss"`
	DailyLossLimit        decimal.Decimal       `json:"daily_loss_limit"`
	Capital               decimal.Decimal       `json:"capital"`
	MarketLimits          map[string]LimitEntry `json:"market_limits"`
	StrategyLimits        map[string]LimitEntry `json:"strategy_limits"`
	EmergencyStop         bool                  `json:"emergency_stop"`
	EmergencyStopReason   string                `json:"emergency_stop_reason"`
	EmergencyStopTimestamp *time.Time           `json:"emergency_stop_timestamp"`
	PeakEquity            decimal.Decimal       `json:"peak_equity"`
	CurrentEquity         decimal.Decimal       `json:"current_equity"`
	Drawdown              decimal.Decimal       `json:"drawdown"`
	DrawdownLimit         decimal.Decimal       `json:"drawdown_limit"`
	UpdatedAt             time.Time             `json:"updated_at"`
}

type LimitEntry struct {
	Exposure    decimal.Decimal `json:"exposure"`
	Limit       decimal.Decimal `json:"limit"`
	Utilization float64         `json:"utilization"`
}

type RiskCheckRequest struct {
	MarketID   string          `json:"market_id"`
	StrategyID string          `json:"strategy_id"`
	TradeSize  decimal.Decimal `json:"trade_size"`
	Side       string          `json:"side"`
}

type RiskCheckResponse struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

type RiskStatePort interface {
	WriteState(state PitBossState) error
	ReadState() (*PitBossState, error)
	Close() error
}

type RiskEventRepository interface {
	InsertRiskEvent(ctx context.Context, event RiskDecision) error
	GetTodayDecisions(ctx context.Context) ([]RiskDecision, error)
	GetDailyLoss(ctx context.Context) (decimal.Decimal, error)
	GetPositionExposures(ctx context.Context) (map[string]decimal.Decimal, map[string]decimal.Decimal, error)
	Close() error
}
