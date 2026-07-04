package catalog

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type StaleDetector struct {
	catalog   *Catalog
	threshold time.Duration
	interval  time.Duration
	logger    *zap.Logger
	onStale   func(market Market)
}

func NewStaleDetector(catalog *Catalog, threshold, interval time.Duration, logger *zap.Logger, onStale func(market Market)) *StaleDetector {
	return &StaleDetector{
		catalog:   catalog,
		threshold: threshold,
		interval:  interval,
		logger:    logger,
		onStale:   onStale,
	}
}

func (sd *StaleDetector) Run(ctx context.Context) {
	ticker := time.NewTicker(sd.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sd.check() // return value intentionally unused in Run loop
		}
	}
}

// Check runs a single staleness check and returns the number of stale markets.
func (sd *StaleDetector) Check() int {
	return sd.check()
}

func (sd *StaleDetector) check() int {
	markets := sd.catalog.List()
	now := time.Now()
	staleCount := 0

	for _, m := range markets {
		if now.Sub(m.LastUpdated) > sd.threshold {
			if sd.catalog.MarkStale(m.ID) {
				sd.logger.Warn("market_stale",
					zap.String("service", "scanner"),
					zap.String("market_id", m.ID),
					zap.String("last_update", m.LastUpdated.Format("2006-01-02T15:04:05.000Z")),
				)
				if sd.onStale != nil {
					sd.onStale(m)
				}
			}
			staleCount++
		}
	}
	return staleCount
}
