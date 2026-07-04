package catalog

import (
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

type Market struct {
	ID             string          `json:"id"`
	Slug           string          `json:"slug"`
	YESPrice       decimal.Decimal `json:"yes_price"`
	NOPrice        decimal.Decimal `json:"no_price"`
	Spread         decimal.Decimal `json:"spread"`
	Volume24h      decimal.Decimal `json:"volume_24h"`
	LiquidityDepth decimal.Decimal `json:"liquidity_depth"`
	IsActive       bool            `json:"is_active"`
	IsStale        bool            `json:"is_stale"`
	LastUpdated    time.Time       `json:"last_updated"`
}

type Catalog struct {
	mu       sync.RWMutex
	markets  map[string]*Market
	onChange func(market Market)
}

func NewCatalog(onChange func(market Market)) *Catalog {
	return &Catalog{
		markets:  make(map[string]*Market),
		onChange: onChange,
	}
}

func (c *Catalog) Add(market Market) bool {
	if market.ID == "" {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.markets[market.ID]; exists {
		return false
	}

	if market.LastUpdated.IsZero() {
		market.LastUpdated = time.Now()
	}
	market.IsActive = true
	c.markets[market.ID] = &market
	return true
}

func (c *Catalog) Update(market Market) bool {
	if market.ID == "" {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	existing, exists := c.markets[market.ID]
	if !exists {
		return false
	}

	priceChanged := !existing.YESPrice.Equal(market.YESPrice) || !existing.NOPrice.Equal(market.NOPrice)

	existing.YESPrice = market.YESPrice
	existing.NOPrice = market.NOPrice
	existing.Spread = market.Spread
	existing.Volume24h = market.Volume24h
	existing.LiquidityDepth = market.LiquidityDepth
	existing.LastUpdated = time.Now()
	existing.IsStale = false

	if market.Slug != "" {
		existing.Slug = market.Slug
	}

	if priceChanged && c.onChange != nil {
		c.onChange(*existing)
	}

	return priceChanged
}

// Upsert atomically inserts or updates a market, eliminating TOCTOU races.
// Returns (existed, wasStale) so callers can detect recovery without a separate Get.
func (c *Catalog) Upsert(market Market) (existed, wasStale bool) {
	if market.ID == "" {
		return false, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	existing, found := c.markets[market.ID]
	if !found {
		if market.LastUpdated.IsZero() {
			market.LastUpdated = time.Now()
		}
		market.IsActive = true
		c.markets[market.ID] = &market
		return false, false
	}

	wasStale = existing.IsStale
	priceChanged := !existing.YESPrice.Equal(market.YESPrice) || !existing.NOPrice.Equal(market.NOPrice)

	existing.YESPrice = market.YESPrice
	existing.NOPrice = market.NOPrice
	existing.Spread = market.Spread
	existing.Volume24h = market.Volume24h
	existing.LiquidityDepth = market.LiquidityDepth
	existing.LastUpdated = time.Now()
	existing.IsStale = false

	if market.Slug != "" {
		existing.Slug = market.Slug
	}

	if priceChanged && c.onChange != nil {
		c.onChange(*existing)
	}

	return true, wasStale
}

func (c *Catalog) Get(marketID string) (Market, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	m, exists := c.markets[marketID]
	if !exists {
		return Market{}, false
	}
	return *m, true
}

func (c *Catalog) List() []Market {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]Market, 0, len(c.markets))
	for _, m := range c.markets {
		if m.IsActive {
			result = append(result, *m)
		}
	}
	return result
}

func (c *Catalog) MarkStale(marketID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	m, exists := c.markets[marketID]
	if !exists {
		return false
	}
	if m.IsStale {
		return false
	}
	m.IsStale = true
	return true
}

func (c *Catalog) ClearStale(marketID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	m, exists := c.markets[marketID]
	if !exists {
		return false
	}
	if !m.IsStale {
		return false
	}
	m.IsStale = false
	return true
}

func (c *Catalog) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.markets)
}

func (c *Catalog) StaleCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	count := 0
	for _, m := range c.markets {
		if m.IsStale {
			count++
		}
	}
	return count
}
