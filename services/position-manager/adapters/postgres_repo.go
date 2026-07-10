package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pqap/services/position-manager/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type PostgresRepo struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewPostgresRepo(ctx context.Context, url string, logger *zap.Logger) (*PostgresRepo, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("invalid PostgreSQL URL: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	repo := &PostgresRepo{
		pool:   pool,
		logger: logger,
	}

	// Schema is managed by migrations
	// positions: migration 028_unify_positions
	// Only create position-manager specific tables

	if err := repo.ensurePositionTables(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ensure position tables: %w", err)
	}

	return repo, nil
}

func (r *PostgresRepo) ensurePositionTables(ctx context.Context) error {
	// Create position-manager specific tables (not managed by shared migrations)

	historyQuery := `
		CREATE TABLE IF NOT EXISTS position_history (
			id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			market_id       TEXT NOT NULL,
			market_slug     TEXT NOT NULL,
			side            TEXT NOT NULL,
			entry_price     NUMERIC(18,8) NOT NULL,
			exit_price      NUMERIC(18,8) NOT NULL,
			quantity        NUMERIC(18,8) NOT NULL,
			realized_pnl    NUMERIC(18,8) NOT NULL,
			strategy_id     TEXT NOT NULL,
			entry_order_id  UUID NOT NULL,
			exit_order_id   UUID DEFAULT NULL,
			exit_reason     TEXT NOT NULL,
			opened_at       TIMESTAMPTZ NOT NULL,
			closed_at       TIMESTAMPTZ NOT NULL,
			account_id      UUID DEFAULT NULL,
			archived_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`
	if _, err := r.pool.Exec(ctx, historyQuery); err != nil {
		return fmt.Errorf("failed to create position_history table: %w", err)
	}

	reconLogQuery := `
		CREATE TABLE IF NOT EXISTS position_reconciliation_log (
			id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			position_id            UUID DEFAULT NULL,
			market_id              TEXT NOT NULL,
			mismatch_type          TEXT NOT NULL,
			internal_state         JSONB NOT NULL,
			api_state              JSONB NOT NULL,
			consecutive_mismatches INTEGER NOT NULL,
			resolved               BOOLEAN NOT NULL DEFAULT FALSE,
			created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`
	if _, err := r.pool.Exec(ctx, reconLogQuery); err != nil {
		return fmt.Errorf("failed to create position_reconciliation_log table: %w", err)
	}

	reconStateQuery := `
		CREATE TABLE IF NOT EXISTS reconciliation_state (
			id                     INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
			consecutive_mismatches INTEGER NOT NULL DEFAULT 0,
			updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		INSERT INTO reconciliation_state (id, consecutive_mismatches) VALUES (1, 0) ON CONFLICT (id) DO NOTHING;
	`
	if _, err := r.pool.Exec(ctx, reconStateQuery); err != nil {
		return fmt.Errorf("failed to create reconciliation_state table: %w", err)
	}

	indices := []string{
		`CREATE INDEX IF NOT EXISTS idx_position_history_market_id ON position_history(market_id)`,
		`CREATE INDEX IF NOT EXISTS idx_position_history_strategy_id ON position_history(strategy_id)`,
		`CREATE INDEX IF NOT EXISTS idx_position_history_closed_at ON position_history(closed_at)`,
		`CREATE INDEX IF NOT EXISTS idx_recon_log_created_at ON position_reconciliation_log(created_at)`,
	}
	for _, idx := range indices {
		if _, err := r.pool.Exec(ctx, idx); err != nil {
			r.logger.Warn("index creation (may already exist)", zap.Error(err))
		}
	}

	return nil
}

func (r *PostgresRepo) CreatePosition(ctx context.Context, p *ports.Position) error {
	query := `
		INSERT INTO positions (id, market_id, market_slug, side, entry_price, current_price, quantity, unrealized_pnl, realized_pnl, status, strategy_id, entry_order_id, exit_order_id, opened_at, closed_at, settled_at, account_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`
	_, err := r.pool.Exec(ctx, query,
		p.ID, p.MarketID, p.MarketSlug, p.Side,
		p.EntryPrice.String(), p.CurrentPrice.String(), p.Quantity.String(),
		p.UnrealizedPnL.String(), p.RealizedPnL.String(),
		string(p.Status), p.StrategyID, p.EntryOrderID, p.ExitOrderID,
		p.OpenedAt, p.ClosedAt, p.SettledAt, p.AccountID,
		p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create position: %w", err)
	}
	return nil
}

