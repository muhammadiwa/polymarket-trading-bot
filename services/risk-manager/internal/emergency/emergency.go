package emergency

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/pqap/services/risk-manager/metrics"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type EmergencyStop struct {
	mu              sync.RWMutex
	active          bool
	reason          string
	activatedAt     *time.Time
	publisher       ports.EventPublisher
	logger          *zap.Logger
	onActivate      func()
	onResume        func()
	orderCanceler   *OrderCanceler
	notifier        *Notifier
}

func NewEmergencyStop(publisher ports.EventPublisher, logger *zap.Logger) *EmergencyStop {
	return &EmergencyStop{
		publisher: publisher,
		logger:    logger,
		notifier:  NewNotifier(publisher, logger),
	}
}

func (e *EmergencyStop) SetOrderCanceler(oc *OrderCanceler) {
	e.orderCanceler = oc
}

func (e *EmergencyStop) SetCallbacks(onActivate, onResume func()) {
	e.onActivate = onActivate
	e.onResume = onResume
}

func (e *EmergencyStop) Activate(ctx context.Context, reason string) error {
	return e.ActivateWithDetails(ctx, reason, nil, nil, nil, decimal.Zero, 0, nil)
}

func (e *EmergencyStop) ActivateWithDetails(
	ctx context.Context,
	reason string,
	drawdown *decimal.Decimal,
	peakEquity *decimal.Decimal,
	currentEquity *decimal.Decimal,
	dailyPnL decimal.Decimal,
	openOrdersCount int,
	extraContext map[string]interface{},
) error {
	e.mu.Lock()
	if e.active {
		e.mu.Unlock()
		e.logger.Warn("emergency stop already active", zap.String("reason", reason))
		return nil
	}

	e.active = true
	e.reason = reason
	now := time.Now().UTC()
	e.activatedAt = &now
	e.mu.Unlock()

	metrics.EmergencyStopTotal.Inc()
	metrics.EmergencyStopActive.Set(1)

	e.logger.Error("EMERGENCY STOP ACTIVATED",
		zap.String("reason", reason),
		zap.Int("open_orders_count", openOrdersCount),
	)

	payload := ports.EmergencyStopPayload{
		Reason:          reason,
		Drawdown:        drawdown,
		PeakEquity:      peakEquity,
		CurrentEquity:   currentEquity,
		DailyPnL:        dailyPnL,
		OpenOrdersCount: openOrdersCount,
		Context:         extraContext,
	}

	event := ports.EmergencyStop{
		EventID:   uuid.New().String(),
		EventType: "EmergencyStop",
		Timestamp: now,
		Source:    "risk-manager",
		Payload:   payload,
	}

	if err := e.publisher.PublishEmergencyStop(ctx, event); err != nil {
		e.logger.Error("failed to publish emergency stop event", zap.Error(err))
	}

	if e.orderCanceler != nil {
		result, err := e.orderCanceler.CancelAll(ctx, reason)
		if err != nil {
			e.logger.Error("order cancellation failed", zap.Error(err))
		} else if result != nil {
			// #13, #26: Don't overwrite openOrdersCount; use result for logging
			e.logger.Info("order cancellation completed",
				zap.Bool("success", result.Success),
				zap.Int64("latency_ms", result.LatencyMs),
			)
		}
	} else {
		cancelEvent := ports.CancelAllOrders{
			EventID:   uuid.New().String(),
			EventType: "CancelAllOrders",
			Timestamp: now,
			Source:    "risk-manager",
			Payload: ports.CancelAllOrdersPayload{
				Reason:      reason,
				RequestedBy: "emergency_stop",
			},
		}
		if err := e.publisher.PublishCancelAllOrders(ctx, cancelEvent); err != nil {
			e.logger.Error("failed to publish cancel all orders", zap.Error(err))
		}
	}

	if e.notifier != nil {
		details := &EmergencyDetails{
			Reason:          reason,
			Drawdown:        drawdown,
			PeakEquity:      peakEquity,
			CurrentEquity:   currentEquity,
			DailyPnL:        dailyPnL,
			OpenOrdersCount: openOrdersCount,
			TriggeredAt:     now,
		}
		if err := e.notifier.SendEmergencyAlert(ctx, details); err != nil {
			e.logger.Error("failed to send emergency notification", zap.Error(err))
		}
	} else {
		notif := ports.NotificationRequest{
			EventID:   uuid.New().String(),
			EventType: "NotificationRequest",
			Timestamp: now,
			Source:    "risk-manager",
			Payload: ports.NotificationRequestPayload{
				Category:       "risk",
				Title:          "EMERGENCY STOP ACTIVATED",
				Message:        reason,
				Channel:        "telegram",
				Priority:       "critical",
				BypassThrottle: true,
			},
		}
		if err := e.publisher.PublishNotificationRequest(ctx, notif); err != nil {
			e.logger.Error("failed to publish emergency stop notification", zap.Error(err))
		}
	}

	if e.onActivate != nil {
		e.onActivate()
	}

	return nil
}

func (e *EmergencyStop) Resume(ctx context.Context) error {
	return e.ResumeByUser(ctx, "manual")
}

func (e *EmergencyStop) ResumeByUser(ctx context.Context, resumedBy string) error {
	e.mu.Lock()
	if !e.active {
		e.mu.Unlock()
		e.logger.Warn("emergency stop not active, nothing to resume")
		return nil
	}

	previousReason := e.reason
	e.active = false
	e.reason = ""
	e.activatedAt = nil
	e.mu.Unlock()

	metrics.EmergencyStopActive.Set(0)
	metrics.ResumeTotal.Inc()

	e.logger.Info("emergency stop resumed - trading can proceed",
		zap.String("previous_reason", previousReason),
		zap.String("resumed_by", resumedBy),
	)

	resumedEvent := ports.TradingResumed{
		EventID:   uuid.New().String(),
		EventType: "TradingResumed",
		Timestamp: time.Now().UTC(),
		Source:    "risk-manager",
		Payload: ports.TradingResumedPayload{
			PreviousReason: previousReason,
			ResumedBy:      resumedBy,
			ResumedAt:      time.Now().UTC(),
		},
	}

	if err := e.publisher.PublishTradingResumed(ctx, resumedEvent); err != nil {
		e.logger.Error("failed to publish trading resumed event", zap.Error(err))
	}

	if e.notifier != nil {
		if err := e.notifier.SendResumeAlert(ctx, previousReason, resumedBy); err != nil {
			e.logger.Error("failed to send resume notification", zap.Error(err))
		}
	}

	if e.onResume != nil {
		e.onResume()
	}

	return nil
}

func (e *EmergencyStop) IsActive() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.active
}

func (e *EmergencyStop) Reason() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.reason
}

func (e *EmergencyStop) ActivatedAt() *time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.activatedAt
}

func (e *EmergencyStop) Duration() time.Duration {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.activatedAt == nil {
		return 0
	}
	return time.Since(*e.activatedAt)
}
