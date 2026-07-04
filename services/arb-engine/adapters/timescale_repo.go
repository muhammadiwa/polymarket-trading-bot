package adapters

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pqap/services/arb-engine/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

var marketIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

type TimescaleRepo struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewTimescaleRepo(url string, logger *zap.Logger) (*TimescaleRepo, error) {
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("invalid TimescaleDB URL: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to TimescaleDB: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping TimescaleDB: %w", err)
	}

	repo := &TimescaleRepo{
		pool:   pool,
		logger: logger,
	}

	if err := repo.ensureSchema(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ensure schema: %w", err)
	}

	return repo, nil
}

func (r *TimescaleRepo) ensureSchema(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS opportunities (
			time              TIMESTAMPTZ NOT NULL,
			opportunity_id    UUID NOT NULL,
			market_id         TEXT NOT NULL,
			yes_price         NUMERIC(10,4) NOT NULL,
			no_price          NUMERIC(10,4) NOT NULL,
			spread            NUMERIC(10,4) NOT NULL,
			liquidity         NUMERIC(10,8) NOT NULL,
			fill_probability  NUMERIC(10,8) NOT NULL,
			score             NUMERIC(10,8) NOT NULL,
			filter_reason     TEXT DEFAULT '',
			latency_ms        INTEGER NOT NULL,
			account_id        UUID DEFAULT NULL,
			filled            BOOLEAN DEFAULT NULL
		);
	`

	_, err := r.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create opportunities table: %w", err)
	}

	_, err = r.pool.Exec(ctx, "SELECT create_hypertable('opportunities', 'time', if_not_exists => TRUE)")
	if err != nil {
		r.logger.Warn("hypertable creation (may already exist)", zap.Error(err))
	}

	_, err = r.pool.Exec(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_opportunities_id_time ON opportunities (opportunity_id, time)`)
	if err != nil {
		r.logger.Warn("unique index creation (may already exist)", zap.Error(err))
	}

	_, err = r.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_opportunities_market_time ON opportunities (market_id, time)`)
	if err != nil {
		r.logger.Warn("market time index creation (may already exist)", zap.Error(err))
	}

	return nil
}

func ValidateMarketID(marketID string) error {
	if marketID == "" {
		return fmt.Errorf("market_id is empty")
	}
	if !marketIDPattern.MatchString(marketID) {
		return fmt.Errorf("market_id contains invalid characters or exceeds length")
	}
	return nil
}

func (r *TimescaleRepo) Insert(ctx context.Context, opp ports.Opportunity) error {
	query := `
		INSERT INTO opportunities (time, opportunity_id, market_id, yes_price, no_price, spread, liquidity, fill_probability, score, filter_reason, latency_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.pool.Exec(ctx, query,
		opp.DetectedAt,
		opp.ID,
		opp.MarketID,
		opp.YESPrice.String(),
		opp.NOPrice.String(),
		opp.Spread.String(),
		opp.Liquidity.String(),
		opp.FillProbability.String(),
		opp.Score.String(),
		opp.FilterReason,
		opp.LatencyMs,
	)
	if err != nil {
		return fmt.Errorf("failed to insert opportunity: %w", err)
	}

	return nil
}

func (r *TimescaleRepo) MarkFilled(ctx context.Context, opportunityID string, filled bool) error {
	query := `UPDATE opportunities SET filled = $1 WHERE opportunity_id = $2 AND filled IS NULL`
	_, err := r.pool.Exec(ctx, query, filled, opportunityID)
	return err
}

func (r *TimescaleRepo) GetHistoricalFillRate(ctx context.Context, marketID string, days int) (decimal.Decimal, int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN filled = true THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0), 0.5) as fill_rate,
			COUNT(*) as sample_count
		FROM opportunities
		WHERE market_id = $1
		AND time > NOW() - INTERVAL '1 day' * $2
	`

	var fillRate float64
	var sampleCount int
	err := r.pool.QueryRow(ctx, query, marketID, days).Scan(&fillRate, &sampleCount)
	if err != nil {
		return decimal.NewFromFloat(0.5), 0, err
	}

	return decimal.NewFromFloat(fillRate), sampleCount, nil
}

func (r *TimescaleRepo) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *TimescaleRepo) Close() error {
	r.pool.Close()
	return nil
}
