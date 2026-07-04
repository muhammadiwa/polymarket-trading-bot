package websocket

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

type Subscriber struct {
	client    *Client
	marketIDs []string
	mu        sync.RWMutex
	logger    *zap.Logger
}

func NewSubscriber(client *Client, logger *zap.Logger) *Subscriber {
	return &Subscriber{
		client: client,
		logger: logger,
	}
}

func (s *Subscriber) Subscribe(ctx context.Context, marketIDs []string) error {
	s.mu.Lock()
	s.marketIDs = marketIDs
	s.mu.Unlock()

	if err := s.client.Subscribe(marketIDs); err != nil {
		return err
	}

	s.logger.Info("subscribed to markets", zap.Int("count", len(marketIDs)))
	return nil
}

func (s *Subscriber) ResubscribeAll(ctx context.Context) error {
	s.mu.RLock()
	ids := s.marketIDs
	s.mu.RUnlock()

	if len(ids) == 0 {
		return nil
	}

	s.logger.Info("resubscribing to markets after reconnect", zap.Int("count", len(ids)))
	return s.client.Subscribe(ids)
}

func (s *Subscriber) AddMarket(marketID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range s.marketIDs {
		if id == marketID {
			return
		}
	}
	s.marketIDs = append(s.marketIDs, marketID)
}
