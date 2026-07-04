package executor

import (
	"sync"
	"time"
)

type IdempotencyChecker struct {
	mu    sync.RWMutex
	cache map[string]time.Time
	ttl   time.Duration
}

func NewIdempotencyChecker() *IdempotencyChecker {
	ic := &IdempotencyChecker{
		cache: make(map[string]time.Time),
		ttl:   5 * time.Minute,
	}

	go ic.cleanup()
	return ic
}

func (ic *IdempotencyChecker) IsDuplicate(clientOrderID string) bool {
	ic.mu.RLock()
	_, exists := ic.cache[clientOrderID]
	ic.mu.RUnlock()

	return exists
}

func (ic *IdempotencyChecker) Mark(clientOrderID string) {
	ic.mu.Lock()
	ic.cache[clientOrderID] = time.Now().Add(ic.ttl)
	ic.mu.Unlock()
}

func (ic *IdempotencyChecker) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ic.mu.Lock()
		now := time.Now()
		for k, v := range ic.cache {
			if now.After(v) {
				delete(ic.cache, k)
			}
		}
		ic.mu.Unlock()
	}
}
