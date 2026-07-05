package detector

import (
	"context"
	"sync"
	"time"

	"github.com/pqap/services/arb-engine/metrics"
	"go.uber.org/zap"
)

type NearResolutionDetector struct {
	mu         sync.RWMutex
	windowMin  int
	threshold  float64
	resolution map[string]time.Time
	logger     *zap.Logger
}

func NewNearResolutionDetector(windowMin int, threshold float64, logger *zap.Logger) *NearResolutionDetector {
	return &NearResolutionDetector{
		windowMin:  windowMin,
		threshold:  threshold,
		resolution: make(map[string]time.Time),
		logger:     logger,
	}
}

func (d *NearResolutionDetector) SetResolutionTime(marketID string, resolutionTime time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.resolution[marketID] = resolutionTime
}

func (d *NearResolutionDetector) RemoveMarket(marketID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.resolution, marketID)
}

func (d *NearResolutionDetector) CleanupExpired() {
	d.mu.Lock()
	defer d.mu.Unlock()
	now := time.Now()
	for id, resTime := range d.resolution {
		if now.After(resTime) {
			delete(d.resolution, id)
		}
	}
}

func (d *NearResolutionDetector) Check(marketID string) (bool, float64) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	resTime, ok := d.resolution[marketID]
	if !ok {
		return false, 1.0
	}

	// #5: Past resolution time — not near resolution
	if time.Until(resTime) < 0 {
		return false, 1.0
	}

	window := time.Duration(d.windowMin) * time.Minute
	if time.Until(resTime) <= window {
		metrics.NearResolutionTotal.Inc()
		d.logger.Debug("market near resolution",
			zap.String("market_id", marketID),
			zap.Time("resolution_time", resTime),
			zap.Float64("confidence_factor", d.threshold),
		)
		return true, d.threshold
	}

	return false, 1.0
}

// #7: StartCleanupLoop periodically removes expired entries
func (d *NearResolutionDetector) StartCleanupLoop(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				d.CleanupExpired()
			}
		}
	}()
}
