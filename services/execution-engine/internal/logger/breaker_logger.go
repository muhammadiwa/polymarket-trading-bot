package logger

import (
	"context"

	"go.uber.org/zap"
)

type BreakerLogger struct {
	repo   AtomicRepository
	logger *zap.Logger
}

func NewBreakerLogger(repo AtomicRepository, logger *zap.Logger) *BreakerLogger {
	return &BreakerLogger{
		repo:   repo,
		logger: logger,
	}
}

func (bl *BreakerLogger) LogTrip(ctx context.Context, consecutiveErrors int, lastError string, cooldownSeconds int) error {
	record := CircuitBreakerEventRecord{
		EventType:         "TRIPPED",
		StateFrom:         "CLOSED",
		StateTo:           "OPEN",
		ConsecutiveErrors: consecutiveErrors,
		LastError:         lastError,
		CooldownSeconds:   cooldownSeconds,
	}

	if err := bl.repo.InsertCircuitBreakerEvent(ctx, record); err != nil {
		bl.logger.Error("failed to log circuit breaker trip",
			zap.Int("consecutive_errors", consecutiveErrors),
			zap.Error(err),
		)
		return err
	}

	bl.logger.Info("circuit breaker trip logged",
		zap.Int("consecutive_errors", consecutiveErrors),
		zap.String("last_error", lastError),
	)
	return nil
}

func (bl *BreakerLogger) LogResume(ctx context.Context, reason, userID string) error {
	record := CircuitBreakerEventRecord{
		EventType: "RESUMED",
		StateFrom: "OPEN",
		StateTo:   "CLOSED",
		UserID:    userID,
		Reason:    reason,
	}

	if err := bl.repo.InsertCircuitBreakerEvent(ctx, record); err != nil {
		bl.logger.Error("failed to log circuit breaker resume",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return err
	}

	bl.logger.Info("circuit breaker resume logged",
		zap.String("user_id", userID),
		zap.String("reason", reason),
	)
	return nil
}

func (bl *BreakerLogger) Close() error {
	return nil
}
