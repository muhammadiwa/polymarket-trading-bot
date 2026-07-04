package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/pqap/services/execution-engine/metrics"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type FillMonitor struct {
	orderPort    ports.OrderPort
	publisher    ports.EventPublisher
	pollInterval time.Duration
	pollTimeout  time.Duration
	logger       *zap.Logger

	activeOrders   map[string]bool
	mu             sync.Mutex
	lastFillQty    map[string]decimal.Decimal
}

func NewFillMonitor(
	orderPort ports.OrderPort,
	publisher ports.EventPublisher,
	pollInterval time.Duration,
	pollTimeout time.Duration,
	logger *zap.Logger,
) *FillMonitor {
	return &FillMonitor{
		orderPort:    orderPort,
		publisher:    publisher,
		pollInterval: pollInterval,
		pollTimeout:  pollTimeout,
		logger:       logger,
		activeOrders: make(map[string]bool),
		lastFillQty:  make(map[string]decimal.Decimal),
	}
}

func (fm *FillMonitor) MonitorOrder(ctx context.Context, order *ports.Order) {
	startTime := time.Now()
	ticker := time.NewTicker(fm.pollInterval)
	defer ticker.Stop()

	fm.logger.Info("monitoring order",
		zap.String("order_id", order.ID),
		zap.String("client_order_id", order.ClientOrderID),
	)

	for {
		select {
		case <-ctx.Done():
			fm.cleanupOrder(order.ID)
			return
		case <-ticker.C:
			if time.Since(startTime) > fm.pollTimeout {
				fm.handleTimeout(ctx, order)
				fm.cleanupOrder(order.ID)
				return
			}

			status, err := fm.orderPort.GetOrderStatus(order.ID)
			if err != nil {
				fm.logger.Error("failed to get order status",
					zap.String("order_id", order.ID),
					zap.Error(err),
				)
				continue
			}

			switch status.Status {
			case "FILLED":
				fm.handleFill(ctx, order, status)
				fm.cleanupOrder(order.ID)
				return
			case "PARTIAL_FILL":
				fm.handlePartialFill(ctx, order, status)
			case "CANCELLED":
				fm.handleCancel(ctx, order)
				fm.cleanupOrder(order.ID)
				return
			case "FAILED":
				fm.handleFailure(ctx, order)
				fm.cleanupOrder(order.ID)
				return
			default:
				fm.logger.Warn("unknown order status received",
					zap.String("order_id", order.ID),
					zap.String("status", status.Status),
				)
				fm.handleUnknownStatus(ctx, order, status)
				fm.cleanupOrder(order.ID)
				return
			}
		}
	}
}

func (fm *FillMonitor) cleanupOrder(orderID string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	delete(fm.activeOrders, orderID)
	delete(fm.lastFillQty, orderID)
}

func (fm *FillMonitor) decrementActiveOrders(orderID string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	if fm.activeOrders[orderID] {
		metrics.ActiveOrders.Dec()
		fm.activeOrders[orderID] = false
	}
}

func (fm *FillMonitor) incrementActiveOrders(orderID string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.activeOrders[orderID] = true
	metrics.ActiveOrders.Inc()
}

func (fm *FillMonitor) handleFill(ctx context.Context, order *ports.Order, status *ports.OrderStatusResponse) {
	fillLatencyMs := time.Since(order.PlacedAt).Milliseconds()

	metrics.OrdersFilled.Inc()
	fm.decrementActiveOrders(order.ID)
	metrics.FillLatency.Observe(float64(fillLatencyMs))

	fm.logger.Info("order filled",
		zap.String("order_id", order.ID),
		zap.String("filled_qty", status.FilledQty.String()),
		zap.Int64("fill_latency_ms", fillLatencyMs),
	)

	filledEvent := ports.OrderFilled{
		EventID:   uuid.New().String(),
		EventType: "OrderFilled",
		Timestamp: time.Now().UTC(),
		Source:    "execution-engine",
		Payload: ports.OrderFilledPayload{
			OrderID:       order.ID,
			ClientOrderID: order.ClientOrderID,
			OpportunityID: order.OpportunityID,
			MarketID:      order.MarketID,
			MarketSlug:    order.MarketSlug,
			Side:          order.Side,
			Price:         status.Price,
			FilledQty:     status.FilledQty,
			LatencyMs:     fillLatencyMs,
			StrategyID:    order.StrategyID,
		},
	}

	if err := fm.publisher.PublishOrderFilled(ctx, filledEvent); err != nil {
		fm.logger.Error("failed to publish OrderFilled event", zap.Error(err))
	}
}

func (fm *FillMonitor) handlePartialFill(ctx context.Context, order *ports.Order, status *ports.OrderStatusResponse) {
	fm.mu.Lock()
	lastQty, exists := fm.lastFillQty[order.ID]
	fm.mu.Unlock()

	if exists && lastQty.Equal(status.FilledQty) {
		return
	}

	fm.mu.Lock()
	fm.lastFillQty[order.ID] = status.FilledQty
	fm.mu.Unlock()

	metrics.PartialFills.Inc()

	fm.logger.Info("order partially filled",
		zap.String("order_id", order.ID),
		zap.String("filled_qty", status.FilledQty.String()),
		zap.String("remaining_qty", status.RemainingQty.String()),
	)

	partialEvent := ports.OrderPartialFill{
		EventID:   uuid.New().String(),
		EventType: "OrderPartialFill",
		Timestamp: time.Now().UTC(),
		Source:    "execution-engine",
		Payload: ports.OrderPartialFillPayload{
			OrderID:       order.ID,
			ClientOrderID: order.ClientOrderID,
			OpportunityID: order.OpportunityID,
			MarketID:      order.MarketID,
			Side:          order.Side,
			Price:         status.Price,
			FilledQty:     status.FilledQty,
			RemainingQty:  status.RemainingQty,
			StrategyID:    order.StrategyID,
		},
	}

	if err := fm.publisher.PublishOrderPartialFill(ctx, partialEvent); err != nil {
		fm.logger.Error("failed to publish OrderPartialFill event", zap.Error(err))
	}
}

