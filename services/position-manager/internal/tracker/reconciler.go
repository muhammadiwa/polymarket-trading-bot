package tracker

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/position-manager/internal/ports"
	"github.com/pqap/services/position-manager/metrics"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type Reconciler struct {
	repo              ports.PositionRepository
	polymarket        ports.PolymarketPort
	publisher         ports.EventPublisher
	interval          time.Duration
	mismatchThreshold int
	logger            *zap.Logger
	mu                sync.Mutex
	consecutiveErrors int
}

func NewReconciler(
	repo ports.PositionRepository,
	polymarket ports.PolymarketPort,
	publisher ports.EventPublisher,
	interval time.Duration,
	mismatchThreshold int,
	logger *zap.Logger,
) *Reconciler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Reconciler{
		repo:              repo,
		polymarket:        polymarket,
		publisher:         publisher,
		interval:          interval,
		mismatchThreshold: mismatchThreshold,
		logger:            logger,
	}
}

func (r *Reconciler) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	r.logger.Info("starting reconciler", zap.Duration("interval", r.interval))

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("reconciler stopped")
			return
		case <-ticker.C:
			if err := r.Reconcile(ctx); err != nil {
				r.logger.Error("reconciliation failed", zap.Error(err))
			}
		}
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	metrics.ReconciliationTotal.Inc()

	internalPositions, err := r.repo.GetOpenPositions(ctx)
	if err != nil {
		return err
	}

	apiPositions, err := r.polymarket.GetPositions()
	if err != nil {
		r.logger.Error("failed to fetch API positions", zap.Error(err))
		return err
	}

	type reconcilerKey struct {
		MarketID string
		Side     string
	}

	apiMap := make(map[reconcilerKey]*ports.APIPosition)
	for _, ap := range apiPositions {
		apiMap[reconcilerKey{MarketID: ap.MarketID, Side: ap.Side}] = ap
	}

	internalMap := make(map[reconcilerKey]*ports.Position)
	for _, ip := range internalPositions {
		internalMap[reconcilerKey{MarketID: ip.MarketID, Side: ip.Side}] = ip
	}

	mismatchFound := false

	for _, internal := range internalPositions {
		key := reconcilerKey{MarketID: internal.MarketID, Side: internal.Side}
		api, found := apiMap[key]

		if !found {
			r.logMismatch(ctx, &internal.ID, internal.MarketID, "missing",
				internal.Quantity.String(), "0",
				internal.Side, internal.Side)
			mismatchFound = true
			continue
		}

		internalQty, err := decimal.NewFromString(internal.Quantity.String())
		if err != nil {
			r.logger.Error("failed to parse internal quantity", zap.Error(err))
			continue
		}
		apiQty, err := decimal.NewFromString(api.Quantity)
		if err != nil {
			r.logger.Error("failed to parse api quantity", zap.String("quantity", api.Quantity), zap.Error(err))
			continue
		}

		if !internalQty.Equal(apiQty) {
			r.logMismatch(ctx, &internal.ID, internal.MarketID, "quantity",
				internal.Quantity.String(), api.Quantity,
				internal.Side, api.Side)
			mismatchFound = true
		}

		if internal.Side != api.Side {
			r.logMismatch(ctx, &internal.ID, internal.MarketID, "side",
				internal.Side, api.Side,
				internal.Side, api.Side)
			mismatchFound = true
		}
	}

	for _, api := range apiPositions {
		key := reconcilerKey{MarketID: api.MarketID, Side: api.Side}
		if _, found := internalMap[key]; !found {
			r.logMismatch(ctx, nil, api.MarketID, "extra",
				"", api.Quantity,
				"", api.Side)
			mismatchFound = true
		}
	}

	r.mu.Lock()
	if mismatchFound {
		r.consecutiveErrors++
		consecutive := r.consecutiveErrors
		r.mu.Unlock()

		metrics.ReconciliationMismatchesTotal.Inc()
		metrics.ReconciliationConsecutive.Set(float64(consecutive))

		if err := r.repo.IncrementMismatchCount(ctx); err != nil {
			r.logger.Error("failed to increment mismatch count", zap.Error(err))
		}

		r.logger.Warn("reconciliation mismatch detected",
			zap.Int("consecutive_mismatches", consecutive),
		)

		if consecutive >= r.mismatchThreshold {
			r.triggerEmergencyStop(ctx, consecutive)
		}
	} else {
		r.consecutiveErrors = 0
		r.mu.Unlock()

		metrics.ReconciliationConsecutive.Set(0)
		if err := r.repo.ResetMismatchCount(ctx); err != nil {
			r.logger.Error("failed to reset mismatch count", zap.Error(err))
		}
		r.logger.Info("reconciliation passed")
	}

	return nil
}

