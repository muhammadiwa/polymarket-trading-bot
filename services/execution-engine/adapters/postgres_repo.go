package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pqap/services/execution-engine/internal/logger"
	"github.com/pqap/services/execution-engine/internal/ports"
	"go.uber.org/zap"
)

type PostgresRepo struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewPostgresRepo(url string, logger *zap.Logger) (*PostgresRepo, error) {
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("invalid PostgreSQL URL: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	repo := &PostgresRepo{
		pool:   pool,
		logger: logger,
	}

	if err := repo.ensureSchema(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ensure schema: %w", err)
	}

	return repo, nil
}

func (r *PostgresRepo) ensureSchema(ctx context.Context) error {
	// NOTE: trades table is managed exclusively by migrations (002_create_trades.up.sql).
	// Do NOT create it here to avoid dual-schema conflicts.

	riskEventsQuery := `
		CREATE TABLE IF NOT EXISTS risk_events (
			id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			market_id   TEXT NOT NULL,
			strategy_id TEXT NOT NULL,
			order_size  NUMERIC(10,8) NOT NULL,
			allowed     BOOLEAN NOT NULL,
			reason      TEXT NOT NULL DEFAULT '',
			latency_ms  INTEGER NOT NULL,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`

	_, err := r.pool.Exec(ctx, riskEventsQuery)
	if err != nil {
		return fmt.Errorf("failed to create risk_events table: %w", err)
	}

	partialFillsQuery := `
		CREATE TABLE IF NOT EXISTS atomic_partial_fills (
			id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			pair_id         UUID NOT NULL,
			leg             TEXT NOT NULL,
			filled_qty      NUMERIC(10,8) NOT NULL,
			remaining_qty   NUMERIC(10,8) NOT NULL,
			fill_price      NUMERIC(10,4) NOT NULL,
			order_id        UUID NOT NULL,
			client_order_id UUID NOT NULL,
			market_id       TEXT NOT NULL,
			strategy_id     TEXT NOT NULL,
			account_id      UUID DEFAULT NULL,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`

	_, err = r.pool.Exec(ctx, partialFillsQuery)
	if err != nil {
		return fmt.Errorf("failed to create atomic_partial_fills table: %w", err)
	}

	circuitBreakerEventsQuery := `
		CREATE TABLE IF NOT EXISTS circuit_breaker_events (
			id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			event_type      TEXT NOT NULL,
			state_from      TEXT NOT NULL,
			state_to        TEXT NOT NULL,
			consecutive_errors INTEGER,
			last_error      TEXT,
			cooldown_seconds INTEGER,
			user_id         TEXT DEFAULT NULL,
			reason          TEXT DEFAULT '',
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`

	_, err = r.pool.Exec(ctx, circuitBreakerEventsQuery)
	if err != nil {
		return fmt.Errorf("failed to create circuit_breaker_events table: %w", err)
	}

	// NOTE: trades indexes are managed by migrations, not here.

	_, err = r.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_risk_events_market_id ON risk_events (market_id)`)
	if err != nil {
		r.logger.Warn("index creation (may already exist)", zap.Error(err))
	}

	_, err = r.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_risk_events_created_at ON risk_events (created_at)`)
	if err != nil {
		r.logger.Warn("index creation (may already exist)", zap.Error(err))
	}

	_, err = r.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_partial_fills_pair_id ON atomic_partial_fills (pair_id)`)
	if err != nil {
		r.logger.Warn("index creation (may already exist)", zap.Error(err))
	}

	_, err = r.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_cb_events_created_at ON circuit_breaker_events (created_at)`)
	if err != nil {
		r.logger.Warn("index creation (may already exist)", zap.Error(err))
	}

	return nil
}

