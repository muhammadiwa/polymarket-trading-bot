package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pqap/services/arb-engine/internal/ports"
	"go.uber.org/zap"
)

type NATSPublisher struct {
	conn   *nats.Conn
	js     nats.JetStreamContext
	logger *zap.Logger
}

func NewNATSPublisher(url string, logger *zap.Logger) (*NATSPublisher, error) {
	opts := []nats.Option{
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Warn("nats publisher disconnected", zap.Error(err))
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("nats publisher reconnected", zap.String("url", nc.ConnectedUrl()))
		}),
	}

	conn, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "PQAP_OPPORTUNITIES",
		Subjects: []string{"pqap.opportunity.>"},
	})
	if err != nil {
		logger.Warn("stream creation (may already exist)", zap.Error(err))
	}

	return &NATSPublisher{
		conn:   conn,
		js:     js,
		logger: logger,
	}, nil
}

func (p *NATSPublisher) PublishOpportunityDetected(ctx context.Context, event ports.OpportunityDetected) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal opportunity event: %w", err)
	}

	_, err = p.js.Publish("pqap.opportunity.detected", data)
	if err != nil {
		p.logger.Error("failed to publish opportunity detected",
			zap.Error(err),
			zap.String("opportunity_id", event.Payload.OpportunityID),
		)
		return err
	}

	p.logger.Info("published opportunity detected",
		zap.String("opportunity_id", event.Payload.OpportunityID),
		zap.String("market_id", event.Payload.MarketID),
		zap.String("score", event.Payload.Score.String()),
	)
	return nil
}

func (p *NATSPublisher) IsConnected() bool {
	return p.conn.IsConnected()
}

func (p *NATSPublisher) Close() error {
	p.conn.Close()
	return nil
}