func (r *Reconciler) logMismatch(ctx context.Context, positionID *string, marketID, mismatchType, internalQty, apiQty, internalSide, apiSide string) {
	internalState, _ := json.Marshal(map[string]string{
		"quantity": internalQty,
		"side":     internalSide,
	})
	apiState, _ := json.Marshal(map[string]string{
		"quantity": apiQty,
		"side":     apiSide,
	})

	r.mu.Lock()
	consecutive := r.consecutiveErrors + 1
	r.mu.Unlock()

	log := &ports.ReconciliationLog{
		PositionID:            positionID,
		MarketID:              marketID,
		MismatchType:          mismatchType,
		InternalState:         internalState,
		APIState:              apiState,
		ConsecutiveMismatches: consecutive,
		Resolved:              false,
		CreatedAt:             time.Now().UTC(),
	}

	if err := r.repo.LogReconciliation(ctx, log); err != nil {
		r.logger.Error("failed to log reconciliation", zap.Error(err))
	}

	mismatchEvent := ports.PositionReconciliationMismatch{
		EventID:   uuid.New().String(),
		EventType: "PositionReconciliationMismatch",
		Timestamp: time.Now().UTC(),
		Source:    "position-manager",
		Payload: ports.PositionReconciliationMismatchPayload{
			PositionID:            "",
			InternalQuantity:      internalQty,
			APIQuantity:           apiQty,
			InternalSide:          internalSide,
			APISide:               apiSide,
			ConsecutiveMismatches: consecutive,
			MismatchType:          mismatchType,
		},
	}
	if positionID != nil {
		mismatchEvent.Payload.PositionID = *positionID
	}

	if err := r.publisher.PublishReconciliationMismatch(ctx, mismatchEvent); err != nil {
		r.logger.Error("failed to publish mismatch event", zap.Error(err))
	}

	notification := ports.NotificationRequest{
		EventID:   uuid.New().String(),
		EventType: "NotificationRequest",
		Timestamp: time.Now().UTC(),
		Source:    "position-manager",
		Payload: ports.NotificationRequestPayload{
			Category:       "reconciliation",
			Title:          "Position Reconciliation Mismatch",
			Message:        "Mismatch detected: " + mismatchType + " on market " + marketID,
			Channel:        "telegram",
			Priority:       "warning",
			BypassThrottle: false,
		},
	}
	if err := r.publisher.PublishNotificationRequest(ctx, notification); err != nil {
		r.logger.Error("failed to send notification", zap.Error(err))
	}
}

func (r *Reconciler) triggerEmergencyStop(ctx context.Context, consecutiveMismatches int) {
	r.logger.Error("EMERGENCY STOP - consecutive mismatch threshold exceeded",
		zap.Int("consecutive_mismatches", consecutiveMismatches),
		zap.Int("threshold", r.mismatchThreshold),
	)

	emergencyEvent := ports.EmergencyStop{
		EventID:   uuid.New().String(),
		EventType: "EmergencyStop",
		Timestamp: time.Now().UTC(),
		Source:    "position-manager",
		Payload: ports.EmergencyStopPayload{
			Reason:                "consecutive reconciliation mismatches exceeded threshold",
			ConsecutiveMismatches: consecutiveMismatches,
		},
	}

	if err := r.publisher.PublishEmergencyStop(ctx, emergencyEvent); err != nil {
		r.logger.Error("failed to publish emergency stop", zap.Error(err))
	}

	notification := ports.NotificationRequest{
		EventID:   uuid.New().String(),
		EventType: "NotificationRequest",
		Timestamp: time.Now().UTC(),
		Source:    "position-manager",
		Payload: ports.NotificationRequestPayload{
			Category:       "emergency",
			Title:          "EMERGENCY STOP TRIGGERED",
			Message:        "Consecutive reconciliation mismatches exceeded threshold",
			Channel:        "telegram",
			Priority:       "critical",
			BypassThrottle: true,
		},
	}
	if err := r.publisher.PublishNotificationRequest(ctx, notification); err != nil {
		r.logger.Error("failed to send emergency notification", zap.Error(err))
	}
}

func (r *Reconciler) GetState() (*ports.ReconciliationState, error) {
	return r.repo.GetReconciliationState(context.Background())
}
