package executor

import (
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type PartialFillRecord struct {
	PairID        string          `json:"pair_id"`
	Leg           string          `json:"leg"`
	FilledQty     decimal.Decimal `json:"filled_qty"`
	RemainingQty  decimal.Decimal `json:"remaining_qty"`
	FillPrice     decimal.Decimal `json:"fill_price"`
	OrderID       string          `json:"order_id"`
	ClientOrderID string          `json:"client_order_id"`
	MarketID      string          `json:"market_id"`
	StrategyID    string          `json:"strategy_id"`
	CreatedAt     time.Time       `json:"created_at"`
}

type PartialTracker struct {
	mu        sync.RWMutex
	records   map[string][]PartialFillRecord
	maxTotal  int
	totalSize int
	logger    *zap.Logger
}

func NewPartialTracker(logger *zap.Logger) *PartialTracker {
	return &PartialTracker{
		records:  make(map[string][]PartialFillRecord),
		maxTotal: 10000,
		logger:   logger,
	}
}

func (pt *PartialTracker) RecordPartialFill(record PartialFillRecord) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.totalSize >= pt.maxTotal {
		pt.evictOldest()
	}

	if _, exists := pt.records[record.PairID]; !exists {
		pt.records[record.PairID] = []PartialFillRecord{}
	}

	record.CreatedAt = time.Now().UTC()
	pt.records[record.PairID] = append(pt.records[record.PairID], record)
	pt.totalSize++

	pt.logger.Info("partial fill recorded",
		zap.String("pair_id", record.PairID),
		zap.String("leg", record.Leg),
		zap.String("filled_qty", record.FilledQty.String()),
		zap.String("remaining_qty", record.RemainingQty.String()),
		zap.String("fill_price", record.FillPrice.String()),
	)
}

func (pt *PartialTracker) evictOldest() {
	var oldestPairID string
	var oldestTime time.Time
	for pairID, records := range pt.records {
		if len(records) > 0 && (oldestPairID == "" || records[0].CreatedAt.Before(oldestTime)) {
			oldestPairID = pairID
			oldestTime = records[0].CreatedAt
		}
	}
	if oldestPairID != "" {
		pt.totalSize -= len(pt.records[oldestPairID])
		delete(pt.records, oldestPairID)
		pt.logger.Warn("evicted oldest partial fill records",
			zap.String("pair_id", oldestPairID),
		)
	}
}

func (pt *PartialTracker) GetPartialFills(pairID string) []PartialFillRecord {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	return pt.records[pairID]
}

func (pt *PartialTracker) Reconcile(pairID string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if records, exists := pt.records[pairID]; exists {
		pt.logger.Info("reconciling partial fills",
			zap.String("pair_id", pairID),
			zap.Int("fill_count", len(records)),
		)
		pt.totalSize -= len(records)
		delete(pt.records, pairID)
	}
}

func (pt *PartialTracker) HasPartialFills(pairID string) bool {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	records, exists := pt.records[pairID]
	return exists && len(records) > 0
}
