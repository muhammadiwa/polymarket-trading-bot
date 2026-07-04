package tracker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/position-manager/internal/ports"
	"github.com/pqap/services/position-manager/metrics"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type Tracker struct {
	repo      ports.PositionRepository
	publisher ports.EventPublisher
	logger    *zap.Logger
	seenIDs   map[string]struct{}
	seenMu    sync.Mutex
}

func NewTracker(repo ports.PositionRepository, publisher ports.EventPublisher, logger *zap.Logger) *Tracker {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Tracker{
		repo:      repo,
		publisher: publisher,
		logger:    logger,
		seenIDs:   make(map[string]struct{}),
	}
}

func (t *Tracker) InitializeActiveCount(ctx context.Context) {
	count, err := t.repo.GetActivePositionCount(ctx)
	if err != nil {
		t.logger.Error("failed to get active position count for initialization", zap.Error(err))
		return
	}
	metrics.PositionActiveCount.Set(float64(count))
	t.logger.Info("initialized position active count from DB", zap.Int("count", count))
}

func (t *Tracker) HandleOrderFilled(ctx context.Context, event ports.OrderFilled) error {
	t.seenMu.Lock()
	if _, exists := t.seenIDs[event.EventID]; exists {
		t.seenMu.Unlock()
		t.logger.Debug("duplicate OrderFilled event, skipping", zap.String("event_id", event.EventID))
		return nil
	}
	t.seenIDs[event.EventID] = struct{}{}
	t.seenMu.Unlock()

	if event.Payload.Side != "YES" && event.Payload.Side != "NO" {
		return fmt.Errorf("invalid side value %q: must be YES or NO", event.Payload.Side)
	}

	now := time.Now().UTC()

	position := &ports.Position{
		ID:            uuid.New().String(),
		MarketID:      event.Payload.MarketID,
		MarketSlug:    event.Payload.MarketSlug,
		Side:          event.Payload.Side,
		EntryPrice:    event.Payload.Price,
		CurrentPrice:  event.Payload.Price,
		Quantity:      event.Payload.FilledQty,
		UnrealizedPnL: decimal.Zero,
		RealizedPnL:   decimal.Zero,
		Status:        ports.StatusOpen,
		StrategyID:    event.Payload.StrategyID,
		EntryOrderID:  event.Payload.OrderID,
		ExitOrderID:   nil,
		OpenedAt:      now,
		ClosedAt:      nil,
		SettledAt:     nil,
		AccountID:     nil,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := t.repo.CreatePosition(ctx, position); err != nil {
		t.logger.Error("failed to create position",
			zap.String("order_id", event.Payload.OrderID),
			zap.Error(err),
		)
		return err
	}

	openedEvent := ports.PositionOpened{
		EventID:   uuid.New().String(),
		EventType: "PositionOpened",
		Timestamp: now,
		Source:    "position-manager",
		Payload: ports.PositionOpenedPayload{
			PositionID:   position.ID,
			MarketID:     position.MarketID,
			MarketSlug:   position.MarketSlug,
			Side:         position.Side,
			EntryPrice:   position.EntryPrice,
			Quantity:     position.Quantity,
			StrategyID:   position.StrategyID,
			EntryOrderID: position.EntryOrderID,
			AccountID:    position.AccountID,
		},
	}

	if err := t.publisher.PublishPositionOpened(ctx, openedEvent); err != nil {
		t.logger.Error("failed to publish position opened event", zap.Error(err))
		return fmt.Errorf("position created but event publish failed: %w", err)
	}

	metrics.PositionOpenTotal.Inc()
	metrics.PositionActiveCount.Inc()

	t.logger.Info("position created",
		zap.String("position_id", position.ID),
		zap.String("market_id", position.MarketID),
		zap.String("side", position.Side),
		zap.String("entry_price", position.EntryPrice.String()),
		zap.String("quantity", position.Quantity.String()),
		zap.String("strategy_id", position.StrategyID),
	)

	return nil
}

func (t *Tracker) HandlePriceUpdate(ctx context.Context, marketID string, event ports.MarketPriceUpdated) error {
	positions, err := t.repo.GetOpenPositionsByMarket(ctx, marketID)
	if err != nil {
		t.logger.Error("failed to get open positions for market",
			zap.String("market_id", marketID),
			zap.Error(err),
		)
		return err
	}

	if len(positions) == 0 {
		return nil
	}

	for _, position := range positions {
		var currentPrice decimal.Decimal
		switch position.Side {
		case "YES":
			currentPrice = event.Payload.YESPrice
		case "NO":
			currentPrice = event.Payload.NOPrice
		default:
			t.logger.Warn("skipping position with invalid side",
				zap.String("position_id", position.ID),
				zap.String("side", position.Side),
			)
			continue
		}

		UpdatePnL(position, currentPrice)

		if err := t.repo.UpdatePosition(ctx, position); err != nil {
			t.logger.Error("failed to update position",
				zap.String("position_id", position.ID),
				zap.Error(err),
			)
			continue
		}

		updatedEvent := ports.PositionUpdated{
			EventID:   uuid.New().String(),
			EventType: "PositionUpdated",
			Timestamp: time.Now().UTC(),
			Source:    "position-manager",
			Payload: ports.PositionUpdatedPayload{
				PositionID:    position.ID,
				MarketID:      position.MarketID,
				CurrentPrice:  position.CurrentPrice,
				UnrealizedPnL: position.UnrealizedPnL,
				UpdatedAt:     position.UpdatedAt,
			},
		}

		if err := t.publisher.PublishPositionUpdated(ctx, updatedEvent); err != nil {
			t.logger.Error("failed to publish position updated event", zap.Error(err))
		}
	}

	t.updateTotalUnrealizedPnL(ctx)

	return nil
}

func (t *Tracker) updateTotalUnrealizedPnL(ctx context.Context) {
	positions, err := t.repo.GetOpenPositions(ctx)
	if err != nil {
		t.logger.Error("failed to get open positions for total unrealized PnL update", zap.Error(err))
		return
	}

	total := decimal.Zero
	for _, p := range positions {
		total = total.Add(p.UnrealizedPnL)
	}

	metrics.UnrealizedPnL.Set(total.InexactFloat64())
}

func (t *Tracker) GetOpenPositions(ctx context.Context) ([]*ports.Position, error) {
	return t.repo.GetOpenPositions(ctx)
}

func (t *Tracker) GetPosition(ctx context.Context, id string) (*ports.Position, error) {
	return t.repo.GetPosition(ctx, id)
}

func (t *Tracker) GetHistory(ctx context.Context, limit, offset int) ([]*ports.PositionHistory, error) {
	return t.repo.GetHistory(ctx, limit, offset)
}
