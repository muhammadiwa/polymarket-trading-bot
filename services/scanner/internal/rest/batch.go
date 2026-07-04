package rest

import (
	"context"

	"github.com/pqap/services/scanner/internal/catalog"
	"github.com/pqap/services/scanner/metrics"
	"go.uber.org/zap"
)

const defaultBatchSize = 100

type BatchFetcher struct {
	client   *Client
	maxBatch int
	logger   *zap.Logger
}

func NewBatchFetcher(client *Client, maxBatch int, logger *zap.Logger) *BatchFetcher {
	if maxBatch <= 0 {
		maxBatch = defaultBatchSize
	}
	return &BatchFetcher{
		client:   client,
		maxBatch: maxBatch,
		logger:   logger,
	}
}

func (bf *BatchFetcher) FetchAllMarkets(ctx context.Context) ([]catalog.Market, error) {
	return bf.client.FetchActiveMarkets(ctx)
}

// FetchMarketBatch fetches only the requested market IDs using the API's id filter.
// IDs are chunked into batches of maxBatch to respect API limits.
// maxBatch caps the total number of results returned across all chunks.
func (bf *BatchFetcher) FetchMarketBatch(ctx context.Context, marketIDs []string) ([]catalog.Market, error) {
	var allMarkets []catalog.Market

	for i := 0; i < len(marketIDs); i += bf.maxBatch {
		end := i + bf.maxBatch
		if end > len(marketIDs) {
			end = len(marketIDs)
		}
		chunk := marketIDs[i:end]

		markets, err := bf.client.FetchMarketsByIDs(ctx, chunk)
		if err != nil {
			// Fix #5: Return nil on failure instead of partial results
			return nil, err
		}

		metrics.RestBatchTotal.Inc()
		allMarkets = append(allMarkets, markets...)

		if len(allMarkets) >= bf.maxBatch {
			allMarkets = allMarkets[:bf.maxBatch]
			break
		}
	}

	return allMarkets, nil
}
