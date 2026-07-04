package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/shopspring/decimal"
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
	query := `
		CREATE TABLE IF NOT EXISTS risk_events (
			id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			decision                TEXT NOT NULL,
			reason                  TEXT NOT NULL,
			market_id               TEXT DEFAULT NULL,
			strategy_id             TEXT DEFAULT NULL,
			trade_size              NUMERIC(18,8) NOT NULL DEFAULT 0,
			current_exposure        NUMERIC(18,8) NOT NULL DEFAULT 0,
			limit_value             NUMERIC(18,8) NOT NULL DEFAULT 0,
			daily_budget_remaining  NUMERIC(18,8) NOT NULL,
			capital                 NUMERIC(18,8) NOT NULL,
			context                 JSONB DEFAULT '{}',
			account_id              UUID DEFAULT NULL,
			created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`

	_, err := r.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create risk_events table: %w", err)
	}

	positionsQuery := `
		CREATE TABLE IF NOT EXISTS positions (
			position_id   TEXT PRIMARY KEY,
			market_id     TEXT NOT NULL,
			strategy_id   TEXT NOT NULL,
			side          TEXT NOT NULL,
			entry_price   NUMERIC(18,8) NOT NULL,
			current_price NUMERIC(18,8) NOT NULL,
			quantity      NUMERIC(18,8) NOT NULL,
			realized_pnl  NUMERIC(18,8) NOT NULL DEFAULT 0,
			status        TEXT NOT NULL DEFAULT 'open',
			account_id    UUID DEFAULT NULL,
			created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`
	if _, err := r.pool.Exec(ctx, positionsQuery); err != nil {
		return fmt.Errorf("failed to create positions table: %w", err)
	}

	correlationGroupsQuery := `
		CREATE TABLE IF NOT EXISTS correlation_groups (
			id                  TEXT PRIMARY KEY,
			name                TEXT NOT NULL,
			detection_method    TEXT NOT NULL,
			market_ids          TEXT[] NOT NULL,
			max_positions       INT NOT NULL DEFAULT 3,
			confidence          NUMERIC(3,2) NOT NULL DEFAULT 0.0,
			last_updated        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`
	if _, err := r.pool.Exec(ctx, correlationGroupsQuery); err != nil {
		return fmt.Errorf("failed to create correlation_groups table: %w", err)
	}

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_risk_events_created_at ON risk_events(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_risk_events_decision ON risk_events(decision)`,
		`CREATE INDEX IF NOT EXISTS idx_risk_events_reason ON risk_events(reason)`,
		`CREATE INDEX IF NOT EXISTS idx_risk_events_market_id ON risk_events(market_id)`,
		`CREATE INDEX IF NOT EXISTS idx_risk_events_strategy_id ON risk_events(strategy_id)`,
		`CREATE INDEX IF NOT EXISTS idx_risk_events_account_id ON risk_events(account_id)`,
		`CREATE INDEX IF NOT EXISTS idx_positions_market_id ON positions(market_id)`,
		`CREATE INDEX IF NOT EXISTS idx_positions_strategy_id ON positions(strategy_id)`,
		`CREATE INDEX IF NOT EXISTS idx_positions_status ON positions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_correlation_groups_detection_method ON correlation_groups(detection_method)`,
		`CREATE INDEX IF NOT EXISTS idx_correlation_groups_last_updated ON correlation_groups(last_updated)`,
	}

	for _, idx := range indexes {
		if _, err := r.pool.Exec(ctx, idx); err != nil {
			r.logger.Warn("index creation (may already exist)", zap.Error(err))
		}
	}

	return nil
}

func (r *PostgresRepo) InsertRiskEvent(ctx context.Context, event ports.RiskDecision) error {
	query := `
		INSERT INTO risk_events (id, decision, reason, market_id, strategy_id, trade_size, current_exposure, limit_value, daily_budget_remaining, capital, context, account_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	contextJSON := "{}"
	if event.Context != nil {
		ctxBytes, marshalErr := json.Marshal(event.Context)
		if marshalErr != nil {
			// #24: Log marshal error, store fallback
			r.logger.Error("failed to marshal event context, storing empty JSON", zap.Error(marshalErr))
			contextJSON = "{}"
		} else {
			contextJSON = string(ctxBytes)
		}
	}

	_, err := r.pool.Exec(ctx, query,
		event.EventID,
		event.Decision,
		event.Reason,
		event.MarketID,
		event.StrategyID,
		event.TradeSize.String(),
		event.CurrentExposure.String(),
		event.LimitValue.String(),
		event.DailyBudgetRemaining.String(),
		event.Capital.String(),
		contextJSON,
		event.AccountID,
		event.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to insert risk event: %w", err)
	}

	return nil
}

func (r *PostgresRepo) GetTodayDecisions(ctx context.Context) ([]ports.RiskDecision, error) {
	query := `
		SELECT id, decision, reason, market_id, strategy_id, trade_size, current_exposure, limit_value, daily_budget_remaining, capital, context, account_id, created_at
		FROM risk_events
		WHERE created_at >= CURRENT_DATE
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query today's decisions: %w", err)
	}
	defer rows.Close()

	var decisions []ports.RiskDecision
	for rows.Next() {
		var d ports.RiskDecision
		var tradeSize, currentExposure, limitValue, dailyBudgetRemaining, capital string
		var contextJSON string

		err := rows.Scan(
			&d.EventID,
			&d.Decision,
			&d.Reason,
			&d.MarketID,
			&d.StrategyID,
			&tradeSize,
			&currentExposure,
			&limitValue,
			&dailyBudgetRemaining,
			&capital,
			&contextJSON,
			&d.AccountID,
			&d.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan risk event: %w", err)
		}

		d.TradeSize, err = decimal.NewFromString(tradeSize)
		if err != nil {
			return nil, fmt.Errorf("invalid trade_size %q: %w", tradeSize, err)
		}
		d.CurrentExposure, err = decimal.NewFromString(currentExposure)
		if err != nil {
			return nil, fmt.Errorf("invalid current_exposure %q: %w", currentExposure, err)
		}
		d.LimitValue, err = decimal.NewFromString(limitValue)
		if err != nil {
			return nil, fmt.Errorf("invalid limit_value %q: %w", limitValue, err)
		}
		d.DailyBudgetRemaining, err = decimal.NewFromString(dailyBudgetRemaining)
		if err != nil {
			return nil, fmt.Errorf("invalid daily_budget_remaining %q: %w", dailyBudgetRemaining, err)
		}
		d.Capital, err = decimal.NewFromString(capital)
		if err != nil {
			return nil, fmt.Errorf("invalid capital %q: %w", capital, err)
		}

		decisions = append(decisions, d)
	}

	// #11: Check rows.Err() after iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating risk events: %w", err)
	}

	return decisions, nil
}

// #3: Fix GetDailyLoss to query actual realized PnL from positions table
func (r *PostgresRepo) GetDailyLoss(ctx context.Context) (decimal.Decimal, error) {
	query := `
		SELECT COALESCE(SUM(ABS(realized_pnl)), 0) as daily_loss
		FROM positions
		WHERE status = 'closed'
		  AND realized_pnl < 0
		  AND updated_at >= CURRENT_DATE
	`

	var dailyLoss string
	err := r.pool.QueryRow(ctx, query).Scan(&dailyLoss)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get daily loss: %w", err)
	}

	return decimal.NewFromString(dailyLoss)
}

// #8, #15: Get current exposures from positions table
func (r *PostgresRepo) GetPositionExposures(ctx context.Context) (map[string]decimal.Decimal, map[string]decimal.Decimal, error) {
	query := `
		SELECT market_id, strategy_id, 
		       COALESCE(SUM(entry_price * quantity), 0) as exposure
		FROM positions
		WHERE status = 'open'
		GROUP BY market_id, strategy_id
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query position exposures: %w", err)
	}
	defer rows.Close()

	marketExposures := make(map[string]decimal.Decimal)
	strategyExposures := make(map[string]decimal.Decimal)

	for rows.Next() {
		var marketID, strategyID string
		var exposureStr string
		if err := rows.Scan(&marketID, &strategyID, &exposureStr); err != nil {
			return nil, nil, fmt.Errorf("failed to scan position exposure: %w", err)
		}
		exposure, err := decimal.NewFromString(exposureStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid exposure %q for market %s: %w", exposureStr, marketID, err)
		}
		marketExposures[marketID] = marketExposures[marketID].Add(exposure)
		strategyExposures[strategyID] = strategyExposures[strategyID].Add(exposure)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating position exposures: %w", err)
	}

	return marketExposures, strategyExposures, nil
}

