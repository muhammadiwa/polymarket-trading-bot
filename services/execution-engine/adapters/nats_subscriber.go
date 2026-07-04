package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type NATSSubscriber struct {
	conn       *nats.Conn
	js         nats.JetStreamContext
	logger     *zap.Logger
	connected  atomic.Bool
	connStatus prometheus.Gauge
	sub        *nats.Subscription
}

func NewNATSSubscriber(url string, logger *zap.Logger, connStatus prometheus.Gauge) (*NATSSubscriber, error) {
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

func (s *NATSSubscriber) Subscribe(ctx context.Context, handler func(ports.OpportunityDetected)) error {
	msgCh := make(chan *nats.Msg, 256)

	// TODO(#16): No dead letter queue configured. Poison messages that fail
	// processing will be termd and lost. Consider configuring a DLQ stream
	// for forensic analysis of unprocessable messages.

	sub, err := s.js.Subscribe("pqap.opportunity.detected", func(msg *nats.Msg) {
		select {
		case msgCh <- msg:
		default:
			s.logger.Warn("message channel full, terming message")
			msg.Term()
		}
	}, nats.Durable("execution-engine"), nats.ManualAck())
	if err != nil {
		return fmt.Errorf("failed to subscribe to pqap.opportunity.detected: %w", err)
	}

	s.sub = sub

	for i := 0; i < 4; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-msgCh:
					if !ok {
						return
					}
					var event ports.OpportunityDetected
					if err := json.Unmarshal(msg.Data, &event); err != nil {
						s.logger.Error("failed to unmarshal opportunity event", zap.Error(err))
						msg.Term()
						continue
					}

					handler(event)
					msg.Ack()
				}
			}
		}()
	}

	go func() {
		<-ctx.Done()
		s.logger.Info("unsubscribing from NATS due to context cancellation")
		sub.Unsubscribe()
		close(msgCh)
	}()

	s.logger.Info("subscribed to pqap.opportunity.detected")
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
