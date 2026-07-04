package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pqap/services/scanner/internal/catalog"
	"github.com/pqap/services/scanner/metrics"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type WSMessage struct {
	Type    string      `json:"type"`
	Market  string      `json:"market"`
	Prices  *PriceData  `json:"prices,omitempty"`
	Assets  []AssetData `json:"assets,omitempty"`
}

type PriceData struct {
	YES string `json:"yes"`
	NO  string `json:"no"`
}

type AssetData struct {
	AssetID string `json:"asset_id"`
	Price   string `json:"price"`
	Outcome string `json:"outcome"`
}

type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

type circuitBreaker struct {
	mu           sync.Mutex
	state        CircuitState
	failures     int
	threshold    int
	resetTimeout time.Duration
	lastFailure  time.Time
}

func newCircuitBreaker(threshold int, resetTimeout time.Duration) *circuitBreaker {
	return &circuitBreaker{
		state:        StateClosed,
		threshold:    threshold,
		resetTimeout: resetTimeout,
	}
}

func (cb *circuitBreaker) Allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpen {
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = StateHalfOpen
			return nil
		}
		return fmt.Errorf("circuit breaker open")
	}
	return nil
}

func (cb *circuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.state = StateClosed
}

func (cb *circuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFailure = time.Now()
	if cb.failures >= cb.threshold {
		cb.state = StateOpen
	}
}

type Client struct {
	url            string
	conn           *websocket.Conn
	logger         *zap.Logger
	reconnectMu    sync.Mutex
	writeMu        sync.Mutex
	initialDelay   time.Duration
	maxDelay       time.Duration
	reconnectCount atomic.Int32
	connected      bool
	onConnect      func()
	onMessage      func(catalog.Market)
	onDisconnect   func()
	cbMu           sync.RWMutex // Protects onConnect, onMessage, onDisconnect
	cond           *sync.Cond   // Signals when connected state changes
	wsCircuit      *circuitBreaker
}

func NewClient(url string, initialDelay, maxDelay time.Duration, logger *zap.Logger) *Client {
	c := &Client{
		url:          url,
		logger:       logger,
		initialDelay: initialDelay,
		maxDelay:     maxDelay,
		wsCircuit:    newCircuitBreaker(5, 30*time.Second),
	}
	c.cond = sync.NewCond(&c.reconnectMu)
	return c
}

func (c *Client) SetOnConnect(fn func()) {
	c.cbMu.Lock()
	c.onConnect = fn
	c.cbMu.Unlock()
}

func (c *Client) SetOnMessage(fn func(catalog.Market)) {
	c.cbMu.Lock()
	c.onMessage = fn
	c.cbMu.Unlock()
}

func (c *Client) SetOnDisconnect(fn func()) {
	c.cbMu.Lock()
	c.onDisconnect = fn
	c.cbMu.Unlock()
}

func (c *Client) Connect(ctx context.Context) error {
	c.reconnectMu.Lock()
	defer c.reconnectMu.Unlock()

	return c.connect(ctx)
}

func (c *Client) connect(ctx context.Context) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, c.url, nil)
	if err != nil {
		c.connected = false
		return err
	}

	c.conn = conn
	c.connected = true
	c.reconnectCount.Store(0)
	c.logger.Info("ws_connected",
		zap.String("service", "scanner"),
		zap.String("url", c.url),
	)

	// Signal waiting goroutines that we are connected
	c.cond.Broadcast()

	// Call onConnect asynchronously to avoid deadlock
	if c.onConnect != nil {
		c.cbMu.RLock()
		onConnect := c.onConnect
		c.cbMu.RUnlock()
		go onConnect()
	}

	return nil
}

func (c *Client) IsConnected() bool {
	c.reconnectMu.Lock()
	defer c.reconnectMu.Unlock()
	return c.connected
}

func (c *Client) ReadMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		c.reconnectMu.Lock()
		for !c.connected || c.conn == nil {
			select {
			case <-ctx.Done():
				c.reconnectMu.Unlock()
				return
			default:
			}
			c.cond.Wait()
		}
		conn := c.conn
		c.reconnectMu.Unlock()

		_, message, err := conn.ReadMessage()
		if err != nil {
			c.logger.Error("websocket read error", zap.Error(err))
			c.handleDisconnect(ctx)
			continue
		}

		c.processMessage(message)
	}
}

