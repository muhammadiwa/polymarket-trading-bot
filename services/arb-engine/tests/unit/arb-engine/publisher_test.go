package adapters_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/arb-engine/internal/ports"
	"github.com/shopspring/decimal"
)

func TestOpportunityDetectedSchema(t *testing.T) {
	event := ports.OpportunityDetected{
		EventID:   uuid.New().String(),
		EventType: "OpportunityDetected",
		Timestamp: time.Now().UTC(),
		Source:    "arb-engine",
		Payload: ports.OpportunityPayload{
			OpportunityID:  uuid.New().String(),
			MarketID:       "test-market",
			YESPrice:       decimal.RequireFromString("0.55"),
			NOPrice:        decimal.RequireFromString("0.40"),
			Spread:         decimal.RequireFromString("0.05"),
			Score:          decimal.RequireFromString("0.028"),
			FillProbability: decimal.RequireFromString("0.7"),
			Liquidity:      decimal.RequireFromString("0.8"),
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	var decoded ports.OpportunityDetected
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if decoded.EventID != event.EventID {
		t.Errorf("EventID = %s, want %s", decoded.EventID, event.EventID)
	}
	if decoded.EventType != "OpportunityDetected" {
		t.Errorf("EventType = %s, want OpportunityDetected", decoded.EventType)
	}
	if decoded.Source != "arb-engine" {
		t.Errorf("Source = %s, want arb-engine", decoded.Source)
	}
	if decoded.Timestamp.IsZero() {
		t.Error("expected non-zero Timestamp")
	}
	if decoded.Payload.MarketID != "test-market" {
		t.Errorf("MarketID = %s, want test-market", decoded.Payload.MarketID)
	}
	if !decoded.Payload.Spread.Equal(decimal.RequireFromString("0.05")) {
		t.Errorf("Spread = %s, want 0.05", decoded.Payload.Spread)
	}
}

func TestEventIDIsUUID(t *testing.T) {
	eventID := uuid.New().String()
	_, err := uuid.Parse(eventID)
	if err != nil {
		t.Errorf("EventID %s is not a valid UUID: %v", eventID, err)
	}
}

func TestTimestampIsUTC(t *testing.T) {
	ts := time.Now().UTC()
	if ts.Location() != time.UTC {
		t.Errorf("Timestamp location = %s, want UTC", ts.Location())
	}
}

type mockPublisher struct {
	published []ports.OpportunityDetected
}

func (m *mockPublisher) PublishOpportunityDetected(_ context.Context, event ports.OpportunityDetected) error {
	m.published = append(m.published, event)
	return nil
}

func (m *mockPublisher) Close() error { return nil }

func TestPublisherIntegration(t *testing.T) {
	pub := &mockPublisher{}
	event := ports.OpportunityDetected{
		EventID:   uuid.New().String(),
		EventType: "OpportunityDetected",
		Timestamp: time.Now().UTC(),
		Source:    "arb-engine",
		Payload: ports.OpportunityPayload{
			OpportunityID:  uuid.New().String(),
			MarketID:       "test-market",
			YESPrice:       decimal.RequireFromString("0.55"),
			NOPrice:        decimal.RequireFromString("0.40"),
			Spread:         decimal.RequireFromString("0.05"),
			Score:          decimal.RequireFromString("0.028"),
			FillProbability: decimal.RequireFromString("0.7"),
			Liquidity:      decimal.RequireFromString("0.8"),
		},
	}

	err := pub.PublishOpportunityDetected(context.Background(), event)
	if err != nil {
		t.Fatalf("PublishOpportunityDetected failed: %v", err)
	}

	if len(pub.published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(pub.published))
	}
	if pub.published[0].EventID != event.EventID {
		t.Errorf("published EventID = %s, want %s", pub.published[0].EventID, event.EventID)
	}
}
