package history

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type Repository interface {
	Insert(ctx context.Context, record *TradeRecord) error
	GetByClientOrderID(ctx context.Context, clientOrderID string) (*TradeRecord, error)
}

type PostgresRepository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewPostgresRepository(pool *pgxpool.Pool, logger *zap.Logger) *PostgresRepository {
	return &PostgresRepository{
		pool:   pool,
		logger: logger,
	}
}

func (r *PostgresRepository) Insert(ctx context.Context, record *TradeRecord) error {
	query := `
		INSERT INTO trades (
			client_order_id, strategy_id, market_id, market_slug, side,
			order_type, price, quantity, filled_quantity, fill_status,
			latency_ms, pnl, fee, slippage_pct,
			signal_timestamp, order_timestamp, fill_timestamp,
			opportunity_id, risk_decision, failure_reason, account_id
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14,
			$15, $16, $17,
			$18, $19, $20, $21
		)
		ON CONFLICT (client_order_id) DO NOTHING
		RETURNING id, created_at
	`

	err := r.pool.QueryRow(ctx, query,
		record.ClientOrderID,
		record.StrategyID,
		record.MarketID,
		record.MarketSlug,
		record.Side,
		record.OrderType,
		record.Price.String(),
		record.Quantity.String(),
		record.FilledQuantity.String(),
		string(record.FillStatus),
		record.LatencyMs,
		record.PnL.String(),
		record.Fee.String(),
		record.SlippagePct.String(),
		record.SignalTimestamp,
		record.OrderTimestamp,
		record.FillTimestamp,
		record.OpportunityID,
		record.RiskDecision,
		record.FailureReason,
		record.AccountID,
	).Scan(&record.ID, &record.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Info("trade record already exists (idempotent skip)",
				zap.String("client_order_id", record.ClientOrderID),
			)
			return nil
		}
		return fmt.Errorf("failed to insert trade record: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetByClientOrderID(ctx context.Context, clientOrderID string) (*TradeRecord, error) {
	query := `
		SELECT
			id, client_order_id, strategy_id, market_id, market_slug, side,
			order_type, price, quantity, filled_quantity, fill_status,
			latency_ms, pnl, fee, slippage_pct,
			signal_timestamp, order_timestamp, fill_timestamp,
			opportunity_id, risk_decision, failure_reason, account_id, created_at
		FROM trades
		WHERE client_order_id = $1
	`

	record := &TradeRecord{}
	var priceStr, qtyStr, filledQtyStr, pnlStr, feeStr, slippageStr string
	var fillStatus string

	err := r.pool.QueryRow(ctx, query, clientOrderID).Scan(
		&record.ID,
		&record.ClientOrderID,
		&record.StrategyID,
		&record.MarketID,
		&record.MarketSlug,
		&record.Side,
		&record.OrderType,
		&priceStr,
		&qtyStr,
		&filledQtyStr,
		&fillStatus,
		&record.LatencyMs,
		&pnlStr,
		&feeStr,
		&slippageStr,
		&record.SignalTimestamp,
		&record.OrderTimestamp,
		&record.FillTimestamp,
		&record.OpportunityID,
		&record.RiskDecision,
		&record.FailureReason,
		&record.AccountID,
		&record.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get trade by client_order_id: %w", err)
	}

	record.FillStatus = FillStatus(fillStatus)

	record.Price, _ = decimal.NewFromString(priceStr)
	record.Quantity, _ = decimal.NewFromString(qtyStr)
	record.FilledQuantity, _ = decimal.NewFromString(filledQtyStr)
	record.PnL, _ = decimal.NewFromString(pnlStr)
	record.Fee, _ = decimal.NewFromString(feeStr)
	record.SlippagePct, _ = decimal.NewFromString(slippageStr)

	return record, nil
}
