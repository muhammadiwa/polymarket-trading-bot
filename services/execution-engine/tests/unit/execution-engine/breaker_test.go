package circuitbreaker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	circuitbreaker "github.com/pqap/services/execution-engine/internal/circuit_breaker"
	"github.com/pqap/services/execution-engine/internal/ports"
	"go.uber.org/zap"
)

type mockPublisherBreaker struct {
	trippedEvents   []ports.CircuitBreakerTripped
	resumedEvents   []ports.CircuitBreakerResumed
	notifications   []ports.NotificationRequest
}

func (m *mockPublisherBreaker) PublishOrderPlaced(ctx context.Context, event ports.OrderPlaced) error {
	return nil
}

func (m *mockPublisherBreaker) PublishOrderFilled(ctx context.Context, event ports.OrderFilled) error {
	return nil
}

func (m *mockPublisherBreaker) PublishOrderPartialFill(ctx context.Context, event ports.OrderPartialFill) error {
	return nil
}

func (m *mockPublisherBreaker) PublishOrderCancelled(ctx context.Context, event ports.OrderCancelled) error {
	return nil
}

func (m *mockPublisherBreaker) PublishOrderFailed(ctx context.Context, event ports.OrderFailed) error {
	return nil
}

func (m *mockPublisherBreaker) PublishRiskAlert(ctx context.Context, event ports.RiskAlert) error {
	return nil
}

func (m *mockPublisherBreaker) PublishAtomicLegFailed(ctx context.Context, event ports.AtomicLegFailed) error {
	return nil
}

func (m *mockPublisherBreaker) PublishCircuitBreakerTripped(ctx context.Context, event ports.CircuitBreakerTripped) error {
	m.trippedEvents = append(m.trippedEvents, event)
	return nil
}

func (m *mockPublisherBreaker) PublishCircuitBreakerResumed(ctx context.Context, event ports.CircuitBreakerResumed) error {
	m.resumedEvents = append(m.resumedEvents, event)
	return nil
}

func (m *mockPublisherBreaker) PublishNotificationRequest(ctx context.Context, event ports.NotificationRequest) error {
	m.notifications = append(m.notifications, event)
	return nil
}

func (m *mockPublisherBreaker) PublishTradeRecorded(ctx context.Context, event ports.TradeRecorded) error {
	return nil
}

func (m *mockPublisherBreaker) Close() error {
	return nil
}

func TestCircuitBreaker_Closed(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockPublisherBreaker{}
	cb, err := circuitbreaker.NewCircuitBreaker(3, 1*time.Second, 5*time.Second, log, mockPub, nil)
	if err != nil {
		t.Fatalf("unexpected error creating circuit breaker: %v", err)
	}

	if cb.GetState() != circuitbreaker.StateClosed {
		t.Error("expected initial state to be closed")
	}

	err = cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if cb.GetState() != circuitbreaker.StateClosed {
		t.Error("expected state to remain closed after success")
	}
}

func TestCircuitBreaker_Trip(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockPublisherBreaker{}
	tripped := false
	cb, err := circuitbreaker.NewCircuitBreaker(3, 1*time.Second, 5*time.Second, log, mockPub, func(lastError string, consecutiveErrors int) {
		tripped = true
	})
	if err != nil {
		t.Fatalf("unexpected error creating circuit breaker: %v", err)
	}

	testErr := errors.New("test error")

	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	if cb.GetState() != circuitbreaker.StateOpen {
		t.Errorf("state = %d, want open", cb.GetState())
	}

	if !tripped {
		t.Error("expected onTrip callback to be called")
	}

	if cb.GetConsecutiveErrors() != 3 {
		t.Errorf("consecutive errors = %d, want 3", cb.GetConsecutiveErrors())
	}

	if len(mockPub.trippedEvents) != 1 {
		t.Errorf("expected 1 tripped event, got %d", len(mockPub.trippedEvents))
	}

	if len(mockPub.notifications) != 1 {
		t.Errorf("expected 1 notification, got %d", len(mockPub.notifications))
	}

	if mockPub.notifications[0].Payload.Category != "CRITICAL" {
		t.Errorf("notification category = %s, want CRITICAL", mockPub.notifications[0].Payload.Category)
	}

	if !mockPub.notifications[0].Payload.BypassThrottle {
		t.Error("expected BypassThrottle to be true")
	}

	err = cb.Execute(func() error {
		return nil
	})

	if err == nil {
		t.Error("expected error when circuit is open")
	}
}

func TestCircuitBreaker_HalfOpen(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockPublisherBreaker{}
	cb, err := circuitbreaker.NewCircuitBreaker(2, 100*time.Millisecond, 5*time.Second, log, mockPub, nil)
	if err != nil {
		t.Fatalf("unexpected error creating circuit breaker: %v", err)
	}

	testErr := errors.New("test error")

	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	if cb.GetState() != circuitbreaker.StateOpen {
		t.Errorf("state = %d, want open", cb.GetState())
	}

	time.Sleep(150 * time.Millisecond)

	if !cb.AllowRequest() {
		t.Error("expected request to be allowed after cooldown")
	}

	if cb.GetState() != circuitbreaker.StateHalfOpen {
		t.Errorf("state = %d, want half-open", cb.GetState())
	}
}