func (r *PostgresRepo) GetRecentTrades(ctx context.Context, limit int) ([]ports.TradeRecord, error) {
	query := `
		SELECT position_id, market_id, strategy_id, side, entry_price, 
		       COALESCE(current_price, entry_price), quantity, realized_pnl, updated_at
		FROM positions
		WHERE status = 'closed'
		ORDER BY updated_at DESC
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent trades: %w", err)
	}
	defer rows.Close()

	var trades []ports.TradeRecord
	for rows.Next() {
		var t ports.TradeRecord
		var entryPrice, exitPrice, quantity, realizedPnL string

		if err := rows.Scan(&t.ID, &t.MarketID, &t.StrategyID, &t.Side, &entryPrice, &exitPrice, &quantity, &realizedPnL, &t.ClosedAt); err != nil {
			return nil, fmt.Errorf("failed to scan trade record: %w", err)
		}

		t.EntryPrice, err = decimal.NewFromString(entryPrice)
		if err != nil {
			return nil, fmt.Errorf("invalid entry_price %q: %w", entryPrice, err)
		}
		t.ExitPrice, err = decimal.NewFromString(exitPrice)
		if err != nil {
			return nil, fmt.Errorf("invalid exit_price %q: %w", exitPrice, err)
		}
		t.Quantity, err = decimal.NewFromString(quantity)
		if err != nil {
			return nil, fmt.Errorf("invalid quantity %q: %w", quantity, err)
		}
		t.RealizedPnL, err = decimal.NewFromString(realizedPnL)
		if err != nil {
			return nil, fmt.Errorf("invalid realized_pnl %q: %w", realizedPnL, err)
		}

		trades = append(trades, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating trade records: %w", err)
	}

	return trades, nil
}

func (r *PostgresRepo) Close() error {
	r.pool.Close()
	return nil
}

// #8: UpsertCorrelationGroup persists a correlation group to PostgreSQL
func (r *PostgresRepo) UpsertCorrelationGroup(ctx context.Context, group ports.CorrelationGroupData) error {
	query := `
		INSERT INTO correlation_groups (id, name, detection_method, market_ids, max_positions, confidence, last_updated)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			detection_method = EXCLUDED.detection_method,
			market_ids = EXCLUDED.market_ids,
			max_positions = EXCLUDED.max_positions,
			confidence = EXCLUDED.confidence,
			last_updated = EXCLUDED.last_updated
	`
	_, err := r.pool.Exec(ctx, query,
		group.ID,
		group.Name,
		group.DetectionMethod,
		group.MarketIDs,
		group.MaxPositions,
		group.Confidence,
		group.LastUpdated,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert correlation group: %w", err)
	}
	return nil
}

// #8: GetCorrelationGroups retrieves all correlation groups from PostgreSQL
func (r *PostgresRepo) GetCorrelationGroups(ctx context.Context) ([]ports.CorrelationGroupData, error) {
	query := `
		SELECT id, name, detection_method, market_ids, max_positions, confidence, last_updated
		FROM correlation_groups
		ORDER BY last_updated DESC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query correlation groups: %w", err)
	}
	defer rows.Close()

	var groups []ports.CorrelationGroupData
	for rows.Next() {
		var g ports.CorrelationGroupData
		if err := rows.Scan(&g.ID, &g.Name, &g.DetectionMethod, &g.MarketIDs, &g.MaxPositions, &g.Confidence, &g.LastUpdated); err != nil {
			return nil, fmt.Errorf("failed to scan correlation group: %w", err)
		}
		groups = append(groups, g)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating correlation groups: %w", err)
	}

	return groups, nil
}

// #8: DeleteCorrelationGroup removes a correlation group from PostgreSQL
func (r *PostgresRepo) DeleteCorrelationGroup(ctx context.Context, id string) error {
	query := `DELETE FROM correlation_groups WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete correlation group: %w", err)
	}
	return nil
}
