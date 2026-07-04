package riskmanager

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pqap/services/risk-manager/internal/emergency"
	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type mockPublisher struct {
	emergencyStopCalled    atomic.Int32
	cancelAllCalled        atomic.Int32
	notificationCalled     atomic.Int32
	tradingResumedCalled   atomic.Int32
	drawdownWarningCalled  atomic.Int32
	drawdownResetCalled    atomic.Int32
	riskAlertCalled        atomic.Int32
	riskStateUpdatedCalled atomic.Int32
	riskDecisionLoggedCalled atomic.Int32
	dailyBudgetWarningCalled atomic.Int32
}

func (m *mockPublisher) PublishRiskStateUpdated(ctx context.Context, event ports.RiskStateUpdated) error {
	m.riskStateUpdatedCalled.Add(1)
	return nil
}
func (m *mockPublisher) PublishRiskDecisionLogged(ctx context.Context, event ports.RiskDecisionLogged) error {
	m.riskDecisionLoggedCalled.Add(1)
	return nil
}
func (m *mockPublisher) PublishDailyBudgetWarning(ctx context.Context, event ports.DailyBudgetWarning) error {
	m.dailyBudgetWarningCalled.Add(1)
	return nil
}
func (m *mockPublisher) PublishRiskAlert(ctx context.Context, event ports.RiskAlert) error {
	m.riskAlertCalled.Add(1)
	return nil
}
func (m *mockPublisher) PublishEmergencyStop(ctx context.Context, event ports.EmergencyStop) error {
	m.emergencyStopCalled.Add(1)
	return nil
}
func (m *mockPublisher) PublishNotificationRequest(ctx context.Context, event ports.NotificationRequest) error {
	m.notificationCalled.Add(1)
	return nil
}
func (m *mockPublisher) PublishDrawdownWarning(ctx context.Context, event ports.DrawdownWarning) error {
	m.drawdownWarningCalled.Add(1)
	return nil
}
func (m *mockPublisher) PublishDrawdownReset(ctx context.Context, event ports.DrawdownReset) error {
	m.drawdownResetCalled.Add(1)
	return nil
}
func (m *mockPublisher) PublishTradingResumed(ctx context.Context, event ports.TradingResumed) error {
	m.tradingResumedCalled.Add(1)
	return nil
}
func (m *mockPublisher) PublishCancelAllOrders(ctx context.Context, event ports.CancelAllOrders) error {
	m.cancelAllCalled.Add(1)
	return nil
}
func (m *mockPublisher) Close() error { return nil }

