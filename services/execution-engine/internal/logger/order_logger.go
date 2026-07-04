package logger

import (
	"context"
	"time"

	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type TradeRecord struct {
	ID            string
	OrderID       string
	ClientOrderID string
	OpportunityID string
	MarketID      string
	Side          string
	Price         decimal.Decimal
	Size          decimal.Decimal
	FilledQty     decimal.Decimal
	FillStatus    string
	PnL           *decimal.Decimal
	StrategyID    string
	LatencyMs     int64
	RiskCheck     string
	SlippageCheck string
	ErrorReason    string
	AccountID     *string
	PlacedAt      time.Time
	FilledAt      *time.Time
}

type TradeRepository interface {
	InsertTrade(ctx context.Context, record TradeRecord) error
	Close() error
}

type OrderLogger struct {
	repo   TradeRepository
	logger *zap.Logger
}

func NewOrderLogger(repo TradeRepository, logger *zap.Logger) *OrderLogger {
	return &OrderLogger{
		repo:   repo,
		logger: logger,
	}
}

func (l *OrderLogger) LogOrder(ctx context.Context, order *ports.Order) error {
	record := TradeRecord{
		OrderID:       order.ID,
		ClientOrderID: order.ClientOrderID,
		OpportunityID: order.OpportunityID,
		MarketID:      order.MarketID,
		Side:          order.Side,
		Price:         order.Price,
		Size:          order.Size,
		FilledQty:     order.FilledQty,
		FillStatus:    string(order.Status),
		StrategyID:    order.StrategyID,
		LatencyMs:     order.LatencyMs,
		RiskCheck:     order.RiskCheckResult,
		SlippageCheck: order.SlippageCheck,
		ErrorReason:   order.ErrorReason,
		AccountID:     order.AccountID,
		PlacedAt:      order.PlacedAt,
		FilledAt:      order.FilledAt,
	}

	if err := l.repo.InsertTrade(ctx, record); err != nil {
		l.logger.Error("failed to log trade to PostgreSQL",
			zap.String("order_id", order.ID),
			zap.String("client_order_id", order.ClientOrderID),
			zap.Error(err),
		)
		return err
	}

	l.logger.Debug("trade logged",
		zap.String("order_id", order.ID),
		zap.String("market_id", order.MarketID),
		zap.String("side", order.Side),
		zap.String("status", string(order.Status)),
	)
	return nil
}

func (l *OrderLogger) Close() error {
	return l.repo.Close()
}
