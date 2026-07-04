package logger

import (
	"context"

	"github.com/pqap/services/arb-engine/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type OpportunityLogger struct {
	repo   OpportunityRepository
	logger *zap.Logger
}

type OpportunityRepository interface {
	Insert(ctx context.Context, opp ports.Opportunity) error
	GetHistoricalFillRate(ctx context.Context, marketID string, days int) (decimal.Decimal, int, error)
	Close() error
}

func NewOpportunityLogger(repo OpportunityRepository, logger *zap.Logger) *OpportunityLogger {
	return &OpportunityLogger{
		repo:   repo,
		logger: logger,
	}
}

func (l *OpportunityLogger) Log(ctx context.Context, opp ports.Opportunity) error {
	if err := l.repo.Insert(ctx, opp); err != nil {
		l.logger.Error("failed to log opportunity to TimescaleDB",
			zap.String("opportunity_id", opp.ID),
			zap.String("market_id", opp.MarketID),
			zap.Error(err),
		)
		return err
	}

	l.logger.Debug("opportunity logged",
		zap.String("opportunity_id", opp.ID),
		zap.String("market_id", opp.MarketID),
		zap.String("spread", opp.Spread.String()),
		zap.String("score", opp.Score.String()),
		zap.String("filter_reason", opp.FilterReason),
	)
	return nil
}

func (l *OpportunityLogger) GetHistoricalFillRate(ctx context.Context, marketID string, days int) (decimal.Decimal, int, error) {
	return l.repo.GetHistoricalFillRate(ctx, marketID, days)
}

func (l *OpportunityLogger) Close() error {
	return l.repo.Close()
}
