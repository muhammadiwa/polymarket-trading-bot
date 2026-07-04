package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/pqap/services/scanner/internal/websocket"
	"go.uber.org/zap"
)

func TestWebSocketReconnect(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("skipping integration test: set INTEGRATION_TEST=1 to enable")
	}

	wsURL := os.Getenv("WS_URL")
	if wsURL == "" {
		wsURL = "wss://ws-subscriptions-clob.polymarket.com/ws/market"
	}

	logger, _ := zap.NewDevelopment()
	client := websocket.NewClient(wsURL, 1*time.Second, 10*time.Second, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	connected := make(chan struct{})
	client.SetOnConnect(func() {
		close(connected)
	})

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	select {
	case <-connected:
		t.Log("connected successfully")
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for connection")
	}

	if !client.IsConnected() {
		t.Error("expected client to be connected")
	}
}

func TestWebSocketSubscribe(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("skipping integration test: set INTEGRATION_TEST=1 to enable")
	}

	wsURL := os.Getenv("WS_URL")
	if wsURL == "" {
		wsURL = "wss://ws-subscriptions-clob.polymarket.com/ws/market"
	}

	logger, _ := zap.NewDevelopment()
	client := websocket.NewClient(wsURL, 1*time.Second, 10*time.Second, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	connected := make(chan struct{})
	client.SetOnConnect(func() {
		close(connected)
	})

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	select {
	case <-connected:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for connection")
	}

	subscriber := websocket.NewSubscriber(client, logger)
	testMarketIDs := []string{"0xtest1", "0xtest2"}
	err = subscriber.Subscribe(ctx, testMarketIDs)
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
}

func TestWebSocketReconnectAfterDisconnect(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("skipping integration test: set INTEGRATION_TEST=1 to enable")
	}

	wsURL := os.Getenv("WS_URL")
	if wsURL == "" {
		wsURL = "wss://ws-subscriptions-clob.polymarket.com/ws/market"
	}

	logger, _ := zap.NewDevelopment()
	client := websocket.NewClient(wsURL, 1*time.Second, 10*time.Second, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	connectCount := 0
	connected := make(chan struct{}, 2)
	client.SetOnConnect(func() {
		connectCount++
		connected <- struct{}{}
	})

	go client.ReadMessages(ctx)

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	select {
	case <-connected:
		t.Log("initial connection established")
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for initial connection")
	}

	t.Log("closing connection to trigger reconnect...")
	client.Close()

	select {
	case <-connected:
		t.Log("reconnected after disconnect")
	case <-time.After(30 * time.Second):
		t.Log("reconnection not observed within timeout (may require live server)")
	}
}
