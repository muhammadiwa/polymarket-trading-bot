package tracker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/position-manager/internal/ports"
	"github.com/pqap/services/position-manager/metrics"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type ExitHandler struct {
	repo      ports.PositionRepository
	publisher ports.EventPublisher
	logger    *zap.Logger
}

func NewExitHandler(repo ports.PositionRepository, publisher ports.EventPublisher, logger *zap.Logger) *ExitHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ExitHandler{
		repo:      repo,
		publisher: publisher,
		logger:    logger,
	}
}

func (eh *ExitHandler) RequestExit(ctx context.Context, positionID string) error {
	start := time.Now()

	position, err := eh.repo.GetPosition(ctx, positionID)
	if err != nil {
		return fmt.Errorf("%w: %w", ports.ErrPositionNotFound, err)
	}

	if position.Status != ports.StatusOpen && position.Status != ports.StatusMonitoring {
		return fmt.Errorf("position %s cannot be exited (status: %s)", positionID, position.Status)
	}

	updated, err := eh.repo.UpdatePositionStatus(ctx, positionID, position.Status, ports.StatusClosing)
	if err != nil {
		return fmt.Errorf("failed to update position status: %w", err)
	}
	if !updated {
		return fmt.Errorf("position %s status changed concurrently, retry required", positionID)
	}

	exitOrderID := uuid.New().String()
	position.ExitOrderID = &exitOrderID
	position.Status = ports.StatusClosing
	position.UpdatedAt = time.Now().UTC()
	if err := eh.repo.UpdatePosition(ctx, position); err != nil {
		rollbackErr := eh.repo.UpdatePositionStatus(ctx, positionID, ports.StatusClosing, position.Status)
		if rollbackErr != nil {
			eh.logger.Error("failed to rollback position status after update failure",
				zap.String("position_id", positionID),
				zap.Error(rollbackErr),
			)
		}
		return fmt.Errorf("failed to record exit order ID: %w", err)
	}

	now := time.Now().UTC()
	exitOrder := ports.ExitOrderRequest{
		EventID:   uuid.New().String(),
		EventType: "ExitOrderRequest",
		Timestamp: now,
		Source:    "position-manager",
		Payload: ports.ExitOrderRequestPayload{
			PositionID: positionID,
			MarketID:   position.MarketID,
			Side:       position.Side,
			Quantity:   position.Quantity,
			OrderType:  "MARKET",
			Reason:     "manual_exit",
		},
	}

	if err := eh.publisher.PublishExitOrderRequest(ctx, exitOrder); err != nil {
		rollbackErr := eh.repo.UpdatePositionStatus(ctx, positionID, ports.StatusClosing, ports.StatusOpen)
		if rollbackErr != nil {
			eh.logger.Error("failed to rollback position status on publish failure",
				zap.String("position_id", positionID),
				zap.Error(rollbackErr),
			)
		}
		return fmt.Errorf("failed to publish exit order: %w", err)
	}

	elapsed := time.Since(start)
	metrics.ExitLatency.Observe(float64(elapsed.Milliseconds()))

	eh.logger.Info("exit order requested",
		zap.String("position_id", positionID),
		zap.String("market_id", position.MarketID),
		zap.Duration("latency", elapsed),
	)

	return nil
}

func (eh *ExitHandler) HandleExitFill(ctx context.Context, positionID string, exitPrice decimal.Decimal) error {
	position, err := eh.repo.GetPosition(ctx, positionID)
	if err != nil {
		return fmt.Errorf("%w: %w", ports.ErrPositionNotFound, err)
	}

	if position.Status != ports.StatusClosing {
		return fmt.Errorf("position %s is not in CLOSING status", positionID)
	}

	now := time.Now().UTC()
	position.RealizedPnL = CalculateRealizedPnL(position.EntryPrice, exitPrice, position.Quantity)
	position.CurrentPrice = exitPrice
	position.Status = ports.StatusClosed
	position.ClosedAt = &now
	position.UpdatedAt = now

	if err := eh.repo.MoveToHistory(ctx, position, exitPrice, "manual"); err != nil {
		return fmt.Errorf("failed to move position to history: %w", err)
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
			ExitReason:  "manual",
			StrategyID:  position.StrategyID,
			AccountID:   position.AccountID,
		},
	}

	if err := eh.publisher.PublishPositionClosed(ctx, closedEvent); err != nil {
		eh.logger.Error("failed to publish position closed event", zap.Error(err))
	}

	metrics.PositionClosedTotal.Inc()
	metrics.PositionActiveCount.Dec()
	metrics.RealizedPnLTotal.Add(position.RealizedPnL.InexactFloat64())

	eh.logger.Info("position closed via manual exit",
		zap.String("position_id", positionID),
		zap.String("exit_price", exitPrice.String()),
		zap.String("realized_pnl", position.RealizedPnL.String()),
	)

	return nil
}
