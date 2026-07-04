package riskmanager

import (
	"context"
	"testing"
	"time"

	"github.com/pqap/services/risk-manager/internal/emergency"
	"go.uber.org/zap"
)

func TestOrderCanceler_CancelAll(t *testing.T) {
	pub := &mockPublisher{}
	logger, _ := zap.NewDevelopment()
	oc := emergency.NewOrderCanceler(pub, 5*time.Second, logger)

	ctx := context.Background()
	result, err := oc.CancelAll(ctx, "drawdown_exceeded")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected successful cancellation")
	}
	if pub.cancelAllCalled.Load() != 1 {
		t.Errorf("expected 1 cancel all publish, got %d", pub.cancelAllCalled.Load())
	}
	if result.LatencyMs < 0 {
		t.Error("expected non-negative latency")
	}
}

func TestOrderCanceler_CancelAllWithReason(t *testing.T) {
	pub := &mockPublisher{}
	logger, _ := zap.NewDevelopment()
	oc := emergency.NewOrderCanceler(pub, 5*time.Second, logger)

	ctx := context.Background()
	result, err := oc.CancelAll(ctx, "api_death_spiral")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected successful cancellation")
	}
	if result.RequestedAt.IsZero() {
		t.Error("expected non-zero requestedAt")
	}
	if result.CompletedAt.IsZero() {
		t.Error("expected non-zero completedAt")
	}
}

func TestOrderCanceler_Timeout(t *testing.T) {
	pub := &mockPublisher{}
	logger, _ := zap.NewDevelopment()
	oc := emergency.NewOrderCanceler(pub, 1*time.Millisecond, logger)

	ctx := context.Background()
	result, err := oc.CancelAll(ctx, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success even with short timeout (publish is sync)")
	}
}