func TestEmergencyStop_Activate(t *testing.T) {
	pub := &mockPublisher{}
	logger, _ := zap.NewDevelopment()
	es := emergency.NewEmergencyStop(pub, logger)

	ctx := context.Background()
	if err := es.Activate(ctx, "test_reason"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !es.IsActive() {
		t.Error("expected emergency stop to be active")
	}
	if es.Reason() != "test_reason" {
		t.Errorf("expected reason 'test_reason', got %s", es.Reason())
	}
	if pub.emergencyStopCalled.Load() != 1 {
		t.Errorf("expected 1 emergency stop publish, got %d", pub.emergencyStopCalled.Load())
	}
	if pub.cancelAllCalled.Load() != 1 {
		t.Errorf("expected 1 cancel all publish, got %d", pub.cancelAllCalled.Load())
	}
	if pub.notificationCalled.Load() != 1 {
		t.Errorf("expected 1 notification publish, got %d", pub.notificationCalled.Load())
	}
}

func TestEmergencyStop_ActivateWithDetails(t *testing.T) {
	pub := &mockPublisher{}
	logger, _ := zap.NewDevelopment()
	es := emergency.NewEmergencyStop(pub, logger)

	ctx := context.Background()
	drawdown := decimal.NewFromFloat(0.12)
	peak := decimal.NewFromFloat(50000)
	current := decimal.NewFromFloat(44000)

	err := es.ActivateWithDetails(ctx, "drawdown_exceeded", &drawdown, &peak, &current, decimal.NewFromFloat(-150), 3, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !es.IsActive() {
		t.Error("expected emergency stop to be active")
	}
	if es.Reason() != "drawdown_exceeded" {
		t.Errorf("expected reason 'drawdown_exceeded', got %s", es.Reason())
	}
	if pub.emergencyStopCalled.Load() != 1 {
		t.Errorf("expected 1 emergency stop publish, got %d", pub.emergencyStopCalled.Load())
	}
}

func TestEmergencyStop_Idempotent(t *testing.T) {
	pub := &mockPublisher{}
	logger, _ := zap.NewDevelopment()
	es := emergency.NewEmergencyStop(pub, logger)

	ctx := context.Background()
	es.Activate(ctx, "first")
	es.Activate(ctx, "second")

	if pub.emergencyStopCalled.Load() != 1 {
		t.Errorf("expected 1 emergency stop publish (idempotent), got %d", pub.emergencyStopCalled.Load())
	}
	if es.Reason() != "first" {
		t.Errorf("expected reason to remain 'first', got %s", es.Reason())
	}
}

func TestEmergencyStop_Resume(t *testing.T) {
	pub := &mockPublisher{}
	logger, _ := zap.NewDevelopment()
	es := emergency.NewEmergencyStop(pub, logger)

	ctx := context.Background()
	es.Activate(ctx, "test_reason")

	if err := es.Resume(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if es.IsActive() {
		t.Error("expected emergency stop to be inactive after resume")
	}
	if es.Reason() != "" {
		t.Errorf("expected empty reason after resume, got %s", es.Reason())
	}
	if pub.tradingResumedCalled.Load() != 1 {
		t.Errorf("expected 1 trading resumed publish, got %d", pub.tradingResumedCalled.Load())
	}
	if pub.notificationCalled.Load() != 2 {
		t.Errorf("expected 2 notifications (activate + resume), got %d", pub.notificationCalled.Load())
	}
}

func TestEmergencyStop_ResumeByUser(t *testing.T) {
	pub := &mockPublisher{}
	logger, _ := zap.NewDevelopment()
	es := emergency.NewEmergencyStop(pub, logger)

	ctx := context.Background()
	es.Activate(ctx, "drawdown_exceeded")
	es.ResumeByUser(ctx, "admin_user")

	if es.IsActive() {
		t.Error("expected emergency stop to be inactive")
	}
}

func TestEmergencyStop_ResumeNotActive(t *testing.T) {
	pub := &mockPublisher{}
	logger, _ := zap.NewDevelopment()
	es := emergency.NewEmergencyStop(pub, logger)

	ctx := context.Background()
	err := es.Resume(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pub.tradingResumedCalled.Load() != 0 {
		t.Errorf("expected 0 trading resumed publish, got %d", pub.tradingResumedCalled.Load())
	}
}

func TestEmergencyStop_ActivatedAt(t *testing.T) {
	pub := &mockPublisher{}
	logger, _ := zap.NewDevelopment()
	es := emergency.NewEmergencyStop(pub, logger)

	if es.ActivatedAt() != nil {
		t.Error("expected nil activatedAt before activation")
	}

	ctx := context.Background()
	es.Activate(ctx, "test")

	if es.ActivatedAt() == nil {
		t.Error("expected non-nil activatedAt after activation")
	}
}

func TestEmergencyStop_Duration(t *testing.T) {
	pub := &mockPublisher{}
	logger, _ := zap.NewDevelopment()
	es := emergency.NewEmergencyStop(pub, logger)

	if es.Duration() != 0 {
		t.Error("expected 0 duration before activation")
	}

	ctx := context.Background()
	es.Activate(ctx, "test")

	time.Sleep(10 * time.Millisecond)
	if es.Duration() < 10*time.Millisecond {
		t.Error("expected positive duration after activation")
	}
}

func TestEmergencyStop_Callbacks(t *testing.T) {
	pub := &mockPublisher{}
	logger, _ := zap.NewDevelopment()
	es := emergency.NewEmergencyStop(pub, logger)

	var activateCalled, resumeCalled bool
	es.SetCallbacks(
		func() { activateCalled = true },
		func() { resumeCalled = true },
	)

	ctx := context.Background()
	es.Activate(ctx, "test")
	if !activateCalled {
		t.Error("expected onActivate callback to be called")
	}

	es.Resume(ctx)
	if !resumeCalled {
		t.Error("expected onResume callback to be called")
	}
}

func TestEmergencyStop_WithOrderCanceler(t *testing.T) {
	pub := &mockPublisher{}
	logger, _ := zap.NewDevelopment()
	es := emergency.NewEmergencyStop(pub, logger)
	oc := emergency.NewOrderCanceler(pub, 5*time.Second, logger)
	es.SetOrderCanceler(oc)

	ctx := context.Background()
	es.Activate(ctx, "test")

	if pub.cancelAllCalled.Load() != 1 {
		t.Errorf("expected 1 cancel all via order canceler, got %d", pub.cancelAllCalled.Load())
	}
}

func TestEmergencyStop_CompletesWithinOneSecond(t *testing.T) {
	pub := &mockPublisher{}
	logger, _ := zap.NewDevelopment()
	es := emergency.NewEmergencyStop(pub, logger)

	ctx := context.Background()
	start := time.Now()
	es.Activate(ctx, "test")
	elapsed := time.Since(start)

	if elapsed > 1*time.Second {
		t.Errorf("emergency stop took %v, must complete within 1 second", elapsed)
	}
}
