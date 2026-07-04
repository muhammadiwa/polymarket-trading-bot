package tracker

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/position-manager/internal/ports"
	"github.com/pqap/services/position-manager/metrics"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type ResolutionDetector struct {
	repo      ports.PositionRepository
	publisher ports.EventPublisher
	logger    *zap.Logger
}

func NewResolutionDetector(repo ports.PositionRepository, publisher ports.EventPublisher, logger *zap.Logger) *ResolutionDetector {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ResolutionDetector{
		repo:      repo,
		publisher: publisher,
		logger:    logger,
	}
}

func (rd *ResolutionDetector) HandleMarketResolved(ctx context.Context, event ports.MarketResolved) error {
	marketID := event.Payload.MarketID
	outcome := event.Payload.Outcome

	positions, err := rd.repo.GetOpenPositionsByMarket(ctx, marketID)
	if err != nil {
		rd.logger.Error("failed to get positions for resolved market",
			zap.String("market_id", marketID),
			zap.Error(err),
		)
		return err
	}

	if len(positions) == 0 {
		rd.logger.Debug("no open positions for resolved market", zap.String("market_id", marketID))
		return nil
	}

	rd.logger.Info("market resolved, settling positions",
		zap.String("market_id", marketID),
		zap.String("outcome", outcome),
		zap.Int("position_count", len(positions)),
	)

	for _, position := range positions {
		if err := rd.settlePosition(ctx, position, outcome); err != nil {
			rd.logger.Error("failed to settle position",
				zap.String("position_id", position.ID),
				zap.Error(err),
			)
			continue
		}
	}

	return nil
}

func (rd *ResolutionDetector) settlePosition(ctx context.Context, position *ports.Position, outcome string) error {
	var exitPrice decimal.Decimal

	if outcome == "YES" {
		if position.Side == "YES" {
			exitPrice = decimal.NewFromFloat(1.0000)
		} else {
			exitPrice = decimal.NewFromFloat(0.0000)
		}
	} else {
		if position.Side == "YES" {
			exitPrice = decimal.NewFromFloat(0.0000)
		} else {
			exitPrice = decimal.NewFromFloat(1.0000)
		}
	}

	now := time.Now().UTC()
	position.RealizedPnL = CalculateRealizedPnL(position.EntryPrice, exitPrice, position.Quantity)
	position.CurrentPrice = exitPrice
	position.Status = ports.StatusSettled
	position.SettledAt = &now
	position.ClosedAt = &now
	position.UpdatedAt = now

	if err := rd.repo.MoveToHistory(ctx, position, exitPrice, "resolution"); err != nil {
		return err
	}

	closedEvent := ports.PositionClosed{
		EventID:   uuid.New().String(),
		EventType: "PositionClosed",
		Timestamp: now,
		Source:    "position-manager",
		Payload: ports.PositionClosedPayload{
			PositionID:  position.ID,
			MarketID:    position.MarketID,
			Side:        position.Side,
			EntryPrice:  position.EntryPrice,
			ExitPrice:   exitPrice,
			Quantity:    position.Quantity,
			RealizedPnL: position.RealizedPnL,
			ExitReason:  "resolution",
			StrategyID:  position.StrategyID,
			AccountID:   position.AccountID,
		},
	}

	if err := rd.publisher.PublishPositionClosed(ctx, closedEvent); err != nil {
		rd.logger.Error("failed to publish position closed event", zap.Error(err))
	}

	metrics.PositionSettledTotal.Inc()
	metrics.PositionActiveCount.Dec()
	metrics.RealizedPnLTotal.Add(position.RealizedPnL.InexactFloat64())

	rd.logger.Info("position settled",
		zap.String("position_id", position.ID),
		zap.String("side", position.Side),
		zap.String("exit_price", exitPrice.String()),
		zap.String("realized_pnl", position.RealizedPnL.String()),
	)

	return nil
}
