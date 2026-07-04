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

type mockNotificationPublisher struct {
	notifications []ports.NotificationRequest
	trippedEvents []ports.CircuitBreakerTripped
}

func (m *mockNotificationPublisher) PublishOrderPlaced(ctx context.Context, event ports.OrderPlaced) error {
	return nil
}

func (m *mockNotificationPublisher) PublishOrderFilled(ctx context.Context, event ports.OrderFilled) error {
	return nil
}

func (m *mockNotificationPublisher) PublishOrderPartialFill(ctx context.Context, event ports.OrderPartialFill) error {
	return nil
}

func (m *mockNotificationPublisher) PublishOrderCancelled(ctx context.Context, event ports.OrderCancelled) error {
	return nil
}

func (m *mockNotificationPublisher) PublishOrderFailed(ctx context.Context, event ports.OrderFailed) error {
	return nil
}

func (m *mockNotificationPublisher) PublishRiskAlert(ctx context.Context, event ports.RiskAlert) error {
	return nil
}

func (m *mockNotificationPublisher) PublishAtomicLegFailed(ctx context.Context, event ports.AtomicLegFailed) error {
	return nil
}

func (m *mockNotificationPublisher) PublishCircuitBreakerTripped(ctx context.Context, event ports.CircuitBreakerTripped) error {
	m.trippedEvents = append(m.trippedEvents, event)
	return nil
}

func (m *mockNotificationPublisher) PublishCircuitBreakerResumed(ctx context.Context, event ports.CircuitBreakerResumed) error {
	return nil
}

func (m *mockNotificationPublisher) PublishNotificationRequest(ctx context.Context, event ports.NotificationRequest) error {
	m.notifications = append(m.notifications, event)
	return nil
}

func (m *mockNotificationPublisher) PublishTradeRecorded(ctx context.Context, event ports.TradeRecorded) error {
	return nil
}

func (m *mockNotificationPublisher) Close() error {
	return nil
}

func TestNotification_CircuitBreakerTrip(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockNotificationPublisher{}

	cb, err := circuitbreaker.NewCircuitBreaker(2, 1*time.Second, 5*time.Second, log, mockPub, nil)
	if err != nil {
		t.Fatalf("unexpected error creating circuit breaker: %v", err)
	}

	testErr := errors.New("API timeout")
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	if len(mockPub.notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(mockPub.notifications))
	}

	notif := mockPub.notifications[0]

	if notif.Payload.Category != "CRITICAL" {
		t.Errorf("category = %s, want CRITICAL", notif.Payload.Category)
	}

	if notif.Payload.Title != "CIRCUIT BREAKER TRIPPED" {
		t.Errorf("title = %s, want CIRCUIT BREAKER TRIPPED", notif.Payload.Title)
	}

	if !notif.Payload.BypassThrottle {
		t.Error("expected BypassThrottle to be true")
	}

	if notif.Payload.Channel != "telegram" {
		t.Errorf("channel = %s, want telegram", notif.Payload.Channel)
	}

	if notif.Payload.Priority != "high" {
		t.Errorf("priority = %s, want high", notif.Payload.Priority)
	}

	if notif.Source != "execution-engine" {
		t.Errorf("source = %s, want execution-engine", notif.Source)
	}

	if notif.EventType != "NotificationRequest" {
		t.Errorf("event_type = %s, want NotificationRequest", notif.EventType)
	}
}

func TestNotification_MessageFormat(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockNotificationPublisher{}

	cb, err := circuitbreaker.NewCircuitBreaker(3, 1*time.Second, 5*time.Second, log, mockPub, nil)
	if err != nil {
		t.Fatalf("unexpected error creating circuit breaker: %v", err)
	}

	testErr := errors.New("connection refused")
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	if len(mockPub.notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(mockPub.notifications))
	}

	msg := mockPub.notifications[0].Payload.Message

	if msg == "" {
		t.Error("message should not be empty")
	}

	if len(msg) < 10 {
		t.Errorf("message too short: %s", msg)
	}
}

func TestNotification_CircuitBreakerTrippedEvent(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockNotificationPublisher{}

	cb, err := circuitbreaker.NewCircuitBreaker(2, 1*time.Second, 5*time.Second, log, mockPub, nil)
	if err != nil {
		t.Fatalf("unexpected error creating circuit breaker: %v", err)
	}

	testErr := errors.New("API error")
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	if len(mockPub.trippedEvents) != 1 {
		t.Fatalf("expected 1 tripped event, got %d", len(mockPub.trippedEvents))
	}

	event := mockPub.trippedEvents[0]

	if event.EventType != "CircuitBreakerTripped" {
		t.Errorf("event_type = %s, want CircuitBreakerTripped", event.EventType)
	}

	if event.Source != "execution-engine" {
		t.Errorf("source = %s, want execution-engine", event.Source)
	}

	if event.Payload.ConsecutiveErrors != 2 {
		t.Errorf("consecutive_errors = %d, want 2", event.Payload.ConsecutiveErrors)
	}

	if event.Payload.LastError != "API error" {
		t.Errorf("last_error = %s, want API error", event.Payload.LastError)
	}

	if event.Payload.CooldownSeconds != 1 {
		t.Errorf("cooldown_seconds = %d, want 1", event.Payload.CooldownSeconds)
	}
}

func TestNotification_BypassThrottle(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockNotificationPublisher{}

	cb, err := circuitbreaker.NewCircuitBreaker(1, 1*time.Second, 5*time.Second, log, mockPub, nil)
	if err != nil {
		t.Fatalf("unexpected error creating circuit breaker: %v", err)
	}

	testErr := errors.New("error")
	_ = cb.Execute(func() error {
		return testErr
	})

	if len(mockPub.notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(mockPub.notifications))
	}

	if !mockPub.notifications[0].Payload.BypassThrottle {
		t.Error("expected BypassThrottle to be true for critical notifications")
	}
}