func (r *PostgresRepo) InsertTrade(ctx context.Context, record logger.TradeRecord) error {
	query := `
		INSERT INTO trades (order_id, client_order_id, opportunity_id, market_id, side, price, size, filled_qty, fill_status, pnl, strategy_id, latency_ms, risk_check, slippage_check, error_reason, account_id, placed_at, filled_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	var pnl interface{}
	if record.PnL != nil {
		pnl = record.PnL.String()
	}

	_, err := r.pool.Exec(ctx, query,
		record.OrderID,
		record.ClientOrderID,
		record.OpportunityID,
		record.MarketID,
		record.Side,
		record.Price.String(),
		record.Size.String(),
		record.FilledQty.String(),
		record.FillStatus,
		pnl,
		record.StrategyID,
		record.LatencyMs,
		record.RiskCheck,
		record.SlippageCheck,
		record.ErrorReason,
		record.AccountID,
		record.PlacedAt,
		record.FilledAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert trade: %w", err)
	}

	return nil
}

func (r *PostgresRepo) InsertRiskEvent(ctx context.Context, event ports.RiskEvent) error {
	query := `
		INSERT INTO risk_events (market_id, strategy_id, order_size, allowed, reason, latency_ms)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.pool.Exec(ctx, query,
		event.MarketID,
		event.StrategyID,
		event.OrderSize.String(),
		event.Allowed,
		event.Reason,
		event.LatencyMs,
	)
	if err != nil {
		r.logger.Error("failed to insert risk event",
			zap.String("market_id", event.MarketID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to insert risk event: %w", err)
	}

	return nil
}

func (r *PostgresRepo) InsertAtomicPair(ctx context.Context, record logger.AtomicPairRecord) error {
	query := `
		INSERT INTO trades (order_id, client_order_id, opportunity_id, market_id, side, price, size, filled_qty, fill_status, strategy_id, latency_ms, risk_check, slippage_check, error_reason, account_id, pair_id, leg, pair_status, placed_at, filled_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`

	_, err := r.pool.Exec(ctx, query,
		record.YesOrderID,
		record.YesClientOrderID,
		record.OpportunityID,
		record.MarketID,
		"BUY",
		record.YesPrice.String(),
		record.YesSize.String(),
		record.YesFilledQty.String(),
		record.Status,
		record.StrategyID,
		record.PlacementLatencyMs,
		"",
		"",
		record.FailureReason,
		record.AccountID,
		record.ID,
		"YES",
		record.Status,
		record.CreatedAt,
		record.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert atomic pair YES leg: %w", err)
	}

	_, err = r.pool.Exec(ctx, query,
		record.NoOrderID,
		record.NoClientOrderID,
		record.OpportunityID,
		record.MarketID,
		"BUY",
		record.NoPrice.String(),
		record.NoSize.String(),
		record.NoFilledQty.String(),
		record.Status,
		record.StrategyID,
		record.PlacementLatencyMs,
		"",
		"",
		record.FailureReason,
		record.AccountID,
		record.ID,
		"NO",
		record.Status,
		record.CreatedAt,
		record.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert atomic pair NO leg: %w", err)
	}

	return nil
}

func (r *PostgresRepo) UpdateAtomicPairStatus(ctx context.Context, pairID, status string, completedAt *time.Time) error {
	query := `
		UPDATE trades SET pair_status = $1, filled_at = $2 WHERE pair_id = $3
	`

	_, err := r.pool.Exec(ctx, query, status, completedAt, pairID)
	if err != nil {
		return fmt.Errorf("failed to update atomic pair status: %w", err)
	}

	return nil
}

func (r *PostgresRepo) InsertPartialFill(ctx context.Context, record logger.PartialFillDBRecord) error {
	query := `
		INSERT INTO atomic_partial_fills (pair_id, leg, filled_qty, remaining_qty, fill_price, order_id, client_order_id, market_id, strategy_id, account_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.pool.Exec(ctx, query,
		record.PairID,
		record.Leg,
		record.FilledQty.String(),
		record.RemainingQty.String(),
		record.FillPrice.String(),
		record.OrderID,
		record.ClientOrderID,
		record.MarketID,
		record.StrategyID,
		record.AccountID,
	)
	if err != nil {
		return fmt.Errorf("failed to insert partial fill: %w", err)
	}

	return nil
}

func (r *PostgresRepo) InsertCircuitBreakerEvent(ctx context.Context, record logger.CircuitBreakerEventRecord) error {
	query := `
		INSERT INTO circuit_breaker_events (event_type, state_from, state_to, consecutive_errors, last_error, cooldown_seconds, user_id, reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.pool.Exec(ctx, query,
		record.EventType,
		record.StateFrom,
		record.StateTo,
		record.ConsecutiveErrors,
		record.LastError,
		record.CooldownSeconds,
		record.UserID,
		record.Reason,
	)
	if err != nil {
		return fmt.Errorf("failed to insert circuit breaker event: %w", err)
	}

	return nil
}

func (r *PostgresRepo) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *PostgresRepo) Close() error {
	r.pool.Close()
	return nil
}

func (r *PostgresRepo) Pool() *pgxpool.Pool {
	return r.pool
}
