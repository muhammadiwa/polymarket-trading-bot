package riskmanager

import (
	"testing"

	"github.com/pqap/services/risk-manager/internal/risk"
	"go.uber.org/zap"
)

func newTestMetabolicMonitor(cpuThreshold float64, memoryThreshold uint64, goroutineThreshold int) *risk.MetabolicMonitor {
	logger, _ := zap.NewDevelopment()
	return risk.NewMetabolicMonitor(cpuThreshold, memoryThreshold, goroutineThreshold, nil, logger)
}

func TestMetabolicMonitor_CollectMetrics(t *testing.T) {
	mm := newTestMetabolicMonitor(80, 1073741824, 10000)

	m := mm.Collect()

	if m.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}

	if m.GoroutineCount <= 0 {
		t.Error("expected positive goroutine count")
	}

	if m.MemoryBytes == 0 {
		t.Error("expected non-zero memory bytes")
	}
}

func TestMetabolicMonitor_AlertWhenMemoryExceeded(t *testing.T) {
	mm := newTestMetabolicMonitor(80, 100, 10000)

	isAlert := mm.IsAlert()
	if !isAlert {
		t.Error("expected alert when memory threshold is very low")
	}
}

func TestMetabolicMonitor_AlertWhenGoroutinesExceeded(t *testing.T) {
	mm := newTestMetabolicMonitor(80, 1073741824, 1)

	isAlert := mm.IsAlert()
	if !isAlert {
		t.Error("expected alert when goroutine threshold is very low")
	}
}

func TestMetabolicMonitor_NoAlertWhenWithinThresholds(t *testing.T) {
	mm := newTestMetabolicMonitor(100, 1<<62, 1000000)

	isAlert := mm.IsAlert()
	if isAlert {
		t.Error("no alert expected when all thresholds are very high")
	}
}

func TestMetabolicMonitor_GetThresholds(t *testing.T) {
	mm := newTestMetabolicMonitor(80, 1073741824, 10000)

	cpu, mem, goroutines := mm.GetThresholds()
	if cpu != 80 {
		t.Errorf("expected cpu threshold 80, got %f", cpu)
	}
	if mem != 1073741824 {
		t.Errorf("expected memory threshold 1073741824, got %d", mem)
	}
	if goroutines != 10000 {
		t.Errorf("expected goroutine threshold 10000, got %d", goroutines)
	}
}

func TestMetabolicMonitor_MemoryMetricsReasonable(t *testing.T) {
	mm := newTestMetabolicMonitor(80, 1<<62, 1000000)

	m := mm.Collect()

	if m.MemoryBytes < 1024 {
		t.Errorf("memory bytes seems too low: %d", m.MemoryBytes)
	}
}

func TestMetabolicMonitor_GoroutineCountPositive(t *testing.T) {
	mm := newTestMetabolicMonitor(80, 1<<62, 1000000)

	m := mm.Collect()

	if m.GoroutineCount < 1 {
		t.Errorf("goroutine count should be at least 1, got %d", m.GoroutineCount)
	}
}
