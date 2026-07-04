package circuitbreaker

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/pqap/services/execution-engine/metrics"
	"go.uber.org/zap"
)

type State int

const (
	StateClosed   State = 0
	StateOpen     State = 1
	StateHalfOpen State = 2
)

type CircuitBreaker struct {
	mu                sync.RWMutex
	state             State
	consecutiveErrors int
	threshold         int
	cooldown          time.Duration
	probeTimeout      time.Duration
	lastError         string
	lastErrorTime     time.Time
	trippedAt         *time.Time
	resumedAt         *time.Time
	totalTrips        int64
	halted            bool
	singleProbe       atomic.Bool
	logger            *zap.Logger
	publisher         ports.EventPublisher
	onTrip            func(lastError string, consecutiveErrors int)
}

func NewCircuitBreaker(
	threshold int,
	cooldown time.Duration,
	probeTimeout time.Duration,
	logger *zap.Logger,
	publisher ports.EventPublisher,
	onTrip func(lastError string, consecutiveErrors int),
) (*CircuitBreaker, error) {
	if threshold < 1 {
		return nil, fmt.Errorf("circuit breaker threshold must be >= 1, got %d", threshold)
	}
	if cooldown <= 0 {
		return nil, fmt.Errorf("circuit breaker cooldown must be > 0, got %v", cooldown)
	}
	if probeTimeout <= 0 {
		return nil, fmt.Errorf("circuit breaker probe timeout must be > 0, got %v", probeTimeout)
	}

	return &CircuitBreaker{
		state:        StateClosed,
		threshold:    threshold,
		cooldown:     cooldown,
		probeTimeout: probeTimeout,
		logger:       logger,
		publisher:    publisher,
		onTrip:       onTrip,
	}, nil
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.AllowRequest() {
		return ErrCircuitBreakerOpen
	}

	err := fn()

	if err != nil {
		cb.RecordFailure(err.Error())
		return err
	}

	cb.RecordSuccess()
	return nil
}

func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.halted {
		return false
	}

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastErrorTime) > cb.cooldown {
			cb.state = StateHalfOpen
			cb.singleProbe.Store(false)
			metrics.CircuitBreakerState.Set(float64(StateHalfOpen))
			cb.logger.Info("circuit breaker transitioning to half-open")
			return true
		}
		remaining := cb.cooldown - time.Since(cb.lastErrorTime)
		metrics.CircuitBreakerCooldownRemaining.Set(float64(remaining.Milliseconds()))
		return false
	case StateHalfOpen:
		if cb.singleProbe.CompareAndSwap(false, true) {
			return true
		}
		return false
	}
	return false
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen {
		cb.state = StateClosed
		cb.consecutiveErrors = 0
		cb.lastError = ""
		cb.singleProbe.Store(false)
		now := time.Now().UTC()
		cb.resumedAt = &now
		metrics.CircuitBreakerState.Set(float64(StateClosed))
		metrics.CircuitBreakerConsecutiveErrors.Set(0)
		cb.logger.Info("circuit breaker closed after successful probe")
	}
}

func (cb *CircuitBreaker) RecordFailure(errMsg string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveErrors++
	cb.lastError = errMsg
	cb.lastErrorTime = time.Now()

	metrics.CircuitBreakerConsecutiveErrors.Set(float64(cb.consecutiveErrors))

	if cb.consecutiveErrors >= cb.threshold {
		cb.state = StateOpen
		cb.totalTrips++
		now := time.Now().UTC()
		cb.trippedAt = &now

		metrics.CircuitBreakerState.Set(float64(StateOpen))
		metrics.CircuitBreakerTrips.Inc()

		cb.logger.Error("circuit breaker tripped",
			zap.Int("consecutive_errors", cb.consecutiveErrors),
			zap.Int("threshold", cb.threshold),
			zap.String("last_error", cb.lastError),
		)

		if cb.publisher != nil {
			cb.publishTrippedEvent()
		}

		if cb.onTrip != nil {
			currentError := cb.lastError
			currentConsecutive := cb.consecutiveErrors
			go func() {
				defer func() {
					if r := recover(); r != nil {
						cb.logger.Error("panic in onTrip callback",
							zap.Any("recover", r),
						)
					}
				}()
				cb.onTrip(currentError, currentConsecutive)
			}()
		}
	}
}

