package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/pqap/services/scanner/adapters"
	"github.com/pqap/services/scanner/internal/catalog"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func TestNATSPublishPriceUpdate(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("skipping integration test: set INTEGRATION_TEST=1 to enable")
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	logger, _ := zap.NewDevelopment()
	publisher, err := adapters.NewNATSPublisher(natsURL, logger)
	if err != nil {
		t.Fatalf("failed to connect to NATS: %v", err)
	}
	defer publisher.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	market := catalog.Market{
		ID:       "integration-test-market-001",
		YESPrice: decimal.NewFromFloat(0.65),
		NOPrice:  decimal.NewFromFloat(0.35),
		Spread:   decimal.NewFromFloat(0.0),
	}

	err = publisher.PublishPriceUpdate(ctx, market)
	if err != nil {
		t.Fatalf("failed to publish price update: %v", err)
	}
}

func TestNATSPublishMarketDiscovered(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("skipping integration test: set INTEGRATION_TEST=1 to enable")
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	logger, _ := zap.NewDevelopment()
	publisher, err := adapters.NewNATSPublisher(natsURL, logger)
	if err != nil {
		t.Fatalf("failed to connect to NATS: %v", err)
	}
	defer publisher.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	market := catalog.Market{
		ID:   "integration-test-market-002",
		Slug: "test-market-002",
	}

	err = publisher.PublishMarketDiscovered(ctx, market)
	if err != nil {
		t.Fatalf("failed to publish market discovered: %v", err)
	}
}

func TestNATSPublishMarketStale(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("skipping integration test: set INTEGRATION_TEST=1 to enable")
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	logger, _ := zap.NewDevelopment()
	publisher, err := adapters.NewNATSPublisher(natsURL, logger)
	if err != nil {
		t.Fatalf("failed to connect to NATS: %v", err)
	}
	defer publisher.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	market := catalog.Market{
		ID: "integration-test-market-003",
	}

	err = publisher.PublishMarketStale(ctx, market)
	if err != nil {
		t.Fatalf("failed to publish market stale: %v", err)
	}
}
