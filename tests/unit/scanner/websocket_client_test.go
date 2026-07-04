package scanner_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	ws "github.com/pqap/services/scanner/internal/websocket"
	"github.com/pqap/services/scanner/internal/catalog"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

var testUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func newTestWSServer(t *testing.T, onConnect func(conn *websocket.Conn)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade error: %v", err)
			return
		}
		defer conn.Close()
		onConnect(conn)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}))
}

func TestClientConnect(t *testing.T) {
	server := newTestWSServer(t, func(conn *websocket.Conn) {})
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	logger, _ := zap.NewDevelopment()
	client := ws.NewClient(wsURL, 1*time.Second, 60*time.Second, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer client.Close()

	if !client.IsConnected() {
		t.Fatal("expected client to be connected")
	}
}

func TestClientConnectInvalidURL(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	client := ws.NewClient("ws://invalid-host:9999", 100*time.Millisecond, 500*time.Millisecond, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestClientParsePriceMessage(t *testing.T) {
	msgCh := make(chan catalog.Market, 1)

	server := newTestWSServer(t, func(conn *websocket.Conn) {
		msg := map[string]interface{}{
			"type":   "price_change",
			"market": "test-market-1",
			"assets": []map[string]string{
				{"asset_id": "a1", "price": "0.55", "outcome": "Yes"},
				{"asset_id": "a2", "price": "0.42", "outcome": "No"},
			},
		}
		data, _ := json.Marshal(msg)
		conn.WriteMessage(websocket.TextMessage, data)
		time.Sleep(100 * time.Millisecond)
	})
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	logger, _ := zap.NewDevelopment()
	client := ws.NewClient(wsURL, 1*time.Second, 60*time.Second, logger)

	client.SetOnMessage(func(m catalog.Market) {
		msgCh <- m
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer client.Close()

	select {
	case m := <-msgCh:
		if m.ID != "test-market-1" {
			t.Errorf("expected market ID 'test-market-1', got '%s'", m.ID)
		}
		if !m.YESPrice.Equal(decimal.NewFromFloat(0.55)) {
			t.Errorf("expected YES price 0.55, got %s", m.YESPrice)
		}
		if !m.NOPrice.Equal(decimal.NewFromFloat(0.42)) {
			t.Errorf("expected NO price 0.42, got %s", m.NOPrice)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestClientSubscribe(t *testing.T) {
	var receivedSub []byte
	subReceived := make(chan struct{})

	server := newTestWSServer(t, func(conn *websocket.Conn) {
		_, receivedSub, _ = conn.ReadMessage()
		close(subReceived)
	})
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	logger, _ := zap.NewDevelopment()
	client := ws.NewClient(wsURL, 1*time.Second, 60*time.Second, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer client.Close()

	err := client.Subscribe([]string{"market-1", "market-2"})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	select {
	case <-subReceived:
		var subMsg map[string]interface{}
		json.Unmarshal(receivedSub, &subMsg)
		if subMsg["type"] != "subscribe" {
			t.Errorf("expected type 'subscribe', got '%v'", subMsg["type"])
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for subscription message")
	}
}

func TestClientReconnect(t *testing.T) {
	connectionCount := 0
	server := newTestWSServer(t, func(conn *websocket.Conn) {
		connectionCount++
		time.Sleep(50 * time.Millisecond)
	})
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	logger, _ := zap.NewDevelopment()
	client := ws.NewClient(wsURL, 100*time.Millisecond, 500*time.Millisecond, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer client.Close()

	if connectionCount < 1 {
		t.Error("expected at least 1 connection")
	}
}