func (fm *FillMonitor) handleCancel(ctx context.Context, order *ports.Order) {
	metrics.OrdersCancelled.Inc()
	fm.decrementActiveOrders(order.ID)

	fm.logger.Info("order cancelled",
		zap.String("order_id", order.ID),
	)

	cancelledEvent := ports.OrderCancelled{
		EventID:   uuid.New().String(),
		EventType: "OrderCancelled",
		Timestamp: time.Now().UTC(),
		Source:    "execution-engine",
		Payload: ports.OrderCancelledPayload{
			OrderID:       order.ID,
			ClientOrderID: order.ClientOrderID,
			OpportunityID: order.OpportunityID,
			MarketID:      order.MarketID,
			Reason:        "cancelled",
			StrategyID:    order.StrategyID,
		},
	}

	if err := fm.publisher.PublishOrderCancelled(ctx, cancelledEvent); err != nil {
		fm.logger.Error("failed to publish OrderCancelled event", zap.Error(err))
	}
}

func (fm *FillMonitor) handleTimeout(ctx context.Context, order *ports.Order) {
	metrics.OrdersCancelled.Inc()
	fm.decrementActiveOrders(order.ID)

	fm.logger.Warn("order monitoring timed out",
		zap.String("order_id", order.ID),
		zap.Duration("timeout", fm.pollTimeout),
	)

	cancelledEvent := ports.OrderCancelled{
		EventID:   uuid.New().String(),
		EventType: "OrderCancelled",
		Timestamp: time.Now().UTC(),
		Source:    "execution-engine",
		Payload: ports.OrderCancelledPayload{
			OrderID:       order.ID,
			ClientOrderID: order.ClientOrderID,
			OpportunityID: order.OpportunityID,
			MarketID:      order.MarketID,
			Reason:        "timeout",
			StrategyID:    order.StrategyID,
		},
	}

	if err := fm.publisher.PublishOrderCancelled(ctx, cancelledEvent); err != nil {
		fm.logger.Error("failed to publish OrderCancelled event", zap.Error(err))
	}
}

func (fm *FillMonitor) handleFailure(ctx context.Context, order *ports.Order) {
	metrics.OrdersFailed.Inc()
	fm.decrementActiveOrders(order.ID)

	fm.logger.Error("order failed",
		zap.String("order_id", order.ID),
	)

	failedEvent := ports.OrderFailed{
		EventID:   uuid.New().String(),
		EventType: "OrderFailed",
		Timestamp: time.Now().UTC(),
		Source:    "execution-engine",
		Payload: ports.OrderFailedPayload{
			OrderID:       order.ID,
			ClientOrderID: order.ClientOrderID,
			OpportunityID: order.OpportunityID,
			MarketID:      order.MarketID,
			Reason:        "order_failed",
			StrategyID:    order.StrategyID,
		},
	}

	if err := fm.publisher.PublishOrderFailed(ctx, failedEvent); err != nil {
		fm.logger.Error("failed to publish OrderFailed event", zap.Error(err))
	}
}

func (fm *FillMonitor) handleUnknownStatus(ctx context.Context, order *ports.Order, status *ports.OrderStatusResponse) {
	metrics.OrdersFailed.Inc()
	fm.decrementActiveOrders(order.ID)

	fm.logger.Error("order has unknown status, treating as failure",
		zap.String("order_id", order.ID),
		zap.String("status", status.Status),
	)

	failedEvent := ports.OrderFailed{
		EventID:   uuid.New().String(),
		EventType: "OrderFailed",
		Timestamp: time.Now().UTC(),
		Source:    "execution-engine",
		Payload: ports.OrderFailedPayload{
			OrderID:       order.ID,
			ClientOrderID: order.ClientOrderID,
			OpportunityID: order.OpportunityID,
			MarketID:      order.MarketID,
			Reason:        "unknown_status",
			ErrorDetail:   "status: " + status.Status,
			StrategyID:    order.StrategyID,
		},
	}

	if err := fm.publisher.PublishOrderFailed(ctx, failedEvent); err != nil {
		fm.logger.Error("failed to publish OrderFailed event", zap.Error(err))
	}
}

func NewOrder(id, clientOrderID, opportunityID, marketID, side, strategyID string, price, size decimal.Decimal) *ports.Order {
	return &ports.Order{
		ID:            id,
		ClientOrderID: clientOrderID,
		OpportunityID: opportunityID,
		MarketID:      marketID,
		Side:          side,
		Price:         price,
		Size:          size,
		FilledQty:     decimal.Zero,
		RemainingQty:  size,
		Status:        ports.OrderStatusPlaced,
		TimeInForce:   "GTC",
		StrategyID:    strategyID,
		PlacedAt:      time.Now().UTC(),
	}
}
