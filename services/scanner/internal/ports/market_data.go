package ports

import (
	"context"

	"github.com/pqap/services/scanner/internal/catalog"
)

type MarketDataPort interface {
	Connect(ctx context.Context) error
	Subscribe(ctx context.Context, marketIDs []string) error
	ReadMessage(ctx context.Context) (*catalog.Market, error)
	Close() error
}