func TestCircuitBreaker_Recovery(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockPublisherBreaker{}
	cb, err := circuitbreaker.NewCircuitBreaker(2, 100*time.Millisecond, 5*time.Second, log, mockPub, nil)
	if err != nil {
		t.Fatalf("unexpected error creating circuit breaker: %v", err)
	}

	testErr := errors.New("test error")

	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	time.Sleep(150 * time.Millisecond)

	err = cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if cb.GetState() != circuitbreaker.StateClosed {
		t.Errorf("state = %d, want closed", cb.GetState())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockPublisherBreaker{}
	cb, err := circuitbreaker.NewCircuitBreaker(2, 1*time.Second, 5*time.Second, log, mockPub, nil)
	if err != nil {
		t.Fatalf("unexpected error creating circuit breaker: %v", err)
	}

	testErr := errors.New("test error")

	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	if cb.GetState() != circuitbreaker.StateOpen {
		t.Errorf("state = %d, want open", cb.GetState())
	}

	cb.Reset()

	if cb.GetState() != circuitbreaker.StateClosed {
		t.Errorf("state = %d, want closed after reset", cb.GetState())
	}

	if cb.GetConsecutiveErrors() != 0 {
		t.Errorf("consecutive errors = %d, want 0 after reset", cb.GetConsecutiveErrors())
	}
}

func TestCircuitBreaker_Resume(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockPublisherBreaker{}
	cb, err := circuitbreaker.NewCircuitBreaker(2, 1*time.Second, 5*time.Second, log, mockPub, nil)
	if err != nil {
		t.Fatalf("unexpected error creating circuit breaker: %v", err)
	}

	testErr := errors.New("test error")

	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	if cb.GetState() != circuitbreaker.StateOpen {
		t.Errorf("state = %d, want open", cb.GetState())
	}

	err = cb.Resume("manual intervention", "user-1")
	if err != nil {
		t.Fatalf("unexpected error resuming: %v", err)
	}

	if cb.GetState() != circuitbreaker.StateClosed {
		t.Errorf("state = %d, want closed after resume", cb.GetState())
	}

	if cb.GetConsecutiveErrors() != 0 {
		t.Errorf("consecutive errors = %d, want 0 after resume", cb.GetConsecutiveErrors())
	}

	if len(mockPub.resumedEvents) != 1 {
		t.Errorf("expected 1 resumed event, got %d", len(mockPub.resumedEvents))
	}

	if mockPub.resumedEvents[0].Payload.UserID != "user-1" {
		t.Errorf("user_id = %s, want user-1", mockPub.resumedEvents[0].Payload.UserID)
	}
}

func TestCircuitBreaker_ResumeNotOpen(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockPublisherBreaker{}
	cb, err := circuitbreaker.NewCircuitBreaker(2, 1*time.Second, 5*time.Second, log, mockPub, nil)
	if err != nil {
		t.Fatalf("unexpected error creating circuit breaker: %v", err)
	}

	err = cb.Resume("manual intervention", "user-1")
	if err == nil {
		t.Error("expected error resuming when not open")
	}
}

func TestCircuitBreaker_InvalidThreshold(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockPublisherBreaker{}
	_, err := circuitbreaker.NewCircuitBreaker(0, 1*time.Second, 5*time.Second, log, mockPub, nil)
	if err == nil {
		t.Error("expected error for threshold < 1")
	}
}

func TestCircuitBreaker_InvalidCooldown(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockPublisherBreaker{}
	_, err := circuitbreaker.NewCircuitBreaker(3, 0, 5*time.Second, log, mockPub, nil)
	if err == nil {
		t.Error("expected error for cooldown <= 0")
	}
}

func TestCircuitBreaker_InvalidProbeTimeout(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockPublisherBreaker{}
	_, err := circuitbreaker.NewCircuitBreaker(3, 1*time.Second, 0, log, mockPub, nil)
	if err == nil {
		t.Error("expected error for probe timeout <= 0")
	}
}

func TestCircuitBreaker_ConsecutiveErrors(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockPublisherBreaker{}
	cb, err := circuitbreaker.NewCircuitBreaker(5, 1*time.Second, 5*time.Second, log, mockPub, nil)
	if err != nil {
		t.Fatalf("unexpected error creating circuit breaker: %v", err)
	}

	testErr := errors.New("test error")

	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	if cb.GetConsecutiveErrors() != 3 {
		t.Errorf("consecutive errors = %d, want 3", cb.GetConsecutiveErrors())
	}

	if cb.GetLastError() != "test error" {
		t.Errorf("last error = %s, want test error", cb.GetLastError())
	}

	if cb.GetState() != circuitbreaker.StateClosed {
		t.Errorf("state = %d, want closed", cb.GetState())
	}
}