func (c *Client) processMessage(data []byte) {
	var msg WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		c.logger.Warn("failed to parse message", zap.Error(err))
		return
	}

	if msg.Type != "price_change" {
		return
	}

	if msg.Prices == nil && len(msg.Assets) == 0 {
		return
	}

	market := catalog.Market{
		ID: msg.Market,
	}

	if len(msg.Assets) > 0 {
		for _, asset := range msg.Assets {
			switch asset.Outcome {
			case "Yes":
				market.YESPrice = parseDecimal(asset.Price)
			case "No":
				market.NOPrice = parseDecimal(asset.Price)
			}
		}
	} else {
		market.YESPrice = parseDecimal(msg.Prices.YES)
		market.NOPrice = parseDecimal(msg.Prices.NO)
	}

	one := decimal.NewFromInt(1)
	market.Spread = one.Sub(market.YESPrice).Sub(market.NOPrice).Abs()

	c.cbMu.RLock()
	onMessage := c.onMessage
	c.cbMu.RUnlock()

	if onMessage != nil {
		onMessage(market)
	}
}

func (c *Client) handleDisconnect(ctx context.Context) {
	c.reconnectMu.Lock()
	c.connected = false
	c.cond.Broadcast() // Wake up ReadMessages to observe disconnection
	c.reconnectMu.Unlock()

	// Invoke onDisconnect callback (e.g. to reset metrics)
	c.cbMu.RLock()
	onDisconnect := c.onDisconnect
	c.cbMu.RUnlock()
	if onDisconnect != nil {
		onDisconnect()
	}

	c.logger.Warn("ws_disconnected",
		zap.String("service", "scanner"),
	)
	c.reconnectLoop(ctx)
}

func (c *Client) reconnectLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := c.wsCircuit.Allow(); err != nil {
			c.logger.Warn("circuit_breaker_open",
				zap.String("service", "scanner"),
			)
			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Second):
				continue
			}
		}

		delay := c.calculateBackoff()
		count := c.reconnectCount.Load()
		c.logger.Info("ws_reconnecting",
			zap.String("service", "scanner"),
			zap.Duration("delay", delay),
			zap.Int("attempt", int(count)+1),
		)

		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}

		// Fix #8: Increment counter on every attempt, not just successful reconnects
		metrics.WSReconnectTotal.Inc()

		c.reconnectMu.Lock()
		err := c.connect(ctx)
		c.reconnectMu.Unlock()

		if err == nil {
			c.wsCircuit.RecordSuccess()
			c.logger.Info("ws_reconnected",
				zap.String("service", "scanner"),
			)
			return
		}

		c.wsCircuit.RecordFailure()
		c.reconnectCount.Add(1)
		count = c.reconnectCount.Load()
		c.logger.Error("ws_reconnect_failed",
			zap.String("service", "scanner"),
			zap.Error(err),
			zap.Int("attempt", int(count)),
		)
	}
}

func (c *Client) calculateBackoff() time.Duration {
	delay := float64(c.initialDelay) * math.Pow(2, float64(c.reconnectCount.Load()))
	if delay > float64(c.maxDelay) {
		delay = float64(c.maxDelay)
	}

	jitter := (rand.Float64() - 0.5) * float64(time.Second)
	result := delay + jitter
	if result < 0 {
		result = 0
	}
	return time.Duration(result)
}

func (c *Client) CalculateBackoffForTesting() time.Duration {
	return c.calculateBackoff()
}

func (c *Client) Subscribe(marketIDs []string) error {
	if !c.IsConnected() {
		return ErrNotConnected
	}

	msg := map[string]interface{}{
		"type":       "subscribe",
		"markets":    marketIDs,
		"assets_ids": []string{},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func parseDecimal(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

var ErrNotConnected = &WSClientError{"websocket not connected"}

type WSClientError struct {
	msg string
}

func (e *WSClientError) Error() string {
	return e.msg
}
