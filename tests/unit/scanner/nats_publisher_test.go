package scanner_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pqap/services/scanner/adapters"
	"github.com/pqap/services/scanner/internal/catalog"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func setupNATSTest(t *testing.T) (*adapters.NATSPublisher, *nats.Conn) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping NATS test in short mode")
	}

	logger, _ := zap.NewDevelopment()
	publisher, err := adapters.NewNATSPublisher("nats://localhost:4222", logger)
	if err != nil {
		t.Skip("NATS not available, skipping test")
	}

	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		publisher.Close()
		t.Skip("NATS not available, skipping test")
	}

	return publisher, nc
}

func TestNATSPublishPriceUpdate(t *testing.T) {
	publisher, _ := setupNATSTest(t)
	defer publisher.Close()

	market := catalog.Market{
		ID:       "test-market-1",
		Slug:     "test-market",
		YESPrice: decimal.NewFromFloat(0.55),
		NOPrice:  decimal.NewFromFloat(0.42),
		IsActive: true,
	}

	err := publisher.PublishPriceUpdate(context.Background(), market)
	if err != nil {
		t.Fatalf("failed to publish price update: %v", err)
	}
}

func TestNATSPublishMarketDiscovered(t *testing.T) {
	publisher, _ := setupNATSTest(t)
	defer publisher.Close()

	market := catalog.Market{
		ID:       "test-market-2",
		Slug:     "new-market",
		YESPrice: decimal.NewFromFloat(0.50),
		NOPrice:  decimal.NewFromFloat(0.50),
		IsActive: true,
	}

	err := publisher.PublishMarketDiscovered(context.Background(), market)
	if err != nil {
		t.Fatalf("failed to publish market discovered: %v", err)
	}
}

func TestNATSEventSchema(t *testing.T) {
	publisher, nc := setupNATSTest(t)
	defer publisher.Close()
	defer nc.Close()

	market := catalog.Market{
		ID:       "test-market-3",
		Slug:     "schema-test",
		YESPrice: decimal.NewFromFloat(0.65),
		NOPrice:  decimal.NewFromFloat(0.32),
	}

	msgCh := make(chan *nats.Msg, 1)
	sub, _ := nc.Subscribe("pqap.market.>", func(msg *nats.Msg) {
		msgCh <- msg
	})
	defer sub.Unsubscribe()
	nc.Flush()

	publisher.PublishPriceUpdate(context.Background(), market)

	select {
	case msg := <-msgCh:
		var event adapters.MarketEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			t.Fatalf("failed to unmarshal event: %v", err)
		}

		if event.EventID == "" {
			t.Error("event_id should not be empty")
		}
		if event.EventType != "MarketPriceUpdated" {
			t.Errorf("expected event_type 'MarketPriceUpdated', got '%s'", event.EventType)
		}
		if event.Source != "scanner" {
			t.Errorf("expected source 'scanner', got '%s'", event.Source)
		}
		if event.Timestamp.IsZero() {
			t.Error("timestamp should not be zero")
		}
		if event.Payload.ID != "test-market-3" {
			t.Errorf("expected payload ID 'test-market-3', got '%s'", event.Payload.ID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestNATSIdempotency(t *testing.T) {
	publisher, nc := setupNATSTest(t)
	defer publisher.Close()
	defer nc.Close()

	market := catalog.Market{
		ID:       "test-market-4",
		YESPrice: decimal.NewFromFloat(0.50),
		NOPrice:  decimal.NewFromFloat(0.50),
	}

	eventIDs := make(map[string]bool)
	msgCh := make(chan *nats.Msg, 2)
	sub, _ := nc.Subscribe("pqap.market.>", func(msg *nats.Msg) {
		msgCh <- msg
	})
	defer sub.Unsubscribe()
	nc.Flush()

	publisher.PublishPriceUpdate(context.Background(), market)
	publisher.PublishPriceUpdate(context.Background(), market)

	for i := 0; i < 2; i++ {
		select {
		case msg := <-msgCh:
			var event adapters.MarketEvent
			json.Unmarshal(msg.Data, &event)
			if eventIDs[event.EventID] {
				t.Error("duplicate event_id detected")
			}
			eventIDs[event.EventID] = true
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for event")
		}
	}
}
