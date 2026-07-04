package ports

import (
	"context"

	"github.com/pqap/services/scanner/internal/catalog"
)

type EventPort interface {
	PublishPriceUpdate(ctx context.Context, market catalog.Market) error
	PublishMarketDiscovered(ctx context.Context, market catalog.Market) error
	PublishMarketStale(ctx context.Context, market catalog.Market) error
	Close() error
}
