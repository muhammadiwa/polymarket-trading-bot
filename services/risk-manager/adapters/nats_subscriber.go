package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/nats-io/nats.go"
	"github.com/pqap/services/risk-manager/internal/ports"
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
	subsMu     sync.Mutex // #22: protect subs slice
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
			if connStatus != nil {
				connStatus.Set(0)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("nats subscriber reconnected", zap.String("url", nc.ConnectedUrl()))
			ns.connected.Store(true)
			if connStatus != nil {
				connStatus.Set(1)
			}
		}),
	}

	conn, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	ns.conn = conn
	ns.connected.Store(true)
	if connStatus != nil {
		connStatus.Set(1)
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}
	ns.js = js

	return ns, nil
}

func (s *NATSSubscriber) SubscribeOrderFilled(ctx context.Context, handler func(ports.OrderFilled)) error {
	return s.subscribe(ctx, "pqap.order.filled", "risk-manager-order-filled", func(data []byte) error {
		var event ports.OrderFilled
		if err := json.Unmarshal(data, &event); err != nil {
			s.logger.Error("failed to unmarshal order filled event", zap.Error(err))
			return err
		}
		handler(event)
		return nil
	})
}

func (s *NATSSubscriber) SubscribePositionOpened(ctx context.Context, handler func(ports.PositionOpened)) error {
	return s.subscribe(ctx, "pqap.position.opened", "risk-manager-position-opened", func(data []byte) error {
		var event ports.PositionOpened
		if err := json.Unmarshal(data, &event); err != nil {
			s.logger.Error("failed to unmarshal position opened event", zap.Error(err))
			return err
		}
		handler(event)
		return nil
	})
}

func (s *NATSSubscriber) SubscribePositionClosed(ctx context.Context, handler func(ports.PositionClosed)) error {
	return s.subscribe(ctx, "pqap.position.closed", "risk-manager-position-closed", func(data []byte) error {
		var event ports.PositionClosed
		if err := json.Unmarshal(data, &event); err != nil {
			s.logger.Error("failed to unmarshal position closed event", zap.Error(err))
			return err
		}
		handler(event)
		return nil
	})
}

func (s *NATSSubscriber) SubscribePositionUpdated(ctx context.Context, handler func(ports.PositionUpdated)) error {
	return s.subscribe(ctx, "pqap.position.updated", "risk-manager-position-updated", func(data []byte) error {
		var event ports.PositionUpdated
		if err := json.Unmarshal(data, &event); err != nil {
			s.logger.Error("failed to unmarshal position updated event", zap.Error(err))
			return err
		}
		handler(event)
		return nil
	})
}

func (s *NATSSubscriber) SubscribeEmergencyStop(ctx context.Context, handler func(ports.EmergencyStop)) error {
	return s.subscribe(ctx, "pqap.risk.emergency", "risk-manager-emergency", func(data []byte) error {
		var event ports.EmergencyStop
		if err := json.Unmarshal(data, &event); err != nil {
			s.logger.Error("failed to unmarshal emergency stop event", zap.Error(err))
			return err
		}
		handler(event)
		return nil
	})
}

func (s *NATSSubscriber) SubscribeCapitalUpdated(ctx context.Context, handler func(ports.CapitalUpdated)) error {
	return s.subscribe(ctx, "pqap.portfolio.capital_updated", "risk-manager-capital-updated", func(data []byte) error {
		var event ports.CapitalUpdated
		if err := json.Unmarshal(data, &event); err != nil {
			s.logger.Error("failed to unmarshal capital updated event", zap.Error(err))
			return err
		}
		handler(event)
		return nil
	})
}

func (s *NATSSubscriber) SubscribeRiskAlert(ctx context.Context, handler func(ports.RiskAlert)) error {
	return s.subscribe(ctx, "pqap.risk.alert", "risk-manager-risk-alert", func(data []byte) error {
		var event ports.RiskAlert
		if err := json.Unmarshal(data, &event); err != nil {
			s.logger.Error("failed to unmarshal risk alert event", zap.Error(err))
			return err
		}
		handler(event)
		return nil
	})
}

func (s *NATSSubscriber) SubscribeRiskCommand(ctx context.Context, handler func(map[string]interface{})) error {
	return s.subscribe(ctx, "pqap.risk.command", "risk-manager-risk-command", func(data []byte) error {
		var cmd map[string]interface{}
		if err := json.Unmarshal(data, &cmd); err != nil {
			s.logger.Error("failed to unmarshal risk command", zap.Error(err))
			return err
		}
		handler(cmd)
		return nil
	})
}

func (s *NATSSubscriber) subscribe(ctx context.Context, subject, durable string, handler func([]byte) error) error {
	sub, err := s.js.Subscribe(subject, func(msg *nats.Msg) {
		if err := handler(msg.Data); err != nil {
			s.logger.Error("handler failed, terming message",
				zap.String("subject", subject),
				zap.Error(err),
			)
			msg.Term()
			return
		}
		msg.Ack()
	}, nats.Durable(durable), nats.ManualAck())
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", subject, err)
	}

	s.subsMu.Lock()
	s.subs = append(s.subs, sub)
	s.subsMu.Unlock()

	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
	}()

	s.logger.Info("subscribed to subject", zap.String("subject", subject))
	return nil
}

func (s *NATSSubscriber) Close() error {
	s.subsMu.Lock()
	for _, sub := range s.subs {
		sub.Unsubscribe()
	}
	s.subsMu.Unlock()
	s.conn.Close()
	s.connected.Store(false)
	if s.connStatus != nil {
		s.connStatus.Set(0)
	}
	return nil
}
