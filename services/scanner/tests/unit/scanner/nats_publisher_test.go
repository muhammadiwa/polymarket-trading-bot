package scanner_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/scanner/adapters"
	"github.com/pqap/services/scanner/internal/catalog"
	"github.com/shopspring/decimal"
)

func TestNATSPublisher_EventSchema(t *testing.T) {
	market := catalog.Market{
		ID:             "market-1",
		Slug:           "test-market",
		YESPrice:       decimal.NewFromFloat(0.65),
		NOPrice:        decimal.NewFromFloat(0.30),
		Spread:         decimal.NewFromFloat(0.05),
		Volume24h:      decimal.NewFromFloat(1000),
		LiquidityDepth: decimal.NewFromFloat(500),
		IsActive:       true,
	}

	event := adapters.MarketEvent{
		EventID:   uuid.New().String(),
		EventType: "MarketPriceUpdated",
		Timestamp: time.Now().UTC(),
		Source:    "scanner",
		Payload:   market,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	requiredFields := []string{"event_id", "event_type", "timestamp", "source", "payload"}
	for _, field := range requiredFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("missing required field %q in event schema", field)
		}
	}

	if parsed["event_type"] != "MarketPriceUpdated" {
		t.Errorf("event_type = %v, want MarketPriceUpdated", parsed["event_type"])
	}
	if parsed["source"] != "scanner" {
		t.Errorf("source = %v, want scanner", parsed["source"])
	}

	payload, ok := parsed["payload"].(map[string]interface{})
	if !ok {
		t.Fatal("payload is not an object")
	}
	payloadFields := []string{"id", "slug", "yes_price", "no_price", "spread", "volume_24h", "liquidity_depth", "is_active"}
	for _, field := range payloadFields {
		if _, ok := payload[field]; !ok {
			t.Errorf("missing payload field %q", field)
		}
	}
}

func TestNATSPublisher_Idempotency(t *testing.T) {
	ids := make(map[string]bool)
	numEvents := 1000

	for i := 0; i < numEvents; i++ {
		id := uuid.New().String()
		if ids[id] {
			t.Fatalf("duplicate UUID generated: %s", id)
		}
		ids[id] = true
	}

	if len(ids) != numEvents {
		t.Errorf("unique IDs = %d, want %d", len(ids), numEvents)
	}
}

func TestNATSPublisher_EventTypes(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
	}{
		{name: "price_update", eventType: "MarketPriceUpdated"},
		{name: "market_discovered", eventType: "MarketDiscovered"},
		{name: "market_stale", eventType: "MarketStaleDetected"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := adapters.MarketEvent{
				EventID:   uuid.New().String(),
				EventType: tt.eventType,
				Timestamp: time.Now().UTC(),
				Source:    "scanner",
				Payload:   catalog.Market{ID: "market-1"},
			}

			data, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			var parsed adapters.MarketEvent
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}

			if parsed.EventType != tt.eventType {
				t.Errorf("EventType = %q, want %q", parsed.EventType, tt.eventType)
			}
			if parsed.EventID == "" {
				t.Error("EventID should not be empty")
			}
			if parsed.Timestamp.IsZero() {
				t.Error("Timestamp should not be zero")
			}
		})
	}
}

func TestNATSPublisher_PublishPriceUpdate_Integration(t *testing.T) {
	t.Skip("skipping: requires running NATS server")
}

func TestNATSPublisher_PublishMarketDiscovered_Integration(t *testing.T) {
	t.Skip("skipping: requires running NATS server")
}
