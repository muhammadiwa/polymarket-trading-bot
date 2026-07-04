package risk

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/pqap/services/risk-manager/metrics"
	"go.uber.org/zap"
)

type MetabolicRateMetrics struct {
	CPUPercent     float64   `json:"cpu_percent"`
	MemoryBytes    uint64    `json:"memory_bytes"`
	GoroutineCount int       `json:"goroutine_count"`
	Timestamp      time.Time `json:"timestamp"`
	IsAlert        bool      `json:"is_alert"`
}

type MetabolicMonitor struct {
	mu              sync.RWMutex
	cpuThreshold    float64
	memoryThreshold uint64
	goroutineThreshold int
	publisher       ports.EventPublisher
	logger          *zap.Logger
	cancel          context.CancelFunc
	cpuSampler      *cpuSampler
}

func NewMetabolicMonitor(cpuThreshold float64, memoryThreshold uint64, goroutineThreshold int, publisher ports.EventPublisher, logger *zap.Logger) *MetabolicMonitor {
	return &MetabolicMonitor{
		cpuThreshold:       cpuThreshold,
		memoryThreshold:    memoryThreshold,
		goroutineThreshold: goroutineThreshold,
		publisher:          publisher,
		logger:             logger,
		cpuSampler:         newCPUSampler(),
	}
}

func (mm *MetabolicMonitor) Start(ctx context.Context) {
	mm.mu.Lock()
	ctx, mm.cancel = context.WithCancel(ctx)
	mm.mu.Unlock()
	go mm.monitorLoop(ctx)
}

func (mm *MetabolicMonitor) Stop() {
	mm.mu.Lock()
	cancel := mm.cancel
	mm.cancel = nil
	mm.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (mm *MetabolicMonitor) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mm.collectAndPublish()
		}
	}
}

func (mm *MetabolicMonitor) collectAndPublish() {
	m := mm.Collect()
	metrics.MetabolicCPUPercent.Set(m.CPUPercent)
	metrics.MetabolicMemoryBytes.Set(float64(m.MemoryBytes))
	metrics.MetabolicGoroutines.Set(float64(m.GoroutineCount))

	if m.IsAlert {
		metrics.MetabolicAlertsTotal.Inc()
		mm.publishAlert(m)
	}
}

func (mm *MetabolicMonitor) Collect() MetabolicRateMetrics {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	cpuPercent := mm.cpuSampler.sample()

	m := MetabolicRateMetrics{
		CPUPercent:     cpuPercent,
		MemoryBytes:    memStats.Sys,
		GoroutineCount: runtime.NumGoroutine(),
		Timestamp:      time.Now().UTC(),
	}

	mm.mu.RLock()
	cpuThr := mm.cpuThreshold
	memThr := mm.memoryThreshold
	gorThr := mm.goroutineThreshold
	mm.mu.RUnlock()

	m.IsAlert = m.CPUPercent > cpuThr || m.MemoryBytes > memThr || m.GoroutineCount > gorThr

	mm.logger.Debug("metabolic metrics collected",
		zap.Float64("cpu_percent", m.CPUPercent),
		zap.Uint64("memory_bytes", m.MemoryBytes),
		zap.Int("goroutine_count", m.GoroutineCount),
		zap.Bool("is_alert", m.IsAlert),
	)

	return m
}

func (mm *MetabolicMonitor) publishAlert(m MetabolicRateMetrics) {
	if mm.publisher == nil {
		return
	}

	event := ports.MetabolicRateAlert{
		EventID:   uuid.New().String(),
		EventType: "MetabolicRateAlert",
		Timestamp: time.Now().UTC(),
		Source:    "risk-manager",
		Payload: ports.MetabolicRatePayload{
			CPUPercent:     m.CPUPercent,
			MemoryBytes:    m.MemoryBytes,
			GoroutineCount: m.GoroutineCount,
			Timestamp:      m.Timestamp,
			IsAlert:        m.IsAlert,
		},
	}

	ctx, cancel := timeoutContext(5 * time.Second)
	defer cancel()
	if err := mm.publisher.PublishMetabolicRateAlert(ctx, event); err != nil {
		mm.logger.Error("failed to publish metabolic rate alert", zap.Error(err))
	}
}

func (mm *MetabolicMonitor) IsAlert() bool {
	m := mm.Collect()
	return m.IsAlert
}

func (mm *MetabolicMonitor) GetThresholds() (cpu float64, memory uint64, goroutines int) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.cpuThreshold, mm.memoryThreshold, mm.goroutineThreshold
}
