package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pqap/services/arb-engine/internal/ports"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type MarketPriceEvent struct {
	EventID   string                   `json:"event_id"`
	EventType string                   `json:"event_type"`
	Timestamp time.Time                `json:"timestamp"`
	Source    string                   `json:"source"`
	Payload   MarketPricePayload       `json:"payload"`
}

type MarketPricePayload struct {
	MarketID        string          `json:"id"`
	YESPrice        decimal.Decimal `json:"yes_price"`
	NOPrice         decimal.Decimal `json:"no_price"`
	Spread          decimal.Decimal `json:"spread"`
	Volume          decimal.Decimal `json:"volume_24h"`
	LiquidityDepth  decimal.Decimal `json:"liquidity_depth"`
	IsStale         bool            `json:"is_stale"`
	Timestamp       time.Time       `json:"last_updated"`
}

type NATSSubscriber struct {
	conn       *nats.Conn
	js         nats.JetStreamContext
	logger     *zap.Logger
	connected  atomic.Bool
	connStatus prometheus.Gauge
	sub        *nats.Subscription
	workCh     chan ports.MarketPriceUpdated
}

func NewNATSSubscriber(url string, logger *zap.Logger, connStatus prometheus.Gauge, workerCount int) (*NATSSubscriber, error) {
	sub := &NATSSubscriber{
		logger:     logger,
		connStatus: connStatus,
		workCh:     make(chan ports.MarketPriceUpdated, 256),
	}

	opts := []nats.Option{
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Warn("nats subscriber disconnected", zap.Error(err))
			sub.connected.Store(false)
			connStatus.Set(0)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("nats subscriber reconnected", zap.String("url", nc.ConnectedUrl()))
			sub.connected.Store(true)
			connStatus.Set(1)
		}),
	}

	conn, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	sub.conn = conn
	sub.connected.Store(true)
	connStatus.Set(1)

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}
	sub.js = js

	return sub, nil
}

func (s *NATSSubscriber) StartWorkers(ctx context.Context, handler func(ports.MarketPriceUpdated), workerCount int) {
	for i := 0; i < workerCount; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case event, ok := <-s.workCh:
					if !ok {
						return
					}
					handler(event)
				}
			}
		}()
	}
}

func (s *NATSSubscriber) Subscribe(ctx context.Context, handler func(ports.MarketPriceUpdated)) error {
	sub, err := s.js.Subscribe("pqap.market.*.price", func(msg *nats.Msg) {
		var event MarketPriceEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			s.logger.Error("failed to unmarshal market price event", zap.Error(err))
			msg.Term()
			return
		}

		if err := ValidateMarketID(event.Payload.MarketID); err != nil {
			s.logger.Warn("invalid market_id in event", zap.String("market_id", event.Payload.MarketID), zap.Error(err))
			msg.Term()
			return
		}

		if event.EventType != "" && event.EventType != "MarketPriceUpdated" {
			s.logger.Debug("skipping non-price market event", zap.String("event_type", event.EventType))
			msg.Ack()
			return
		}

		converted := ports.MarketPriceUpdated{
			MarketID:       event.Payload.MarketID,
			YESPrice:       event.Payload.YESPrice,
			NOPrice:        event.Payload.NOPrice,
			Spread:         event.Payload.Spread,
			Volume:         event.Payload.Volume,
			LiquidityDepth: event.Payload.LiquidityDepth,
			IsStale:        event.Payload.IsStale,
			Timestamp:      event.Payload.Timestamp,
		}

		select {
		case s.workCh <- converted:
			msg.Ack()
		default:
			s.logger.Warn("worker pool full, nacking message")
			msg.Nak()
		}
	}, nats.Durable("arb-engine"), nats.ManualAck())
	if err != nil {
		return fmt.Errorf("failed to subscribe to pqap.market.*.price: %w", err)
	}

	s.sub = sub

	go func() {
		<-ctx.Done()
		s.logger.Info("unsubscribing from NATS due to context cancellation")
		sub.Unsubscribe()
		close(s.workCh)
	}()

	s.logger.Info("subscribed to pqap.market.*.price")
	return nil
}

func (s *NATSSubscriber) Close() error {
	if s.sub != nil {
		s.sub.Unsubscribe()
	}
	s.conn.Close()
	s.connected.Store(false)
	s.connStatus.Set(0)
	return nil
}
