package logger

import (
	"context"
	"time"

	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type AtomicPairRecord struct {
	ID                 string
	OpportunityID      string
	MarketID           string
	YesOrderID         string
	NoOrderID          string
	YesClientOrderID   string
	NoClientOrderID    string
	YesPrice           decimal.Decimal
	NoPrice            decimal.Decimal
	YesSize            decimal.Decimal
	NoSize             decimal.Decimal
	YesFilledQty       decimal.Decimal
	NoFilledQty        decimal.Decimal
	Status             string
	PlacementLatencyMs int64
	FailureReason      string
	FailedLeg          string
	StrategyID         string
	AccountID          *string
	CreatedAt          time.Time
	CompletedAt        *time.Time
}

type PartialFillDBRecord struct {
	PairID        string
	Leg           string
	FilledQty     decimal.Decimal
	RemainingQty  decimal.Decimal
	FillPrice     decimal.Decimal
	OrderID       string
	ClientOrderID string
	MarketID      string
	StrategyID    string
	AccountID     *string
}

type CircuitBreakerEventRecord struct {
	EventType         string
	StateFrom         string
	StateTo           string
	ConsecutiveErrors int
	LastError         string
	CooldownSeconds   int
	UserID            string
	Reason            string
}

type AtomicRepository interface {
	InsertAtomicPair(ctx context.Context, record AtomicPairRecord) error
	UpdateAtomicPairStatus(ctx context.Context, pairID, status string, completedAt *time.Time) error
	InsertPartialFill(ctx context.Context, record PartialFillDBRecord) error
	InsertCircuitBreakerEvent(ctx context.Context, record CircuitBreakerEventRecord) error
}

type AtomicLogger struct {
	repo   AtomicRepository
	logger *zap.Logger
}

func NewAtomicLogger(repo AtomicRepository, logger *zap.Logger) *AtomicLogger {
	return &AtomicLogger{
		repo:   repo,
		logger: logger,
	}
}

func (al *AtomicLogger) LogAtomicPair(ctx context.Context, record AtomicPairRecord) error {
	if err := al.repo.InsertAtomicPair(ctx, record); err != nil {
		al.logger.Error("failed to log atomic pair",
			zap.String("pair_id", record.ID),
			zap.Error(err),
		)
		return err
	}

	al.logger.Debug("atomic pair logged",
		zap.String("pair_id", record.ID),
		zap.String("status", record.Status),
		zap.Int64("placement_latency_ms", record.PlacementLatencyMs),
	)
	return nil
}

func (al *AtomicLogger) UpdatePairStatus(ctx context.Context, pairID, status string, completedAt *time.Time) error {
	if err := al.repo.UpdateAtomicPairStatus(ctx, pairID, status, completedAt); err != nil {
		al.logger.Error("failed to update atomic pair status",
			zap.String("pair_id", pairID),
			zap.String("status", status),
			zap.Error(err),
		)
		return err
	}

	al.logger.Debug("atomic pair status updated",
		zap.String("pair_id", pairID),
		zap.String("status", status),
	)
	return nil
}

func (al *AtomicLogger) LogPartialFill(ctx context.Context, record PartialFillDBRecord) error {
	if err := al.repo.InsertPartialFill(ctx, record); err != nil {
		al.logger.Error("failed to log partial fill",
			zap.String("pair_id", record.PairID),
			zap.String("leg", record.Leg),
			zap.Error(err),
		)
		return err
	}

	al.logger.Debug("partial fill logged",
		zap.String("pair_id", record.PairID),
		zap.String("leg", record.Leg),
		zap.String("filled_qty", record.FilledQty.String()),
	)
	return nil
}

func (al *AtomicLogger) LogCircuitBreakerEvent(ctx context.Context, record CircuitBreakerEventRecord) error {
	if err := al.repo.InsertCircuitBreakerEvent(ctx, record); err != nil {
		al.logger.Error("failed to log circuit breaker event",
			zap.String("event_type", record.EventType),
			zap.Error(err),
		)
		return err
	}

	al.logger.Debug("circuit breaker event logged",
		zap.String("event_type", record.EventType),
		zap.String("state_from", record.StateFrom),
		zap.String("state_to", record.StateTo),
	)
	return nil
}

func (al *AtomicLogger) LogCircuitBreakerTrip(ctx context.Context, consecutiveErrors int, lastError string, cooldownSeconds int) error {
	return al.LogCircuitBreakerEvent(ctx, CircuitBreakerEventRecord{
		EventType:         "TRIPPED",
		StateFrom:         "CLOSED",
		StateTo:           "OPEN",
		ConsecutiveErrors: consecutiveErrors,
		LastError:         lastError,
		CooldownSeconds:   cooldownSeconds,
	})
}

func (al *AtomicLogger) LogCircuitBreakerResume(ctx context.Context, reason, userID string) error {
	return al.LogCircuitBreakerEvent(ctx, CircuitBreakerEventRecord{
		EventType: "RESUMED",
		StateFrom: "OPEN",
		StateTo:   "CLOSED",
		UserID:    userID,
		Reason:    reason,
	})
}

func (al *AtomicLogger) Close() error {
	return nil
}

type AtomicLoggerAdapter struct {
	logger *ports.EventPublisher
}

func (al *AtomicLoggerAdapter) LogOrder(ctx context.Context, order *ports.Order) error {
	return nil
}
