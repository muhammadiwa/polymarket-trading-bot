package executor_test

import (
	"context"
	"testing"

	"github.com/pqap/services/execution-engine/internal/executor"
	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type mockRiskPort struct {
	decision *ports.RiskDecision
	err      error
}

func (m *mockRiskPort) CheckRisk(ctx context.Context, marketID, strategyID string, orderSize decimal.Decimal) (*ports.RiskDecision, error) {
	return m.decision, m.err
}

type mockRiskEventRepo struct {
	events []ports.RiskEvent
}

func (m *mockRiskEventRepo) InsertRiskEvent(ctx context.Context, event ports.RiskEvent) error {
	m.events = append(m.events, event)
	return nil
}

func TestRiskChecker_Allow(t *testing.T) {
	mock := &mockRiskPort{
		decision: &ports.RiskDecision{
			Allowed: true,
			Reason:  "",
		},
	}

	mockRepo := &mockRiskEventRepo{}
	log, _ := zap.NewDevelopment()
	checker := executor.NewRiskChecker(mock, mockRepo, log)
	decision, latencyMs, err := checker.Check(context.Background(), "market-1", "default", decimal.NewFromInt(100))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !decision.Allowed {
		t.Error("expected decision to be allowed")
	}
	if latencyMs < 0 {
		t.Errorf("latencyMs = %d, want >= 0", latencyMs)
	}
	if len(mockRepo.events) != 1 {
		t.Errorf("expected 1 risk event logged, got %d", len(mockRepo.events))
	}
}

func TestRiskChecker_DenyBudgetExhausted(t *testing.T) {
	mock := &mockRiskPort{
		decision: &ports.RiskDecision{
			Allowed: false,
			Reason:  "daily_budget_exhausted",
		},
	}

	mockRepo := &mockRiskEventRepo{}
	log, _ := zap.NewDevelopment()
	checker := executor.NewRiskChecker(mock, mockRepo, log)
	decision, _, err := checker.Check(context.Background(), "market-1", "default", decimal.NewFromInt(100))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Allowed {
		t.Error("expected decision to be denied")
	}
	if decision.Reason != "daily_budget_exhausted" {
		t.Errorf("reason = %s, want daily_budget_exhausted", decision.Reason)
	}
	if len(mockRepo.events) != 1 {
		t.Errorf("expected 1 risk event logged, got %d", len(mockRepo.events))
	}
	if mockRepo.events[0].Allowed {
		t.Error("expected risk event to show denied")
	}
}

func TestRiskChecker_DenyMarketLimitExceeded(t *testing.T) {
	mock := &mockRiskPort{
		decision: &ports.RiskDecision{
			Allowed: false,
			Reason:  "per_market_position_limit_exceeded",
		},
	}

	mockRepo := &mockRiskEventRepo{}
	log, _ := zap.NewDevelopment()
	checker := executor.NewRiskChecker(mock, mockRepo, log)
	decision, _, err := checker.Check(context.Background(), "market-1", "default", decimal.NewFromInt(100))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Allowed {
		t.Error("expected decision to be denied")
	}
	if decision.Reason != "per_market_position_limit_exceeded" {
		t.Errorf("reason = %s, want per_market_position_limit_exceeded", decision.Reason)
	}
}

func TestRiskChecker_DenyStrategyLimitExceeded(t *testing.T) {
	mock := &mockRiskPort{
		decision: &ports.RiskDecision{
			Allowed: false,
			Reason:  "per_strategy_position_limit_exceeded",
		},
	}

	mockRepo := &mockRiskEventRepo{}
	log, _ := zap.NewDevelopment()
	checker := executor.NewRiskChecker(mock, mockRepo, log)
	decision, _, err := checker.Check(context.Background(), "market-1", "strategy-1", decimal.NewFromInt(100))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Allowed {
		t.Error("expected decision to be denied")
	}
	if decision.Reason != "per_strategy_position_limit_exceeded" {
		t.Errorf("reason = %s, want per_strategy_position_limit_exceeded", decision.Reason)
	}
}

func TestRiskChecker_Error(t *testing.T) {
	mock := &mockRiskPort{
		decision: nil,
		err:      context.DeadlineExceeded,
	}

	mockRepo := &mockRiskEventRepo{}
	log, _ := zap.NewDevelopment()
	checker := executor.NewRiskChecker(mock, mockRepo, log)
	decision, _, err := checker.Check(context.Background(), "market-1", "default", decimal.NewFromInt(100))

	if err == nil {
		t.Error("expected error, got nil")
	}
	if decision.Allowed {
		t.Error("expected decision to be denied on error")
	}
	if len(mockRepo.events) != 1 {
		t.Errorf("expected 1 risk event logged on error, got %d", len(mockRepo.events))
	}
}
