package executor

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/pqap/services/execution-engine/metrics"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type LegFailureContext struct {
	PairID           string
	OpportunityID    string
	MarketID         string
	FailedLeg        string
	CancelledLeg     string
	CancelledOrderID string
	StrategyID       string
}

type LegHandler struct {
	orderPort        ports.OrderPort
	publisher        ports.EventPublisher
	cancelTimeout    time.Duration
	logger           *zap.Logger
}

func NewLegHandler(
	orderPort ports.OrderPort,
	publisher ports.EventPublisher,
	cancelTimeout time.Duration,
	logger *zap.Logger,
) *LegHandler {
	return &LegHandler{
		orderPort:     orderPort,
		publisher:     publisher,
		cancelTimeout: cancelTimeout,
		logger:        logger,
	}
}

func (lh *LegHandler) HandleLegFailure(ctx context.Context, lfc *LegFailureContext) {
	lh.logger.Warn("handling leg failure",
		zap.String("pair_id", lfc.PairID),
		zap.String("failed_leg", lfc.FailedLeg),
		zap.String("cancelled_leg", lfc.CancelledLeg),
		zap.String("cancelled_order_id", lfc.CancelledOrderID),
	)

	cancelStart := time.Now()

	if lfc.CancelledOrderID != "" {
		cancelCtx, cancel := context.WithTimeout(ctx, lh.cancelTimeout)
		defer cancel()

		err := lh.orderPort.CancelOrder(cancelCtx, lfc.CancelledOrderID)
		cancelLatencyMs := time.Since(cancelStart).Milliseconds()

		metrics.AtomicCancelLatency.Observe(float64(cancelLatencyMs))

		if err != nil {
			lh.logger.Error("failed to cancel other leg",
				zap.String("pair_id", lfc.PairID),
				zap.String("cancelled_order_id", lfc.CancelledOrderID),
				zap.Int64("cancel_latency_ms", cancelLatencyMs),
				zap.Error(err),
			)
		} else {
			lh.logger.Info("cancelled other leg",
				zap.String("pair_id", lfc.PairID),
				zap.String("cancelled_order_id", lfc.CancelledOrderID),
				zap.Int64("cancel_latency_ms", cancelLatencyMs),
			)
		}
	}

	lh.publishAtomicLegFailed(ctx, lfc)
}

func (lh *LegHandler) publishAtomicLegFailed(ctx context.Context, lfc *LegFailureContext) {
	event := ports.AtomicLegFailed{
		EventID:   uuid.New().String(),
		EventType: "AtomicLegFailed",
		Timestamp: time.Now().UTC(),
		Source:    "execution-engine",
		Payload: ports.AtomicLegFailedPayload{
			PairID:              lfc.PairID,
			OpportunityID:       lfc.OpportunityID,
			MarketID:            lfc.MarketID,
			FailedLeg:           lfc.FailedLeg,
			FailedOrderID:       "",
			FailureReason:       "leg_placement_failed",
			SuccessfulLeg:       "",
			SuccessfulOrderID:   "",
			SuccessfulFilledQty: decimal.Zero,
			CancelledLeg:        lfc.CancelledLeg,
			CancelledOrderID:    lfc.CancelledOrderID,
			StrategyID:          lfc.StrategyID,
		},
	}

	if err := lh.publisher.PublishAtomicLegFailed(ctx, event); err != nil {
		lh.logger.Error("failed to publish AtomicLegFailed event",
			zap.String("pair_id", lfc.PairID),
			zap.Error(err),
		)
	}
}
