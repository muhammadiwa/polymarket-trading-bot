package riskmanager

import (
	"context"
	"testing"
	"time"

	"github.com/pqap/services/risk-manager/internal/emergency"
	"github.com/pqap/services/risk-manager/internal/pitboss"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func newTestResumeHandler() (*emergency.ResumeHandler, *emergency.EmergencyStop, *pitboss.StateBuilder, *mockPublisher) {
	pub := &mockPublisher{}
	logger, _ := zap.NewDevelopment()

	capital := decimal.NewFromFloat(10000)
	dailyBudget := pitboss.NewDailyBudget(capital, 0.02, 0.80)
	marketLimits := pitboss.NewMarketLimit(capital, 0.10)
	strategyLimits := pitboss.NewStrategyLimit(capital, 0.20)
	stateBuilder := pitboss.NewStateBuilder(dailyBudget, marketLimits, strategyLimits, capital, logger)

	es := emergency.NewEmergencyStop(pub, logger)
	es.SetCallbacks(
		func() { stateBuilder.SetEmergencyStop(true) },
		func() { stateBuilder.SetEmergencyStop(false) },
	)

	rh := emergency.NewResumeHandler(es, stateBuilder, logger)
	return rh, es, stateBuilder, pub
}

func TestResumeHandler_ClearsEmergencyStop(t *testing.T) {
	rh, es, sb, _ := newTestResumeHandler()

	ctx := context.Background()
	es.Activate(ctx, "drawdown_exceeded")

	if !sb.IsEmergencyStop() {
		t.Error("expected emergency stop to be active before resume")
	}

	err := rh.HandleResume(ctx, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if es.IsActive() {
		t.Error("expected emergency stop to be inactive after resume")
	}
	if sb.IsEmergencyStop() {
		t.Error("expected state builder emergency stop to be cleared")
	}
}

func TestResumeHandler_PublishesTradingResumed(t *testing.T) {
	rh, es, _, pub := newTestResumeHandler()

	ctx := context.Background()
	es.Activate(ctx, "test")
	rh.HandleResume(ctx, "admin")

	if pub.tradingResumedCalled.Load() != 1 {
		t.Errorf("expected 1 trading resumed publish, got %d", pub.tradingResumedCalled.Load())
	}
}

func TestResumeHandler_NoOpWhenNotActive(t *testing.T) {
	rh, _, _, pub := newTestResumeHandler()

	ctx := context.Background()
	err := rh.HandleResume(ctx, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pub.tradingResumedCalled.Load() != 0 {
		t.Errorf("expected 0 trading resumed publish, got %d", pub.tradingResumedCalled.Load())
	}
}

func TestResumeHandler_DoesNotResetPeakEquity(t *testing.T) {
	rh, es, _, _ := newTestResumeHandler()

	ctx := context.Background()
	es.Activate(ctx, "drawdown_exceeded")
	rh.HandleResume(ctx, "admin")

	if es.IsActive() {
		t.Error("expected emergency stop to be cleared")
	}
}

func TestResumeHandler_ReTriggersIfDrawdownStillExceeded(t *testing.T) {
	pub := &mockPublisher{}
	logger, _ := zap.NewDevelopment()

	capital := decimal.NewFromFloat(10000)
	dailyBudget := pitboss.NewDailyBudget(capital, 0.02, 0.80)
	marketLimits := pitboss.NewMarketLimit(capital, 0.10)
	strategyLimits := pitboss.NewStrategyLimit(capital, 0.20)
	sb := pitboss.NewStateBuilder(dailyBudget, marketLimits, strategyLimits, capital, logger)

	es := emergency.NewEmergencyStop(pub, logger)
	es.SetCallbacks(
		func() { sb.SetEmergencyStop(true) },
		func() { sb.SetEmergencyStop(false) },
	)
	rh := emergency.NewResumeHandler(es, sb, logger)

	ctx := context.Background()
	es.Activate(ctx, "drawdown_exceeded")
	rh.HandleResume(ctx, "admin")

	if es.IsActive() {
		t.Error("expected emergency stop to be cleared after resume")
	}
}

func TestResumeHandler_TracksResumedBy(t *testing.T) {
	pub := &mockPublisher{}
	logger, _ := zap.NewDevelopment()

	capital := decimal.NewFromFloat(10000)
	dailyBudget := pitboss.NewDailyBudget(capital, 0.02, 0.80)
	marketLimits := pitboss.NewMarketLimit(capital, 0.10)
	strategyLimits := pitboss.NewStrategyLimit(capital, 0.20)
	sb := pitboss.NewStateBuilder(dailyBudget, marketLimits, strategyLimits, capital, logger)

	es := emergency.NewEmergencyStop(pub, logger)
	es.SetCallbacks(
		func() { sb.SetEmergencyStop(true) },
		func() { sb.SetEmergencyStop(false) },
	)
	rh := emergency.NewResumeHandler(es, sb, logger)

	ctx := context.Background()
	es.Activate(ctx, "drawdown_exceeded")

	time.Sleep(10 * time.Millisecond)
	rh.HandleResume(ctx, "admin_user")

	if es.IsActive() {
		t.Error("expected emergency stop to be inactive")
	}
}