func (cb *CircuitBreaker) publishTrippedEvent() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cooldownSeconds := int(cb.cooldown.Seconds())

	sanitizedError := cb.lastError
	if len(sanitizedError) > 200 {
		sanitizedError = sanitizedError[:200] + "..."
	}
	sanitizedError = strings.ReplaceAll(sanitizedError, "\n", " ")
	sanitizedError = strings.ReplaceAll(sanitizedError, "\r", " ")

	event := ports.CircuitBreakerTripped{
		EventID:   uuid.New().String(),
		EventType: "CircuitBreakerTripped",
		Timestamp: time.Now().UTC(),
		Source:    "execution-engine",
		Payload: ports.CircuitBreakerTrippedPayload{
			ConsecutiveErrors: cb.consecutiveErrors,
			LastError:         cb.lastError,
			LastErrorTime:     cb.lastErrorTime,
			CooldownSeconds:   cooldownSeconds,
			Message:           fmt.Sprintf("%d consecutive API errors. Trading halted. Manual resume required.", cb.consecutiveErrors),
		},
	}

	if err := cb.publisher.PublishCircuitBreakerTripped(ctx, event); err != nil {
		cb.logger.Error("failed to publish CircuitBreakerTripped event", zap.Error(err))
	}

	notification := ports.NotificationRequest{
		EventID:   uuid.New().String(),
		EventType: "NotificationRequest",
		Timestamp: time.Now().UTC(),
		Source:    "execution-engine",
		Payload: ports.NotificationRequestPayload{
			Category:       "CRITICAL",
			Title:          "CIRCUIT BREAKER TRIPPED",
			Message:        fmt.Sprintf("%d consecutive API errors. Trading halted. Manual resume required.", cb.consecutiveErrors),
			Channel:        "telegram",
			Priority:       "high",
			BypassThrottle: true,
		},
	}

	if err := cb.publisher.PublishNotificationRequest(ctx, notification); err != nil {
		cb.logger.Error("failed to publish notification request", zap.Error(err))
	}
}

func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) GetConsecutiveErrors() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.consecutiveErrors
}

func (cb *CircuitBreaker) GetLastError() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.lastError
}

func (cb *CircuitBreaker) GetLastErrorTime() time.Time {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.lastErrorTime
}

func (cb *CircuitBreaker) GetTrippedAt() *time.Time {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.trippedAt
}

func (cb *CircuitBreaker) GetResumedAt() *time.Time {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.resumedAt
}

func (cb *CircuitBreaker) GetTotalTrips() int64 {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.totalTrips
}

func (cb *CircuitBreaker) GetCooldownRemaining() time.Duration {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.state != StateOpen {
		return 0
	}

	remaining := cb.cooldown - time.Since(cb.lastErrorTime)
	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.consecutiveErrors = 0
	cb.lastError = ""
	cb.halted = false
	now := time.Now().UTC()
	cb.resumedAt = &now

	metrics.CircuitBreakerState.Set(float64(StateClosed))
	metrics.CircuitBreakerConsecutiveErrors.Set(0)
	metrics.CircuitBreakerCooldownRemaining.Set(0)

	cb.logger.Info("circuit breaker manually reset")
}

func (cb *CircuitBreaker) Halt() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.halted = true
	cb.state = StateOpen
	cb.totalTrips++
	now := time.Now().UTC()
	cb.trippedAt = &now
	cb.lastError = "emergency_halt"
	cb.lastErrorTime = now

	metrics.CircuitBreakerState.Set(float64(StateOpen))
	metrics.CircuitBreakerTrips.Inc()

	cb.logger.Error("circuit breaker halted permanently (emergency stop)")
}

func (cb *CircuitBreaker) Resume(reason, userID string) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.halted {
		cb.halted = false
	}

	if cb.state != StateOpen && !cb.halted {
		return fmt.Errorf("circuit breaker is not in OPEN state, current state: %d", cb.state)
	}

	cb.state = StateClosed
	cb.consecutiveErrors = 0
	cb.lastError = ""
	now := time.Now().UTC()
	cb.resumedAt = &now

	metrics.CircuitBreakerState.Set(float64(StateClosed))
	metrics.CircuitBreakerConsecutiveErrors.Set(0)
	metrics.CircuitBreakerCooldownRemaining.Set(0)

	cb.logger.Info("circuit breaker resumed",
		zap.String("reason", reason),
		zap.String("user_id", userID),
	)

	if cb.publisher != nil {
		cb.publishResumedEvent(reason, userID)
	}

	return nil
}

func (cb *CircuitBreaker) publishResumedEvent(reason, userID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	event := ports.CircuitBreakerResumed{
		EventID:   uuid.New().String(),
		EventType: "CircuitBreakerResumed",
		Timestamp: time.Now().UTC(),
		Source:    "execution-engine",
		Payload: ports.CircuitBreakerResumedPayload{
			Reason: reason,
			UserID: userID,
		},
	}

	if err := cb.publisher.PublishCircuitBreakerResumed(ctx, event); err != nil {
		cb.logger.Error("failed to publish CircuitBreakerResumed event", zap.Error(err))
	}
}

var ErrCircuitBreakerOpen = &CircuitBreakerError{}

type CircuitBreakerError struct{}

func (e *CircuitBreakerError) Error() string {
	return "circuit breaker is open"
}
