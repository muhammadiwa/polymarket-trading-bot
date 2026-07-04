package scanner_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pqap/services/scanner/internal/catalog"
	ws "github.com/pqap/services/scanner/internal/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func TestWebSocketClient_ConnectInvalidURL(t *testing.T) {
	logger := zap.NewNop()
	client := ws.NewClient("ws://invalid-host:9999", 100*time.Millisecond, 1*time.Second, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestWebSocketClient_ParsePriceMessage(t *testing.T) {
	tests := []struct {
		name        string
		msg         ws.WSMessage
		wantYES     string
		wantNO      string
		wantSpread  string
		wantMarket  string
	}{
		{
			name: "price_change with Prices field",
			msg: ws.WSMessage{
				Type:   "price_change",
				Market: "market-1",
				Prices: &ws.PriceData{YES: "0.65", NO: "0.30"},
			},
			wantYES:    "0.65",
			wantNO:     "0.30",
			wantSpread: "0.05",
			wantMarket: "market-1",
		},
		{
			name: "price_change with Assets field",
			msg: ws.WSMessage{
				Type:   "price_change",
				Market: "market-2",
				Assets: []ws.AssetData{
					{AssetID: "a1", Price: "0.70", Outcome: "Yes"},
					{AssetID: "a2", Price: "0.25", Outcome: "No"},
				},
			},
			wantYES:    "0.70",
			wantNO:     "0.25",
			wantSpread: "0.05",
			wantMarket: "market-2",
		},
		{
			name: "spread calculation edge case equal prices",
			msg: ws.WSMessage{
				Type:   "price_change",
				Market: "market-3",
				Prices: &ws.PriceData{YES: "0.50", NO: "0.50"},
			},
			wantYES:    "0.50",
			wantNO:     "0.50",
			wantSpread: "0",
			wantMarket: "market-3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var received catalog.Market
			logger := zap.NewNop()
			client := ws.NewClient("ws://dummy", 100*time.Millisecond, 1*time.Second, logger)
			client.SetOnMessage(func(m catalog.Market) {
				received = m
			})

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					return
				}
				defer conn.Close()

				data, _ := json.Marshal(tt.msg)
				conn.WriteMessage(websocket.TextMessage, data)
				time.Sleep(50 * time.Millisecond)
			}))
			defer server.Close()

			wsURL := "ws" + server.URL[4:]
			client = ws.NewClient(wsURL, 100*time.Millisecond, 1*time.Second, logger)
			client.SetOnMessage(func(m catalog.Market) {
				received = m
			})

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			if err := client.Connect(ctx); err != nil {
				t.Fatalf("connect failed: %v", err)
			}
			defer client.Close()

			go client.ReadMessages(ctx)
			time.Sleep(200 * time.Millisecond)

			if received.ID != tt.wantMarket {
				t.Errorf("market ID = %q, want %q", received.ID, tt.wantMarket)
			}
			if received.YESPrice.String() != tt.wantYES {
				t.Errorf("YES price = %q, want %q", received.YESPrice.String(), tt.wantYES)
			}
			if received.NOPrice.String() != tt.wantNO {
				t.Errorf("NO price = %q, want %q", received.NOPrice.String(), tt.wantNO)
			}
			if received.Spread.String() != tt.wantSpread {
				t.Errorf("spread = %q, want %q", received.Spread.String(), tt.wantSpread)
			}
		})
	}
}

func TestWebSocketClient_Subscribe(t *testing.T) {
	var receivedMsg []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		receivedMsg = msg
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	logger := zap.NewNop()
	client := ws.NewClient(wsURL, 100*time.Millisecond, 1*time.Second, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer client.Close()

	marketIDs := []string{"m1", "m2", "m3"}
	if err := client.Subscribe(marketIDs); err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	var parsed map[string]interface{}
	if err := json.Unmarshal(receivedMsg, &parsed); err != nil {
		t.Fatalf("failed to parse subscribe message: %v", err)
	}

	if parsed["type"] != "subscribe" {
		t.Errorf("type = %v, want subscribe", parsed["type"])
	}

	markets, ok := parsed["markets"].([]interface{})
	if !ok {
		t.Fatal("markets field missing or wrong type")
	}
	if len(markets) != 3 {
		t.Errorf("markets count = %d, want 3", len(markets))
	}
}

func TestWebSocketClient_SubscribeNotConnected(t *testing.T) {
	logger := zap.NewNop()
	client := ws.NewClient("ws://dummy", 100*time.Millisecond, 1*time.Second, logger)

	err := client.Subscribe([]string{"m1"})
	if err == nil {
		t.Fatal("expected error when subscribing without connection, got nil")
	}
}

func TestWebSocketClient_Reconnect(t *testing.T) {
	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		if attempt == 1 {
			conn.Close()
			return
		}
		defer conn.Close()
		time.Sleep(500 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	logger := zap.NewNop()
	client := ws.NewClient(wsURL, 50*time.Millisecond, 200*time.Millisecond, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	connected := make(chan struct{}, 5)
	client.SetOnConnect(func() {
		connected <- struct{}{}
	})

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("initial connect failed: %v", err)
	}
	defer client.Close()

	go client.ReadMessages(ctx)

	select {
	case <-connected:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for reconnect")
	}

	if !client.IsConnected() {
		t.Error("expected client to be connected after reconnect")
	}
}

func TestWebSocketClient_IgnoresNonPriceMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		msg := ws.WSMessage{Type: "heartbeat", Market: "m1"}
		data, _ := json.Marshal(msg)
		conn.WriteMessage(websocket.TextMessage, data)
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	logger := zap.NewNop()
	called := false
	client := ws.NewClient(wsURL, 100*time.Millisecond, 1*time.Second, logger)
	client.SetOnMessage(func(m catalog.Market) {
		called = true
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer client.Close()

	go client.ReadMessages(ctx)
	time.Sleep(200 * time.Millisecond)

	if called {
		t.Error("onMessage should not be called for non-price_change messages")
	}
}
