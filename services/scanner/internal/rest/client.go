package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/pqap/services/scanner/internal/catalog"
	"github.com/pqap/services/scanner/metrics"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type MarketResponse struct {
	ConditionID string `json:"condition_id"`
	Slug        string `json:"slug"`
	Tokens      []TokenResponse `json:"tokens"`
	Active      bool   `json:"active"`
	Volume24h   string `json:"volume_num_24hr,omitempty"`
	Liquidity   string `json:"liquidity_num,omitempty"`
}

type TokenResponse struct {
	TokenID string `json:"token_id"`
	Outcome string `json:"outcome"`
	Price   string `json:"price"`
}

type MarketsResponse struct {
	Markets []MarketResponse `json:"data"`
	NextCursor string `json:"next_cursor"`
}

type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	mu           sync.RWMutex
	state        CircuitState
	failures     int
	threshold    int
	resetTimeout time.Duration
	lastFailure  time.Time
}

func NewCircuitBreaker(threshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:        StateClosed,
		threshold:    threshold,
		resetTimeout: resetTimeout,
	}
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
	// Fix #6: Single lock block to eliminate TOCTOU race
	cb.mu.Lock()
	state := cb.state
	if state == StateOpen {
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = StateHalfOpen
			state = StateHalfOpen
		} else {
			cb.mu.Unlock()
			return fmt.Errorf("circuit breaker open")
		}
	}
	cb.mu.Unlock()

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()
		if cb.failures >= cb.threshold {
			cb.state = StateOpen
		}
		return err
	}

	cb.failures = 0
	cb.state = StateClosed
	return nil
}

type Client struct {
	baseURL        string
	httpClient     *http.Client
	circuitBreaker *CircuitBreaker
	logger         *zap.Logger
}

func NewClient(baseURL string, logger *zap.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		circuitBreaker: NewCircuitBreaker(5, 30*time.Second),
		logger:         logger,
	}
}

func (c *Client) FetchMarkets(ctx context.Context, cursor string) (*MarketsResponse, error) {
	var result *MarketsResponse

	start := time.Now()
	metrics.RestRequestsTotal.Inc()

	err := c.circuitBreaker.Execute(func() error {
		url := fmt.Sprintf("%s/markets?limit=100", c.baseURL)
		if cursor != "" {
			url += "&cursor=" + cursor
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			metrics.RestErrorsTotal.Inc()
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			metrics.RestErrorsTotal.Inc()
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var marketsResp MarketsResponse
		if err := json.NewDecoder(resp.Body).Decode(&marketsResp); err != nil {
			metrics.RestErrorsTotal.Inc()
			return err
		}

		result = &marketsResp
		return nil
	})

	elapsed := time.Since(start)
	metrics.RestLatency.Observe(float64(elapsed.Milliseconds()))

	return result, err
}

func (c *Client) FetchAllMarkets(ctx context.Context, existingMarkets []catalog.Market) ([]catalog.Market, error) {
	var allMarkets []catalog.Market
	cursor := ""

	for {
		resp, err := c.FetchMarkets(ctx, cursor)
		if err != nil {
			return allMarkets, err
		}

		for _, m := range resp.Markets {
			if !m.Active {
				continue
			}
			market := convertMarket(m)
			allMarkets = append(allMarkets, market)
		}

		if resp.NextCursor == "" {
			break
		}
		cursor = resp.NextCursor
	}

	return allMarkets, nil
}

func (c *Client) FetchActiveMarkets(ctx context.Context) ([]catalog.Market, error) {
	return c.FetchAllMarkets(ctx, nil)
}

func (c *Client) FetchMarketsByIDs(ctx context.Context, ids []string) ([]catalog.Market, error) {
	var result []catalog.Market

	start := time.Now()
	metrics.RestRequestsTotal.Inc()

	err := c.circuitBreaker.Execute(func() error {
		reqURL := fmt.Sprintf("%s/markets?limit=100", c.baseURL)
		for _, id := range ids {
			reqURL += "&id=" + id
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			metrics.RestErrorsTotal.Inc()
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			metrics.RestErrorsTotal.Inc()
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var marketsResp MarketsResponse
		if err := json.NewDecoder(resp.Body).Decode(&marketsResp); err != nil {
			metrics.RestErrorsTotal.Inc()
			return err
		}

		for _, m := range marketsResp.Markets {
			if !m.Active {
				continue
			}
			result = append(result, convertMarket(m))
		}
		return nil
	})

	elapsed := time.Since(start)
	metrics.RestLatency.Observe(float64(elapsed.Milliseconds()))

	return result, err
}

func convertMarket(m MarketResponse) catalog.Market {
	market := catalog.Market{
		ID:       m.ConditionID,
		Slug:     m.Slug,
		IsActive: m.Active,
	}

	for _, t := range m.Tokens {
		price := parseDecimal(t.Price)
		switch t.Outcome {
		case "Yes":
			market.YESPrice = price
		case "No":
			market.NOPrice = price
		}
	}

	one := decimal.NewFromInt(1)
	market.Spread = one.Sub(market.YESPrice).Sub(market.NOPrice).Abs()
	market.Volume24h = parseDecimal(m.Volume24h)
	market.LiquidityDepth = parseDecimal(m.Liquidity)

	return market
}

func parseDecimal(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}
