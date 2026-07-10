package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/nats-io/nats.go"
	"github.com/pqap/services/risk-manager/internal/ports"
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
		{"PQAP_RISK", []string{"pqap.risk.>"}},
		{"PQAP_NOTIFICATIONS", []string{"pqap.notification.>"}},
		{"PQAP_ORDERS", []string{"pqap.order.>"}},
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

func (p *NATSPublisher) PublishRiskStateUpdated(ctx context.Context, event ports.RiskStateUpdated) error {
	return p.publish(ctx, "pqap.risk.state_updated", event)
}

func (p *NATSPublisher) PublishRiskDecisionLogged(ctx context.Context, event ports.RiskDecisionLogged) error {
	return p.publish(ctx, "pqap.risk.decision_logged", event)
}

func (p *NATSPublisher) PublishDailyBudgetWarning(ctx context.Context, event ports.DailyBudgetWarning) error {
	return p.publish(ctx, "pqap.risk.daily_budget_warning", event)
}

func (p *NATSPublisher) PublishRiskAlert(ctx context.Context, event ports.RiskAlert) error {
	return p.publish(ctx, "pqap.risk.alert", event)
}

func (p *NATSPublisher) PublishEmergencyStop(ctx context.Context, event ports.EmergencyStop) error {
	return p.publish(ctx, "pqap.risk.emergency", event)
}

func (p *NATSPublisher) PublishNotificationRequest(ctx context.Context, event ports.NotificationRequest) error {
	return p.publish(ctx, "pqap.notification.request", event)
}

func (p *NATSPublisher) PublishDrawdownWarning(ctx context.Context, event ports.DrawdownWarning) error {
	return p.publish(ctx, "pqap.risk.drawdown_warning", event)
}

func (p *NATSPublisher) PublishDrawdownReset(ctx context.Context, event ports.DrawdownReset) error {
	return p.publish(ctx, "pqap.risk.drawdown_reset", event)
}

func (p *NATSPublisher) PublishTradingResumed(ctx context.Context, event ports.TradingResumed) error {
	return p.publish(ctx, "pqap.risk.trading_resumed", event)
}

func (p *NATSPublisher) PublishCancelAllOrders(ctx context.Context, event ports.CancelAllOrders) error {
	return p.publish(ctx, "pqap.order.cancel_all", event)
}

func (p *NATSPublisher) PublishCorrelationRejection(ctx context.Context, event ports.CorrelationRejectionEvent) error {
	return p.publish(ctx, "pqap.risk.correlation.rejected", event)
}

func (p *NATSPublisher) PublishBatasiWinTriggered(ctx context.Context, event ports.BatasiWinTriggered) error {
	return p.publish(ctx, "pqap.risk.batasi.triggered", event)
}

func (p *NATSPublisher) PublishMetabolicRateAlert(ctx context.Context, event ports.MetabolicRateAlert) error {
	return p.publish(ctx, "pqap.risk.metabolic.alert", event)
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

	p.logger.Debug("published event", zap.String("subject", subject))
	return nil
}

func (p *NATSPublisher) IsConnected() bool {
	return p.connected.Load()
}

func (p *NATSPublisher) Close() error {
	p.conn.Close()
	p.connected.Store(false)
	if p.connStatus != nil {
		p.connStatus.Set(0)
	}
	return nil
}
