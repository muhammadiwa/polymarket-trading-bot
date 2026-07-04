package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
	"sync/atomic"

	"github.com/nats-io/nats.go"
	"github.com/pqap/services/position-manager/internal/ports"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type NATSSubscriber struct {
	conn       *nats.Conn
	js         nats.JetStreamContext
	logger     *zap.Logger
	connected  atomic.Bool
	connStatus prometheus.Gauge
	subs       []*nats.Subscription
	wg         sync.WaitGroup
}

func NewNATSSubscriber(url string, logger *zap.Logger, connStatus prometheus.Gauge) (*NATSSubscriber, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	ns := &NATSSubscriber{
		logger:     logger,
		connStatus: connStatus,
	}

	opts := []nats.Option{
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Warn("nats subscriber disconnected", zap.Error(err))
			ns.connected.Store(false)
			connStatus.Set(0)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("nats subscriber reconnected", zap.String("url", nc.ConnectedUrl()))
			ns.connected.Store(true)
			connStatus.Set(1)
		}),
	}

	conn, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	ns.conn = conn
	ns.connected.Store(true)
	connStatus.Set(1)

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}
	ns.js = js

	return ns, nil
}

func (s *NATSSubscriber) SubscribeOrderFilled(ctx context.Context, handler func(ports.OrderFilled)) error {
	msgCh := make(chan *nats.Msg, 256)

	sub, err := s.js.Subscribe("pqap.order.filled", func(msg *nats.Msg) {
		select {
		case msgCh <- msg:
		default:
			s.logger.Warn("order filled channel full, terming message")
			msg.Term()
		}
	}, nats.Durable("position-manager-order-filled"), nats.ManualAck())
	if err != nil {
		return fmt.Errorf("failed to subscribe to pqap.order.filled: %w", err)
	}
	s.subs = append(s.subs, sub)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgCh:
				if !ok {
					return
				}
				var event ports.OrderFilled
				if err := json.Unmarshal(msg.Data, &event); err != nil {
					s.logger.Error("failed to unmarshal order filled event", zap.Error(err))
					msg.Term()
					continue
				}
				handler(event)
				msg.Ack()
			}
		}
	}()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		<-ctx.Done()
		sub.Unsubscribe()
		close(msgCh)
	}()

	s.logger.Info("subscribed to pqap.order.filled")
	return nil
}

var priceSubjectRegex = regexp.MustCompile(`^pqap\.market\.(.+)\.price$`)

func (s *NATSSubscriber) SubscribeMarketPriceUpdated(ctx context.Context, handler func(marketID string, event ports.MarketPriceUpdated)) error {
	msgCh := make(chan *nats.Msg, 1024)

	sub, err := s.js.Subscribe("pqap.market.*.price", func(msg *nats.Msg) {
		select {
		case msgCh <- msg:
		default:
			s.logger.Warn("price update channel full, terming message")
			msg.Term()
		}
	}, nats.Durable("position-manager-price-update"), nats.ManualAck())
	if err != nil {
		return fmt.Errorf("failed to subscribe to pqap.market.*.price: %w", err)
	}
	s.subs = append(s.subs, sub)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgCh:
				if !ok {
					return
				}
				matches := priceSubjectRegex.FindStringSubmatch(msg.Subject)
				if len(matches) < 2 {
					s.logger.Warn("unexpected price subject", zap.String("subject", msg.Subject))
					msg.Ack()
					continue
				}
				marketID := matches[1]

				var event ports.MarketPriceUpdated
				if err := json.Unmarshal(msg.Data, &event); err != nil {
					s.logger.Error("failed to unmarshal price event", zap.Error(err))
					msg.Term()
					continue
				}
				handler(marketID, event)
				msg.Ack()
			}
		}
	}()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		<-ctx.Done()
		sub.Unsubscribe()
		close(msgCh)
	}()

	s.logger.Info("subscribed to pqap.market.*.price")
	return nil
}

func (s *NATSSubscriber) SubscribeMarketResolved(ctx context.Context, handler func(ports.MarketResolved)) error {
	// TODO(#7): No service currently publishes MarketResolved events.
	// This subscription is a placeholder for future market resolution detection.
	// When a market resolves, positions should be settled automatically.
	msgCh := make(chan *nats.Msg, 64)

	sub, err := s.js.Subscribe("pqap.market.resolved", func(msg *nats.Msg) {
		select {
		case msgCh <- msg:
		default:
			s.logger.Warn("market resolved channel full, terming message")
			msg.Term()
		}
	}, nats.Durable("position-manager-market-resolved"), nats.ManualAck())
	if err != nil {
		return fmt.Errorf("failed to subscribe to pqap.market.resolved: %w", err)
	}
	s.subs = append(s.subs, sub)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgCh:
				if !ok {
					return
				}
				var event ports.MarketResolved
				if err := json.Unmarshal(msg.Data, &event); err != nil {
					s.logger.Error("failed to unmarshal market resolved event", zap.Error(err))
					msg.Term()
					continue
				}
				handler(event)
				msg.Ack()
			}
		}
	}()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		<-ctx.Done()
		sub.Unsubscribe()
		close(msgCh)
	}()

	s.logger.Info("subscribed to pqap.market.resolved")
	return nil
}

func (s *NATSSubscriber) Close() error {
	for _, sub := range s.subs {
		sub.Unsubscribe()
	}
	s.conn.Close()
	s.connected.Store(false)
	s.connStatus.Set(0)
	s.wg.Wait()
	return nil
}
