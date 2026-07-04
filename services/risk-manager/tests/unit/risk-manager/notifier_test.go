package riskmanager

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pqap/services/risk-manager/internal/emergency"
	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type capturePublisher struct {
	mockPublisher
	lastNotification *ports.NotificationRequest
}

func (c *capturePublisher) PublishNotificationRequest(ctx context.Context, event ports.NotificationRequest) error {
	c.lastNotification = &event
	return c.mockPublisher.PublishNotificationRequest(ctx, event)
}

func TestNotifier_SendEmergencyAlert(t *testing.T) {
	pub := &capturePublisher{}
	logger, _ := zap.NewDevelopment()
	n := emergency.NewNotifier(pub, logger)

	drawdown := decimal.NewFromFloat(0.12)
	limit := decimal.NewFromFloat(0.10)
	peak := decimal.NewFromFloat(50000)
	current := decimal.NewFromFloat(44000)

	details := &emergency.EmergencyDetails{
		Reason:          "drawdown_exceeded",
		Drawdown:        &drawdown,
		DrawdownLimit:   &limit,
		PeakEquity:      &peak,
		CurrentEquity:   &current,
		DailyPnL:        decimal.NewFromFloat(-150),
		OpenOrdersCount: 3,
		TriggeredAt:     testTime(),
	}

	ctx := context.Background()
	err := n.SendEmergencyAlert(ctx, details)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pub.notificationCalled.Load() != 1 {
		t.Errorf("expected 1 notification, got %d", pub.notificationCalled.Load())
	}

	if pub.lastNotification == nil {
		t.Fatal("expected notification to be captured")
	}

	payload := pub.lastNotification.Payload
	if payload.Priority != "critical" {
		t.Errorf("expected priority 'critical', got %s", payload.Priority)
	}
	if !payload.BypassThrottle {
		t.Error("expected bypass_throttle to be true")
	}
	if payload.Title != "EMERGENCY STOP TRIGGERED" {
		t.Errorf("expected title 'EMERGENCY STOP TRIGGERED', got %s", payload.Title)
	}
	if payload.Channel != "telegram" {
		t.Errorf("expected channel 'telegram', got %s", payload.Channel)
	}
	if payload.Category != "risk" {
		t.Errorf("expected category 'risk', got %s", payload.Category)
	}
}

func TestNotifier_SendEmergencyAlert_BypassesThrottling(t *testing.T) {
	pub := &capturePublisher{}
	logger, _ := zap.NewDevelopment()
	n := emergency.NewNotifier(pub, logger)

	details := &emergency.EmergencyDetails{
		Reason:      "manual",
		DailyPnL:   decimal.Zero,
		TriggeredAt: testTime(),
	}

	ctx := context.Background()
	n.SendEmergencyAlert(ctx, details)

	if pub.lastNotification == nil {
		t.Fatal("expected notification")
	}
	if !pub.lastNotification.Payload.BypassThrottle {
		t.Error("emergency notification must bypass throttling")
	}
}

func TestNotifier_SendResumeAlert(t *testing.T) {
	pub := &capturePublisher{}
	logger, _ := zap.NewDevelopment()
	n := emergency.NewNotifier(pub, logger)

	ctx := context.Background()
	err := n.SendResumeAlert(ctx, "drawdown_exceeded", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pub.lastNotification == nil {
		t.Fatal("expected notification")
	}

	payload := pub.lastNotification.Payload
	if payload.Priority != "high" {
		t.Errorf("expected priority 'high', got %s", payload.Priority)
	}
	if !payload.BypassThrottle {
		t.Error("expected bypass_throttle to be true for resume")
	}
}

func TestNotifier_EmergencyAlertMessageContainsFields(t *testing.T) {
	pub := &capturePublisher{}
	logger, _ := zap.NewDevelopment()
	n := emergency.NewNotifier(pub, logger)

	drawdown := decimal.NewFromFloat(0.1234)
	limit := decimal.NewFromFloat(0.10)
	peak := decimal.NewFromFloat(50000)
	current := decimal.NewFromFloat(43830)

	details := &emergency.EmergencyDetails{
		Reason:          "drawdown_exceeded",
		Drawdown:        &drawdown,
		DrawdownLimit:   &limit,
		PeakEquity:      &peak,
		CurrentEquity:   &current,
		DailyPnL:        decimal.NewFromFloat(-150),
		OpenOrdersCount: 3,
		TriggeredAt:     testTime(),
	}

	ctx := context.Background()
	n.SendEmergencyAlert(ctx, details)

	if pub.lastNotification == nil {
		t.Fatal("expected notification")
	}

	msg := pub.lastNotification.Payload.Message
	assertContains(t, msg, "drawdown_exceeded")
	assertContains(t, msg, "Drawdown")
	assertContains(t, msg, "Peak Equity")
	assertContains(t, msg, "Current Equity")
	assertContains(t, msg, "Daily PnL")
	assertContains(t, msg, "Open Orders Cancelled: 3")
	assertContains(t, msg, "Manual resume required")
}

func testTime() time.Time {
	return time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected string to contain %q, got: %s", substr, s)
	}
}
