package ports

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

type PositionStatus string

const (
	StatusOpen       PositionStatus = "OPEN"
	StatusMonitoring PositionStatus = "MONITORING"
	StatusClosing    PositionStatus = "CLOSING"
	StatusClosed     PositionStatus = "CLOSED"
	StatusSettled    PositionStatus = "SETTLED"
)

var ErrPositionNotFound = errors.New("position not found")

type Position struct {
	ID            string          `json:"id"`
	MarketID      string          `json:"market_id"`
	MarketSlug    string          `json:"market_slug"`
	Side          string          `json:"side"`
	EntryPrice    decimal.Decimal `json:"entry_price"`
	CurrentPrice  decimal.Decimal `json:"current_price"`
	Quantity      decimal.Decimal `json:"quantity"`
	UnrealizedPnL decimal.Decimal `json:"unrealized_pnl"`
	RealizedPnL   decimal.Decimal `json:"realized_pnl"`
	Status        PositionStatus  `json:"status"`
	StrategyID    string          `json:"strategy_id"`
	EntryOrderID  string          `json:"entry_order_id"`
	ExitOrderID   *string         `json:"exit_order_id"`
	OpenedAt      time.Time       `json:"opened_at"`
	ClosedAt      *time.Time      `json:"closed_at"`
	SettledAt     *time.Time      `json:"settled_at"`
	AccountID     *string         `json:"account_id"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type PositionHistory struct {
	ID           string          `json:"id"`
	MarketID     string          `json:"market_id"`
	MarketSlug   string          `json:"market_slug"`
	Side         string          `json:"side"`
	EntryPrice   decimal.Decimal `json:"entry_price"`
	ExitPrice    decimal.Decimal `json:"exit_price"`
	Quantity     decimal.Decimal `json:"quantity"`
	RealizedPnL  decimal.Decimal `json:"realized_pnl"`
	StrategyID   string          `json:"strategy_id"`
	EntryOrderID string          `json:"entry_order_id"`
	ExitOrderID  *string         `json:"exit_order_id"`
	ExitReason   string          `json:"exit_reason"`
	OpenedAt     time.Time       `json:"opened_at"`
	ClosedAt     time.Time       `json:"closed_at"`
	AccountID    *string         `json:"account_id"`
}

type ReconciliationLog struct {
	ID                    string    `json:"id"`
	PositionID            *string   `json:"position_id"`
	MarketID              string    `json:"market_id"`
	MismatchType          string    `json:"mismatch_type"`
	InternalState         []byte    `json:"internal_state"`
	APIState              []byte    `json:"api_state"`
	ConsecutiveMismatches int       `json:"consecutive_mismatches"`
	Resolved              bool      `json:"resolved"`
	CreatedAt             time.Time `json:"created_at"`
}

type ReconciliationState struct {
	LastReconciledAt      time.Time `json:"last_reconciled_at"`
	ConsecutiveMismatches int       `json:"consecutive_mismatches"`
	TotalMismatches       int64     `json:"total_mismatches"`
	TotalReconciliations  int64     `json:"total_reconciliations"`
}

type APIPosition struct {
	MarketID string `json:"market_id"`
	Side     string `json:"side"`
	Quantity string `json:"size"`
	Price    string `json:"price"`
}

type PositionRepository interface {
	CreatePosition(ctx context.Context, p *Position) error
	UpdatePosition(ctx context.Context, p *Position) error
	UpdatePositionStatus(ctx context.Context, id string, expectedStatus, newStatus PositionStatus) (bool, error)
	GetPosition(ctx context.Context, id string) (*Position, error)
	GetOpenPositions(ctx context.Context) ([]*Position, error)
	GetOpenPositionsByMarket(ctx context.Context, marketID string) ([]*Position, error)
	GetOpenPositionsByStrategy(ctx context.Context, strategyID string) ([]*Position, error)
	GetActivePositionCount(ctx context.Context) (int, error)
	MoveToHistory(ctx context.Context, p *Position, exitPrice decimal.Decimal, exitReason string) error
	GetHistory(ctx context.Context, limit, offset int) ([]*PositionHistory, error)
	LogReconciliation(ctx context.Context, log *ReconciliationLog) error
	GetReconciliationState(ctx context.Context) (*ReconciliationState, error)
	IncrementMismatchCount(ctx context.Context) error
	ResetMismatchCount(ctx context.Context) error
}

type PolymarketPort interface {
	GetPositions() ([]*APIPosition, error)
	GetMarketResolution(marketID string) (resolved bool, outcome string, err error)
}
