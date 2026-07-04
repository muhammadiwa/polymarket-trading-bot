package riskmanager

import (
	"context"
	"testing"
	"time"

	"github.com/pqap/services/risk-manager/internal/pitboss"
	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type mockRepo struct {
	insertCalled bool
}

func (m *mockRepo) InsertRiskEvent(ctx context.Context, event ports.RiskDecision) error {
	m.insertCalled = true
	return nil
}

func (m *mockRepo) GetTodayDecisions(ctx context.Context) ([]ports.RiskDecision, error) {
	return nil, nil
}

func (m *mockRepo) GetDailyLoss(ctx context.Context) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

func (m *mockRepo) GetPositionExposures(ctx context.Context) (map[string]decimal.Decimal, map[string]decimal.Decimal, error) {
	return nil, nil, nil
}

func (m *mockRepo) Close() error { return nil }

func newTestLogger() (*zap.Logger, *pitboss.Logger, *mockRepo) {
	zapLogger, _ := zap.NewDevelopment()
	repo := &mockRepo{}
	riskLogger := pitboss.NewLogger(repo, zapLogger)
	return zapLogger, riskLogger, repo
}

func TestLogger_LogDecision(t *testing.T) {
	_, riskLogger, repo := newTestLogger()

	decision := ports.RiskDecision{
		EventID:              "test-id",
		Timestamp:            time.Now().UTC(),
		Decision:             "ALLOW",
		Reason:               "approved",
		TradeSize:            decimal.NewFromFloat(100),
		DailyBudgetRemaining: decimal.NewFromFloat(9800),
		Capital:              decimal.NewFromFloat(10000),
	}

	ctx := context.Background()
	err := riskLogger.LogDecision(ctx, decision)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !repo.insertCalled {
		t.Error("expected InsertRiskEvent to be called")
	}
}

func TestLogger_LogDecisionAsync(t *testing.T) {
	_, riskLogger, _ := newTestLogger()

	decision := ports.RiskDecision{
		EventID:              "test-id-async",
		Timestamp:            time.Now().UTC(),
		Decision:             "DENY",
		Reason:               "daily_limit",
		TradeSize:            decimal.NewFromFloat(100),
		DailyBudgetRemaining: decimal.Zero,
		Capital:              decimal.NewFromFloat(10000),
	}

	riskLogger.LogDecisionAsync(decision)
	time.Sleep(100 * time.Millisecond)
}

func TestLogger_LogEmergencyEvent(t *testing.T) {
	_, riskLogger, repo := newTestLogger()

	ctx := context.Background()
	err := riskLogger.LogEmergencyEvent(ctx, "drawdown_exceeded", map[string]interface{}{
		"drawdown": "0.1234",
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !repo.insertCalled {
		t.Error("expected InsertRiskEvent to be called")
	}
}
