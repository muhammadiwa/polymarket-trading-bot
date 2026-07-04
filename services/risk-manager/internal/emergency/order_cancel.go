package emergency

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/pqap/services/risk-manager/metrics"
	"go.uber.org/zap"
)

type OrderCanceler struct {
	publisher ports.EventPublisher
	timeout   time.Duration
	logger    *zap.Logger
}

func NewOrderCanceler(publisher ports.EventPublisher, timeout time.Duration, logger *zap.Logger) *OrderCanceler {
	return &OrderCanceler{
		publisher: publisher,
		timeout:   timeout,
		logger:    logger,
	}
}

type CancelResult struct {
	RequestedAt time.Time
	CompletedAt time.Time
	LatencyMs   int64
	Success     bool
	Error       error
}

// CancelAll publishes a CancelAllOrders event for downstream consumers.
// #17: Fire-and-forget — returns after publishing; actual cancellation is async.
func (oc *OrderCanceler) CancelAll(ctx context.Context, reason string) (*CancelResult, error) {
	start := time.Now()

	event := ports.CancelAllOrders{
		EventID:   uuid.New().String(),
		EventType: "CancelAllOrders",
		Timestamp: start,
		Source:    "risk-manager",
		Payload: ports.CancelAllOrdersPayload{
			Reason:      reason,
			RequestedBy: "emergency_stop",
		},
	}

	if err := oc.publisher.PublishCancelAllOrders(ctx, event); err != nil {
		oc.logger.Error("failed to publish cancel all orders", zap.Error(err))
		metrics.OrderCancelLatency.Observe(float64(time.Since(start).Milliseconds()))
		return &CancelResult{
			RequestedAt: start,
			CompletedAt: time.Now().UTC(),
			LatencyMs:   time.Since(start).Milliseconds(),
			Success:     false,
			Error:       err,
		}, err
	}

	oc.logger.Info("cancel all orders published",
		zap.String("event_id", event.EventID),
		zap.String("reason", reason),
	)

	elapsed := time.Since(start)
	metrics.OrderCancelLatency.Observe(float64(elapsed.Milliseconds()))

	return &CancelResult{
		RequestedAt: start,
		CompletedAt: time.Now().UTC(),
		LatencyMs:   elapsed.Milliseconds(),
		Success:     true,
	}, nil
}
