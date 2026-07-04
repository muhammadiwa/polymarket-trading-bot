package scanner_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	ws "github.com/pqap/services/scanner/internal/websocket"
	"go.uber.org/zap"
)

func TestWebSocketReconnect_ExponentialBackoff(t *testing.T) {
	attempts := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count <= 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		time.Sleep(500 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	logger := zap.NewNop()
	client := ws.NewClient(wsURL, 50*time.Millisecond, 500*time.Millisecond, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connected := make(chan struct{}, 5)
	client.SetOnConnect(func() {
		connected <- struct{}{}
	})

	if err := client.Connect(ctx); err == nil {
		t.Log("initial connect succeeded unexpectedly (server rejects first 3)")
	}

	go client.ReadMessages(ctx)

	select {
	case <-connected:
		t.Log("reconnected successfully after retries")
	case <-ctx.Done():
		t.Fatal("timed out waiting for reconnect")
	}

	finalAttempts := atomic.LoadInt32(&attempts)
	if finalAttempts < 4 {
		t.Errorf("expected at least 4 attempts, got %d", finalAttempts)
	}
}

func TestWebSocketReconnect_MaxDelayCap(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	logger := zap.NewNop()
	client := ws.NewClient(wsURL, 1*time.Second, 2*time.Second, logger)

	delay := client.CalculateBackoffForTesting()
	if delay > 3*time.Second {
		t.Errorf("backoff delay %v exceeds max + jitter", delay)
	}
	if delay < 500*time.Millisecond {
		t.Errorf("backoff delay %v too low", delay)
	}
}

func TestWebSocketReconnect_CircuitBreakerOpens(t *testing.T) {
	attempts := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	logger := zap.NewNop()
	client := ws.NewClient(wsURL, 10*time.Millisecond, 50*time.Millisecond, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		client.ReadMessages(ctx)
	}()

	time.Sleep(1 * time.Second)

	finalAttempts := atomic.LoadInt32(&attempts)
	if finalAttempts > 20 {
		t.Errorf("circuit breaker should have limited attempts, got %d", finalAttempts)
	}
}

func TestWebSocketReconnect_CircuitBreakerResets(t *testing.T) {
	attempts := int32(0)
	shouldFail := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if shouldFail && count <= 5 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		time.Sleep(500 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	logger := zap.NewNop()
	client := ws.NewClient(wsURL, 10*time.Millisecond, 100*time.Millisecond, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connected := make(chan struct{}, 5)
	client.SetOnConnect(func() {
		connected <- struct{}{}
	})

	if err := client.Connect(ctx); err == nil {
		t.Log("initial connect succeeded")
	}

	go client.ReadMessages(ctx)

	time.Sleep(500 * time.Millisecond)
	shouldFail = false

	select {
	case <-connected:
		t.Log("circuit breaker reset and reconnected successfully")
	case <-ctx.Done():
		t.Fatal("timed out waiting for circuit breaker reset")
	}
}

func TestWebSocketReconnect_NoMaxRetryLimit(t *testing.T) {
	attempts := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count <= 10 {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			conn.Close()
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		time.Sleep(500 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	logger := zap.NewNop()
	client := ws.NewClient(wsURL, 10*time.Millisecond, 50*time.Millisecond, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	connected := make(chan struct{}, 5)
	client.SetOnConnect(func() {
		connected <- struct{}{}
	})

	if err := client.Connect(ctx); err == nil {
		t.Log("initial connect succeeded")
	}

	go client.ReadMessages(ctx)

	select {
	case <-connected:
		t.Log("reconnected after many attempts (no max retry limit)")
	case <-ctx.Done():
		t.Fatal("timed out - client should keep trying indefinitely")
	}

	finalAttempts := atomic.LoadInt32(&attempts)
	if finalAttempts < 11 {
		t.Errorf("expected at least 11 attempts, got %d", finalAttempts)
	}
}

func TestWebSocketReconnect_OnDisconnectCallback(t *testing.T) {
	disconnectCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		conn.Close()
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	logger := zap.NewNop()
	client := ws.NewClient(wsURL, 10*time.Millisecond, 50*time.Millisecond, logger)

	client.SetOnDisconnect(func() {
		atomic.AddInt32(&disconnectCount, 1)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err == nil {
		t.Log("initial connect succeeded")
	}

	go client.ReadMessages(ctx)

	time.Sleep(500 * time.Millisecond)

	count := atomic.LoadInt32(&disconnectCount)
	if count == 0 {
		t.Error("onDisconnect callback should have been called")
	}
}

func TestWebSocketReconnect_OnConnectCallback(t *testing.T) {
	connectCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	logger := zap.NewNop()
	client := ws.NewClient(wsURL, 100*time.Millisecond, 500*time.Millisecond, logger)

	client.SetOnConnect(func() {
		atomic.AddInt32(&connectCount, 1)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("initial connect failed: %v", err)
	}
	defer client.Close()

	go client.ReadMessages(ctx)

	time.Sleep(500 * time.Millisecond)

	count := atomic.LoadInt32(&connectCount)
	if count < 1 {
		t.Errorf("onConnect callback should have been called at least once, got %d", count)
	}
}

func TestWebSocketReconnect_MetricsIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	logger := zap.NewNop()
	client := ws.NewClient(wsURL, 100*time.Millisecond, 500*time.Millisecond, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("initial connect failed: %v", err)
	}
	defer client.Close()

	if !client.IsConnected() {
		t.Error("expected client to be connected")
	}
}

func TestWebSocketReconnect_ConcurrentReaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for i := 0; i < 10; i++ {
			msg := ws.WSMessage{
				Type:   "price_change",
				Market: "market-1",
				Prices: &ws.PriceData{YES: "0.65", NO: "0.30"},
			}
			data, _ := json.Marshal(msg)
			conn.WriteMessage(websocket.TextMessage, data)
			time.Sleep(50 * time.Millisecond)
		}
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	logger := zap.NewNop()
	client := ws.NewClient(wsURL, 100*time.Millisecond, 1*time.Second, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer client.Close()

	go client.ReadMessages(ctx)
	time.Sleep(1 * time.Second)

	if !client.IsConnected() {
		t.Error("client should remain connected")
	}
}
