package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/pqap/services/scanner/internal/catalog"
	"go.uber.org/zap"
)

type MarketEvent struct {
	EventID   string          `json:"event_id"`
	EventType string          `json:"event_type"`
	Timestamp time.Time       `json:"timestamp"`
	Source    string          `json:"source"`
	Payload   catalog.Market  `json:"payload"`
}

type NATSPublisher struct {
	conn   *nats.Conn
	js     nats.JetStreamContext
	logger *zap.Logger
}

func NewNATSPublisher(url string, logger *zap.Logger) (*NATSPublisher, error) {
	opts := []nats.Option{
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Warn("nats disconnected", zap.Error(err))
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("nats reconnected", zap.String("url", nc.ConnectedUrl()))
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
		Name:     "PQAP_MARKETS",
		Subjects: []string{"pqap.market.>"},
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

func (p *NATSPublisher) PublishPriceUpdate(ctx context.Context, market catalog.Market) error {
	if market.ID == "" {
		return fmt.Errorf("market ID is empty")
	}

	event := MarketEvent{
		EventID:   uuid.New().String(),
		EventType: "MarketPriceUpdated",
		Timestamp: time.Now().UTC(),
		Source:    "scanner",
		Payload:   market,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Fix #7: Sanitize market ID to prevent NATS subject injection
	subject := fmt.Sprintf("pqap.market.%s.price", sanitizeSubject(market.ID))
	_, err = p.js.Publish(subject, data)
	if err != nil {
		p.logger.Error("failed to publish price update", zap.Error(err), zap.String("market_id", market.ID))
		return err
	}

	p.logger.Debug("published price update", zap.String("market_id", market.ID), zap.String("subject", subject))
	return nil
}

func (p *NATSPublisher) PublishMarketDiscovered(ctx context.Context, market catalog.Market) error {
	if market.ID == "" {
		return fmt.Errorf("market ID is empty")
	}

	event := MarketEvent{
		EventID:   uuid.New().String(),
		EventType: "MarketDiscovered",
		Timestamp: time.Now().UTC(),
		Source:    "scanner",
		Payload:   market,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = p.js.Publish("pqap.market.discovered", data)
	if err != nil {
		p.logger.Error("failed to publish market discovered", zap.Error(err), zap.String("market_id", market.ID))
		return err
	}

	p.logger.Info("published market discovered", zap.String("market_id", market.ID))
	return nil
}

func (p *NATSPublisher) PublishMarketStale(ctx context.Context, market catalog.Market) error {
	if market.ID == "" {
		return fmt.Errorf("market ID is empty")
	}

	event := MarketEvent{
		EventID:   uuid.New().String(),
		EventType: "MarketStaleDetected",
		Timestamp: time.Now().UTC(),
		Source:    "scanner",
		Payload:   market,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Fix #1: Use flat subject pqap.market.stale (market_id already in payload)
	_, err = p.js.Publish("pqap.market.stale", data)
	if err != nil {
		p.logger.Error("failed to publish market stale", zap.Error(err), zap.String("market_id", market.ID))
		return err
	}

	p.logger.Warn("published market stale", zap.String("market_id", market.ID))
	return nil
}

func (p *NATSPublisher) IsConnected() bool {
	return p.conn.IsConnected()
}

func (p *NATSPublisher) Close() error {
	p.conn.Close()
	return nil
}

// sanitizeSubject removes NATS wildcard characters (* and >) from subject components
// to prevent unintended subject matching or injection.
func sanitizeSubject(s string) string {
	s = strings.ReplaceAll(s, "*", "_")
	s = strings.ReplaceAll(s, ">", "_")
	return s
}
