package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/nats-io/nats.go"
	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type NATSPublisher struct {
	conn       *nats.Conn
	js         nats.JetStreamContext
	logger     *zap.Logger
	connected  atomic.Bool
	connStatus prometheus.Gauge
}

func NewNATSPublisher(url string, logger *zap.Logger, connStatus prometheus.Gauge) (*NATSPublisher, error) {
	np := &NATSPublisher{
		logger:     logger,
		connStatus: connStatus,
	}

	opts := []nats.Option{
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Warn("nats publisher disconnected", zap.Error(err))
			np.connected.Store(false)
			if connStatus != nil {
				connStatus.Set(0)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("nats publisher reconnected", zap.String("url", nc.ConnectedUrl()))
			np.connected.Store(true)
			if connStatus != nil {
				connStatus.Set(1)
			}
		}),
	}

	conn, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	np.conn = conn
	np.connected.Store(true)
	if connStatus != nil {
		connStatus.Set(1)
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}
	np.js = js

	streams := []struct {
		name     string
		subjects []string
	}{
		{"PQAP_ORDERS", []string{"pqap.order.>"}},
		{"PQAP_RISK", []string{"pqap.risk.>"}},
		{"PQAP_TRADES", []string{"pqap.trade.>"}},
	}

	for _, s := range streams {
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     s.name,
			Subjects: s.subjects,
		})
		if err != nil {
			logger.Warn("stream creation (may already exist)", zap.Error(err))
		}
	}

	return np, nil
}

func (p *NATSPublisher) PublishOrderPlaced(ctx context.Context, event ports.OrderPlaced) error {
	return p.publish(ctx, "pqap.order.placed", event)
}

func (p *NATSPublisher) PublishOrderFilled(ctx context.Context, event ports.OrderFilled) error {
	return p.publish(ctx, "pqap.order.filled", event)
}

func (p *NATSPublisher) PublishOrderPartialFill(ctx context.Context, event ports.OrderPartialFill) error {
	return p.publish(ctx, "pqap.order.partial_fill", event)
}

func (p *NATSPublisher) PublishOrderCancelled(ctx context.Context, event ports.OrderCancelled) error {
	return p.publish(ctx, "pqap.order.cancelled", event)
}

func (p *NATSPublisher) PublishOrderFailed(ctx context.Context, event ports.OrderFailed) error {
	return p.publish(ctx, "pqap.order.failed", event)
}

func (p *NATSPublisher) PublishRiskAlert(ctx context.Context, event ports.RiskAlert) error {
	return p.publish(ctx, "pqap.risk.alert", event)
}

func (p *NATSPublisher) PublishAtomicLegFailed(ctx context.Context, event ports.AtomicLegFailed) error {
	return p.publish(ctx, "pqap.order.atomic_leg_failed", event)
}

func (p *NATSPublisher) PublishCircuitBreakerTripped(ctx context.Context, event ports.CircuitBreakerTripped) error {
	return p.publish(ctx, "pqap.risk.circuit_breaker_tripped", event)
}

func (p *NATSPublisher) PublishCircuitBreakerResumed(ctx context.Context, event ports.CircuitBreakerResumed) error {
	return p.publish(ctx, "pqap.risk.circuit_breaker_resumed", event)
}

func (p *NATSPublisher) PublishNotificationRequest(ctx context.Context, event ports.NotificationRequest) error {
	return p.publish(ctx, "pqap.notification.send", event)
}

func (p *NATSPublisher) PublishTradeRecorded(ctx context.Context, event ports.TradeRecorded) error {
	return p.publish(ctx, "pqap.trade.recorded", event)
}

func (p *NATSPublisher) publish(ctx context.Context, subject string, event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = p.js.Publish(subject, data, nats.Context(ctx))
	if err != nil {
		p.logger.Error("failed to publish event",
			zap.String("subject", subject),
			zap.Error(err),
		)
		return err
	}

	p.logger.Debug("published event",
		zap.String("subject", subject),
	)
	return nil
}

func (p *NATSPublisher) IsConnected() bool {
	return p.connected.Load()
}

func (p *NATSPublisher) Subscribe(subject string, handler nats.MsgHandler) error {
	_, err := p.conn.Subscribe(subject, handler)
	return err
}

func (p *NATSPublisher) SubscribeJetStream(subject, durable string, handler nats.MsgHandler) error {
	_, err := p.js.Subscribe(subject, handler, nats.Durable(durable), nats.ManualAck())
	return err
}

func (p *NATSPublisher) Close() error {
	p.conn.Close()
	p.connected.Store(false)
	if p.connStatus != nil {
		p.connStatus.Set(0)
	}
	return nil
}
