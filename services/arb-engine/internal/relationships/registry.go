package relationships

import (
	"context"
	"sync"
	"time"

	"github.com/pqap/services/arb-engine/internal/ports"
	"github.com/pqap/services/arb-engine/metrics"
	"go.uber.org/zap"
)

type Registry struct {
	repo        ports.RelationshipRepository
	mu          sync.RWMutex
	byMarketID  map[string][]ports.MarketRelationship // marketID → related relationships
	logger      *zap.Logger
	refreshInterval time.Duration
}

func NewRegistry(repo ports.RelationshipRepository, logger *zap.Logger) *Registry {
	return &Registry{
		repo:           repo,
		byMarketID:     make(map[string][]ports.MarketRelationship),
		logger:         logger,
		refreshInterval: 1 * time.Hour,
	}
}

// Refresh loads all relationships from the database and rebuilds the in-memory index.
func (r *Registry) Refresh(ctx context.Context) error {
	rels, err := r.repo.GetRelationships(ctx)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.byMarketID = make(map[string][]ports.MarketRelationship)
	for _, rel := range rels {
		r.byMarketID[rel.MarketAID] = append(r.byMarketID[rel.MarketAID], rel)
		// Also index in reverse direction
		reverse := ports.MarketRelationship{
			ID:               rel.ID,
			MarketAID:        rel.MarketBID,
			MarketBID:        rel.MarketAID,
			RelationshipType: rel.RelationshipType,
			Confidence:       rel.Confidence,
		}
		r.byMarketID[rel.MarketBID] = append(r.byMarketID[rel.MarketBID], reverse)
	}

	metrics.RelationshipCount.Set(float64(len(rels)))
	r.logger.Info("relationship registry refreshed", zap.Int("count", len(rels)))
	return nil
}

// GetRelatedMarkets returns all relationships where marketID is MarketA or MarketB.
func (r *Registry) GetRelatedMarkets(ctx context.Context, marketID string) ([]ports.MarketRelationship, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rels, ok := r.byMarketID[marketID]
	if !ok {
		return nil, nil
	}

	result := make([]ports.MarketRelationship, len(rels))
	copy(result, rels)
	return result, nil
}

// StartRefreshLoop starts a background goroutine that refreshes the registry periodically.
// Returns a channel that is closed when the goroutine exits.
func (r *Registry) StartRefreshLoop(ctx context.Context) chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(r.refreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := r.Refresh(ctx); err != nil {
					r.logger.Error("failed to refresh relationship registry", zap.Error(err))
				}
			}
		}
	}()
	return done
}