func (r *PostgresRepo) UpdatePosition(ctx context.Context, p *ports.Position) error {
	query := `
		UPDATE positions SET
			current_price = $1, unrealized_pnl = $2, realized_pnl = $3,
			status = $4, exit_order_id = $5, closed_at = $6, settled_at = $7,
			updated_at = $8
		WHERE id = $9
	`
	_, err := r.pool.Exec(ctx, query,
		p.CurrentPrice.String(), p.UnrealizedPnL.String(), p.RealizedPnL.String(),
		string(p.Status), p.ExitOrderID, p.ClosedAt, p.SettledAt,
		time.Now().UTC(), p.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update position: %w", err)
	}
	return nil
}

func (r *PostgresRepo) UpdatePositionStatus(ctx context.Context, id string, expectedStatus, newStatus ports.PositionStatus) (bool, error) {
	query := `UPDATE positions SET status = $1, updated_at = NOW() WHERE id = $2 AND status = $3`
	tag, err := r.pool.Exec(ctx, query, string(newStatus), id, string(expectedStatus))
	if err != nil {
		return false, fmt.Errorf("failed to update position status: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *PostgresRepo) GetPosition(ctx context.Context, id string) (*ports.Position, error) {
	query := `
		SELECT id, market_id, market_slug, side, entry_price, current_price, quantity,
			   unrealized_pnl, realized_pnl, status, strategy_id, entry_order_id, exit_order_id,
			   opened_at, closed_at, settled_at, account_id, created_at, updated_at
		FROM positions WHERE id = $1
	`
	p := &ports.Position{}
	var entryPrice, currentPrice, quantity, unrealizedPnL, realizedPnL string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.MarketID, &p.MarketSlug, &p.Side,
		&entryPrice, &currentPrice, &quantity,
		&unrealizedPnL, &realizedPnL,
		&p.Status, &p.StrategyID, &p.EntryOrderID, &p.ExitOrderID,
		&p.OpenedAt, &p.ClosedAt, &p.SettledAt, &p.AccountID,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get position: %w", err)
	}
	if p.EntryPrice, err = decimal.NewFromString(entryPrice); err != nil {
		return nil, fmt.Errorf("invalid entry_price %q: %w", entryPrice, err)
	}
	if p.CurrentPrice, err = decimal.NewFromString(currentPrice); err != nil {
		return nil, fmt.Errorf("invalid current_price %q: %w", currentPrice, err)
	}
	if p.Quantity, err = decimal.NewFromString(quantity); err != nil {
		return nil, fmt.Errorf("invalid quantity %q: %w", quantity, err)
	}
	if p.UnrealizedPnL, err = decimal.NewFromString(unrealizedPnL); err != nil {
		return nil, fmt.Errorf("invalid unrealized_pnl %q: %w", unrealizedPnL, err)
	}
	if p.RealizedPnL, err = decimal.NewFromString(realizedPnL); err != nil {
		return nil, fmt.Errorf("invalid realized_pnl %q: %w", realizedPnL, err)
	}
	return p, nil
}

func (r *PostgresRepo) GetOpenPositions(ctx context.Context) ([]*ports.Position, error) {
	query := `
		SELECT id, market_id, market_slug, side, entry_price, current_price, quantity,
			   unrealized_pnl, realized_pnl, status, strategy_id, entry_order_id, exit_order_id,
			   opened_at, closed_at, settled_at, account_id, created_at, updated_at
		FROM positions WHERE status IN ('OPEN', 'MONITORING', 'CLOSING')
	`
	return r.scanPositions(ctx, query)
}

func (r *PostgresRepo) GetOpenPositionsByMarket(ctx context.Context, marketID string) ([]*ports.Position, error) {
	query := `
		SELECT id, market_id, market_slug, side, entry_price, current_price, quantity,
			   unrealized_pnl, realized_pnl, status, strategy_id, entry_order_id, exit_order_id,
			   opened_at, closed_at, settled_at, account_id, created_at, updated_at
		FROM positions WHERE market_id = $1 AND status IN ('OPEN', 'MONITORING', 'CLOSING')
	`
	return r.scanPositionsWithArgs(ctx, query, marketID)
}

func (r *PostgresRepo) GetOpenPositionsByStrategy(ctx context.Context, strategyID string) ([]*ports.Position, error) {
	query := `
		SELECT id, market_id, market_slug, side, entry_price, current_price, quantity,
			   unrealized_pnl, realized_pnl, status, strategy_id, entry_order_id, exit_order_id,
			   opened_at, closed_at, settled_at, account_id, created_at, updated_at
		FROM positions WHERE strategy_id = $1 AND status IN ('OPEN', 'MONITORING', 'CLOSING')
	`
	return r.scanPositionsWithArgs(ctx, query, strategyID)
}

func (r *PostgresRepo) GetActivePositionCount(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM positions WHERE status IN ('OPEN', 'MONITORING', 'CLOSING')`,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get active position count: %w", err)
	}
	return count, nil
}

func (r *PostgresRepo) scanPositions(ctx context.Context, query string) ([]*ports.Position, error) {
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query positions: %w", err)
	}
	defer rows.Close()

	var positions []*ports.Position
	for rows.Next() {
		p, err := r.scanPosition(rows)
		if err != nil {
			return nil, err
		}
		positions = append(positions, p)
	}
	return positions, nil
}

func (r *PostgresRepo) scanPositionsWithArgs(ctx context.Context, query string, args ...interface{}) ([]*ports.Position, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query positions: %w", err)
	}
	defer rows.Close()

	var positions []*ports.Position
	for rows.Next() {
		p, err := r.scanPosition(rows)
		if err != nil {
			return nil, err
		}
		positions = append(positions, p)
	}
	return positions, nil
}

func (r *PostgresRepo) scanPosition(rows interface{ Scan(dest ...interface{}) error }) (*ports.Position, error) {
	p := &ports.Position{}
	var entryPrice, currentPrice, quantity, unrealizedPnL, realizedPnL string
	if err := rows.Scan(
		&p.ID, &p.MarketID, &p.MarketSlug, &p.Side,
		&entryPrice, &currentPrice, &quantity,
		&unrealizedPnL, &realizedPnL,
		&p.Status, &p.StrategyID, &p.EntryOrderID, &p.ExitOrderID,
		&p.OpenedAt, &p.ClosedAt, &p.SettledAt, &p.AccountID,
		&p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to scan position: %w", err)
	}
	var err error
	if p.EntryPrice, err = decimal.NewFromString(entryPrice); err != nil {
		return nil, fmt.Errorf("invalid entry_price %q: %w", entryPrice, err)
	}
	if p.CurrentPrice, err = decimal.NewFromString(currentPrice); err != nil {
		return nil, fmt.Errorf("invalid current_price %q: %w", currentPrice, err)
	}
	if p.Quantity, err = decimal.NewFromString(quantity); err != nil {
		return nil, fmt.Errorf("invalid quantity %q: %w", quantity, err)
	}
	if p.UnrealizedPnL, err = decimal.NewFromString(unrealizedPnL); err != nil {
		return nil, fmt.Errorf("invalid unrealized_pnl %q: %w", unrealizedPnL, err)
	}
	if p.RealizedPnL, err = decimal.NewFromString(realizedPnL); err != nil {
		return nil, fmt.Errorf("invalid realized_pnl %q: %w", realizedPnL, err)
	}
	return p, nil
}

func (r *PostgresRepo) MoveToHistory(ctx context.Context, p *ports.Position, exitPrice decimal.Decimal, exitReason string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	insertQuery := `
		INSERT INTO position_history (id, market_id, market_slug, side, entry_price, exit_price, quantity, realized_pnl, strategy_id, entry_order_id, exit_order_id, exit_reason, opened_at, closed_at, account_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	_, err = tx.Exec(ctx, insertQuery,
		p.ID, p.MarketID, p.MarketSlug, p.Side,
		p.EntryPrice.String(), exitPrice.String(), p.Quantity.String(),
		p.RealizedPnL.String(), p.StrategyID, p.EntryOrderID, p.ExitOrderID,
		exitReason, p.OpenedAt, p.ClosedAt, p.AccountID,
	)
	if err != nil {
		return fmt.Errorf("failed to insert position history: %w", err)
	}

	deleteQuery := `DELETE FROM positions WHERE id = $1`
	_, err = tx.Exec(ctx, deleteQuery, p.ID)
	if err != nil {
		return fmt.Errorf("failed to delete position: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepo) GetHistory(ctx context.Context, limit, offset int) ([]*ports.PositionHistory, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `
		SELECT id, market_id, market_slug, side, entry_price, exit_price, quantity,
			   realized_pnl, strategy_id, entry_order_id, exit_order_id, exit_reason,
			   opened_at, closed_at, account_id
		FROM position_history ORDER BY closed_at DESC LIMIT $1 OFFSET $2
	`
	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query position history: %w", err)
	}
	defer rows.Close()

	var history []*ports.PositionHistory
	for rows.Next() {
		h := &ports.PositionHistory{}
		var entryPrice, exitPrice, quantity, realizedPnL string
		if err := rows.Scan(
			&h.ID, &h.MarketID, &h.MarketSlug, &h.Side,
			&entryPrice, &exitPrice, &quantity,
			&realizedPnL, &h.StrategyID, &h.EntryOrderID, &h.ExitOrderID,
			&h.ExitReason, &h.OpenedAt, &h.ClosedAt, &h.AccountID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan history: %w", err)
		}
		var err error
		if h.EntryPrice, err = decimal.NewFromString(entryPrice); err != nil {
			return nil, fmt.Errorf("invalid entry_price %q: %w", entryPrice, err)
		}
		if h.ExitPrice, err = decimal.NewFromString(exitPrice); err != nil {
			return nil, fmt.Errorf("invalid exit_price %q: %w", exitPrice, err)
		}
		if h.Quantity, err = decimal.NewFromString(quantity); err != nil {
			return nil, fmt.Errorf("invalid quantity %q: %w", quantity, err)
		}
		if h.RealizedPnL, err = decimal.NewFromString(realizedPnL); err != nil {
			return nil, fmt.Errorf("invalid realized_pnl %q: %w", realizedPnL, err)
		}
		history = append(history, h)
	}
	return history, nil
}

func (r *PostgresRepo) LogReconciliation(ctx context.Context, log *ports.ReconciliationLog) error {
	query := `
		INSERT INTO position_reconciliation_log (position_id, market_id, mismatch_type, internal_state, api_state, consecutive_mismatches, resolved)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		log.PositionID, log.MarketID, log.MismatchType,
		log.InternalState, log.APIState, log.ConsecutiveMismatches, log.Resolved,
	)
	if err != nil {
		return fmt.Errorf("failed to log reconciliation: %w", err)
	}
	return nil
}

func (r *PostgresRepo) GetReconciliationState(ctx context.Context) (*ports.ReconciliationState, error) {
	state := &ports.ReconciliationState{}

	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(created_at), '1970-01-01T00:00:00Z'::timestamptz) FROM position_reconciliation_log`,
	).Scan(&state.LastReconciledAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get reconciliation state: %w", err)
	}

	err = r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM position_reconciliation_log`,
	).Scan(&state.TotalReconciliations)
	if err != nil {
		return nil, fmt.Errorf("failed to get total reconciliations: %w", err)
	}

	err = r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM position_reconciliation_log WHERE NOT resolved`,
	).Scan(&state.TotalMismatches)
	if err != nil {
		return nil, fmt.Errorf("failed to get total mismatches: %w", err)
	}

	err = r.pool.QueryRow(ctx,
		`SELECT consecutive_mismatches FROM reconciliation_state WHERE id = 1`,
	).Scan(&state.ConsecutiveMismatches)
	if err != nil {
		return nil, fmt.Errorf("failed to get consecutive mismatches: %w", err)
	}

	return state, nil
}

func (r *PostgresRepo) IncrementMismatchCount(ctx context.Context) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE reconciliation_state SET consecutive_mismatches = consecutive_mismatches + 1, updated_at = NOW() WHERE id = 1`,
	)
	if err != nil {
		return fmt.Errorf("failed to increment mismatch count: %w", err)
	}
	return err
}

func (r *PostgresRepo) ResetMismatchCount(ctx context.Context) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE reconciliation_state SET consecutive_mismatches = 0, updated_at = NOW() WHERE id = 1`,
	)
	if err != nil {
		return fmt.Errorf("failed to reset mismatch count: %w", err)
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
