package adapters

import (
	"context"
	"database/sql"
	"time"

	"github.com/pqap/services/arb-engine/internal/ports"
	"go.uber.org/zap"
)

type PostgresRepo struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewPostgresRepo(postgresURL string, logger *zap.Logger) (*PostgresRepo, error) {
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	logger.Info("connected to PostgreSQL for market relationships")
	return &PostgresRepo{db: db, logger: logger}, nil
}

func (r *PostgresRepo) GetRelationships(ctx context.Context) ([]ports.MarketRelationship, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, market_a_id, market_b_id, relationship_type, confidence
		 FROM market_relationships
		 ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rels []ports.MarketRelationship
	for rows.Next() {
		var rel ports.MarketRelationship
		if err := rows.Scan(&rel.ID, &rel.MarketAID, &rel.MarketBID, &rel.RelationshipType, &rel.Confidence); err != nil {
			return nil, err
		}
		rels = append(rels, rel)
	}
	return rels, rows.Err()
}

func (r *PostgresRepo) GetRelatedMarkets(ctx context.Context, marketID string) ([]ports.MarketRelationship, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, market_a_id, market_b_id, relationship_type, confidence
		 FROM market_relationships
		 WHERE market_a_id = $1 OR market_b_id = $1`, marketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rels []ports.MarketRelationship
	for rows.Next() {
		var rel ports.MarketRelationship
		if err := rows.Scan(&rel.ID, &rel.MarketAID, &rel.MarketBID, &rel.RelationshipType, &rel.Confidence); err != nil {
			return nil, err
		}
		rels = append(rels, rel)
	}
	return rels, rows.Err()
}

func (r *PostgresRepo) UpsertRelationship(ctx context.Context, rel ports.MarketRelationship) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO market_relationships (id, market_a_id, market_b_id, relationship_type, confidence, updated_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())
		 ON CONFLICT (market_a_id, market_b_id, relationship_type)
		 DO UPDATE SET confidence = $5, updated_at = NOW()`,
		rel.ID, rel.MarketAID, rel.MarketBID, rel.RelationshipType, rel.Confidence)
	return err
}

func (r *PostgresRepo) DeleteRelationship(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM market_relationships WHERE id = $1`, id)
	return err
}

func (r *PostgresRepo) Close() error {
	return r.db.Close()
}

func (r *PostgresRepo) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return r.db.PingContext(ctx)
}
