package websocket

import (
	"context"
	"time"

	"github.com/pqap/services/scanner/internal/catalog"
	"github.com/pqap/services/scanner/internal/rest"
	"github.com/pqap/services/scanner/metrics"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const reconciliationBatchSize = 100

type Reconciler struct {
	catalog     *catalog.Catalog
	batchClient *rest.BatchFetcher
	logger      *zap.Logger
	tickThreshold decimal.Decimal
}

func NewReconciler(catalog *catalog.Catalog, restClient *rest.Client, logger *zap.Logger) *Reconciler {
	return &Reconciler{
		catalog:       catalog,
		batchClient:   rest.NewBatchFetcher(restClient, reconciliationBatchSize, logger),
		logger:        logger,
		tickThreshold: decimal.NewFromFloat(0.01),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	start := time.Now()
	r.logger.Info("reconciliation_started",
		zap.String("service", "scanner"),
	)

	markets := r.catalog.List()
	if len(markets) == 0 {
		r.logger.Info("reconciliation_complete",
			zap.String("service", "scanner"),
			zap.Int("markets_checked", 0),
			zap.Int("discrepancies", 0),
			zap.Int64("duration_ms", time.Since(start).Milliseconds()),
		)
		metrics.ReconciliationTotal.Inc()
		return nil
	}

	marketIDs := make([]string, len(markets))
	for i, m := range markets {
		marketIDs[i] = m.ID
	}

	snapshots, err := r.batchClient.FetchMarketBatch(ctx, marketIDs)
	if err != nil {
		r.logger.Error("reconciliation_fetch_failed",
			zap.String("service", "scanner"),
			zap.Error(err),
		)
		return err
	}

	discrepancies := 0
	for _, snapshot := range snapshots {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Fix #7: Capture stale state before Update clears it
		existing, exists := r.catalog.Get(snapshot.ID)
		wasStale := exists && existing.IsStale

		if exists {
			yesDiff := existing.YESPrice.Sub(snapshot.YESPrice).Abs()
			noDiff := existing.NOPrice.Sub(snapshot.NOPrice).Abs()

			if yesDiff.GreaterThan(r.tickThreshold) || noDiff.GreaterThan(r.tickThreshold) {
				discrepancies++
				metrics.PriceDiscrepanciesTotal.Inc()
				r.logger.Warn("price_discrepancy",
					zap.String("service", "scanner"),
					zap.String("market_id", snapshot.ID),
					zap.String("internal_yes", existing.YESPrice.String()),
					zap.String("snapshot_yes", snapshot.YESPrice.String()),
					zap.String("internal_no", existing.NOPrice.String()),
					zap.String("snapshot_no", snapshot.NOPrice.String()),
				)
			}
		}

		// Fix #4: Update already sets IsStale=false; ClearStale was redundant
		r.catalog.Update(snapshot)
		_ = wasStale
	}

	metrics.ReconciliationTotal.Inc()
	duration := time.Since(start)
	r.logger.Info("reconciliation_complete",
		zap.String("service", "scanner"),
		zap.Int("markets_checked", len(snapshots)),
		zap.Int("discrepancies", discrepancies),
		zap.Int64("duration_ms", duration.Milliseconds()),
	)

	return nil
}
