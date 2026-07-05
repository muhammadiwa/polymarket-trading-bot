package detector

import (
	"time"

	"github.com/pqap/services/arb-engine/metrics"
	"go.uber.org/zap"
)

type NearResolutionDetector struct {
	windowMin  int
	threshold  float64
	resolution map[string]time.Time // marketID → expected resolution time
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

// SetResolutionTime sets the expected resolution time for a market.
func (d *NearResolutionDetector) SetResolutionTime(marketID string, resolutionTime time.Time) {
	d.resolution[marketID] = resolutionTime
}

// RemoveMarket removes a market from tracking.
func (d *NearResolutionDetector) RemoveMarket(marketID string) {
	delete(d.resolution, marketID)
}

// Check returns true if the market is near resolution, and the confidence factor to apply.
// confidenceFactor = threshold (e.g., 0.5) when near resolution, 1.0 otherwise.
func (d *NearResolutionDetector) Check(marketID string) (bool, float64) {
	resTime, ok := d.resolution[marketID]
	if !ok {
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
