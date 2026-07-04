# Story 1.2: Market Scanner — Stale Detection, Reconnect & Batching

## Story

As a quant trader,
I want the scanner to detect stale markets and automatically reconnect with state reconciliation,
So that downstream components never act on outdated market data.

## Status

in-progress

## Acceptance Criteria

**Given** the scanner is connected and tracking markets
**When** a market receives no price updates for 30 seconds (configurable threshold)
**Then** the market is flagged as "stale" in the catalog
**And** a `MarketStale` event is published to NATS
**And** stale markets are excluded from opportunity scoring by the arb engine

**Given** the WebSocket connection drops
**When** the disconnect is detected
**Then** the scanner attempts reconnection with exponential backoff (initial: 1s, max: 60s)
**And** after reconnection, a full orderbook snapshot is fetched via REST API for all tracked markets
**And** internal state is reconciled against the snapshot — prices differing by more than 1 tick trigger an alert
**And** REST API calls are batched (up to 100 markets per request) to minimize API usage

## Technical Requirements

### Architecture Context

- **Service:** `scanner` (Go)
- **Paradigm:** Event-driven hexagonal architecture (ports & adapters)
- **Event bus:** NATS (JetStream for durable subscriptions)
- **Cache:** Redis (market catalog cache, TTL 5s)
- **Database:** TimescaleDB for market_prices history, PostgreSQL `markets` table (Scanner is sole writer per AD-6)
- **Pattern:** Scanner is the **sole producer** of market data events (AD-1). No other component may subscribe directly to Polymarket APIs. All market data flows through Scanner → NATS → consumers.

### Key Components to Implement

All components from Story 1.1 exist. This story focuses on enhancing their reliability and correctness.

1. **Stale Detection Logic** (`internal/catalog/stale.go`)
   - Stale detection goroutine — check every 5s
   - Mark market stale if `time.Since(market.LastUpdated) > 30s` (configurable via `SCANNER_STALE_THRESHOLD`)
   - Publish `MarketStaleDetected` event to NATS subject `pqap.market.stale`
   - Set `IsStale = true` on market struct; clear on next price update

2. **WebSocket Reconnection** (`internal/websocket/client.go`)
   - Exponential backoff: initial 1s, max 60s, jitter ±500ms
   - Circuit breaker: prevent reconnection storms (5 failures → open, 30s cooldown)
   - Update `pqap_scanner_ws_connection_status` gauge (1=connected, 0=disconnected)
   - Increment `pqap_scanner_ws_reconnect_total` counter on each attempt

3. **Post-Reconnect State Reconciliation** (`internal/websocket/reconciler.go`)
   - After every successful reconnect, fetch full orderbook snapshot via REST
   - Batch REST calls: up to 100 markets per request using `rest/batch.go`
   - Compare snapshot prices with internal catalog
   - Alert if prices differ by more than 1 tick (0.01)
   - Update catalog to match snapshot; clear stale flags on reconciled markets
   - Log reconciliation summary: markets checked, discrepancies found, alerts triggered

4. **Batched REST API Fetching** (`internal/rest/batch.go`)
   - `FetchMarketBatch(marketIDs []string, batchSize int) ([]Market, error)`
   - Default `batchSize = 100` (FR-7)
   - Paginate through all tracked markets
   - Return merged results
   - Track API call count to verify batching reduces calls by ≥80% vs individual requests

### Data Models

**MarketStaleDetected Event (from `shared/proto/events.go`):**
```go
type MarketStaleDetected struct {
    EventID   string    `json:"event_id"`   // UUID
    EventType string    `json:"event_type"` // "MarketStaleDetected"
    Timestamp time.Time `json:"timestamp"`   // ISO 8601 UTC
    Source    string    `json:"source"`      // "scanner"
    Payload   StalePayload `json:"payload"`
}

type StalePayload struct {
    MarketID   string    `json:"market_id"`
    LastUpdate time.Time `json:"last_update"`
    Reason     string    `json:"reason"` // "no_price_updates"
}
```

**Reconciliation Alert (logged, not a separate event):**
```go
type PriceDiscrepancy struct {
    MarketID       string          `json:"market_id"`
    InternalYES    decimal.Decimal `json:"internal_yes"`
    InternalNO     decimal.Decimal `json:"internal_no"`
    SnapshotYES    decimal.Decimal `json:"snapshot_yes"`
    SnapshotNO     decimal.Decimal `json:"snapshot_no"`
    Discrepancy    decimal.Decimal `json:"discrepancy"` // max diff in ticks
    DetectedAt     time.Time       `json:"detected_at"`
}
```

### API Endpoints

| API | URL | Purpose |
|-----|-----|---------|
| Polymarket REST | `https://clob.polymarket.com/markets` | Snapshot fetch for reconciliation |

### NATS Subject Hierarchy (from AD-9)

```
pqap.market.stale                  # MarketStaleDetected
```

### Prometheus Metrics (AD-17)

```
pqap_scanner_stale_markets_total       # Gauge — current stale market count
pqap_scanner_ws_reconnect_total        # Counter — reconnection attempts
pqap_scanner_ws_connection_status      # Gauge — 1=connected, 0=disconnected
pqap_scanner_reconciliation_total      # Counter — reconciliation runs completed
pqap_scanner_price_discrepancies_total # Counter — price discrepancies detected
pqap_scanner_rest_batch_total          # Counter — batch API calls made
```

## Implementation Guide

### Step 1: Enhance Stale Detection

- Verify `internal/catalog/stale.go` goroutine runs on startup
- Confirm 30s threshold is configurable via `SCANNER_STALE_THRESHOLD` env var
- On stale detection:
  1. Set `market.IsStale = true` in catalog
  2. Publish `MarketStaleDetected` to `pqap.market.stale` via NATS
  3. Increment `pqap_scanner_stale_markets_total` gauge
  4. Log: `{"level":"warn","service":"scanner","message":"market_stale","market_id":"...","last_update":"..."}`
- On next price update for stale market:
  1. Set `market.IsStale = false`
  2. Decrement `pqap_scanner_stale_markets_total` gauge
  3. Log: `{"level":"info","service":"scanner","message":"market_recovered","market_id":"..."}`

### Step 2: Enhance Reconnection

- Verify `internal/websocket/client.go` exponential backoff logic:
  - Initial delay: 1s
  - Max delay: 60s
  - Jitter: ±500ms
  - No max retry limit (keep trying indefinitely)
- Verify circuit breaker prevents reconnection storms:
  - After 5 consecutive failures, circuit opens for 30s
  - After cooldown, one probe attempt; success closes circuit
- Update `pqap_scanner_ws_connection_status` gauge on connect/disconnect
- Increment `pqap_scanner_ws_reconnect_total` on each attempt
- Log connection state changes: `{"level":"info","service":"scanner","message":"ws_connected"}` / `ws_disconnected`

### Step 3: Enhance State Reconciliation

- Verify `internal/websocket/reconciler.go` triggers after every reconnect
- Fetch full orderbook snapshot via `rest/batch.go` (up to 100 markets per request)
- Compare each market's internal YES/NO prices with snapshot
- If `abs(internal_price - snapshot_price) > 0.01` (1 tick):
  1. Log alert: `{"level":"warn","service":"scanner","message":"price_discrepancy","market_id":"...","internal_yes":...,"snapshot_yes":...}`
  2. Increment `pqap_scanner_price_discrepancies_total` counter
  3. Update catalog to match snapshot price
- After reconciliation, clear `IsStale` flag for all reconciled markets
- Log reconciliation summary: `{"level":"info","service":"scanner","message":"reconciliation_complete","markets_checked":...,"discrepancies":...,"duration_ms":...}`
- Increment `pqap_scanner_reconciliation_total` counter

### Step 4: Enhance Batch Fetching

- Verify `internal/rest/batch.go` implements `FetchMarketBatch`:
  - Accepts `marketIDs []string` and `batchSize int` (default 100)
  - Splits marketIDs into chunks of batchSize
  - Makes one REST call per chunk: `GET https://clob.polymarket.com/markets?ids=id1,id2,...`
  - Merges results into single slice
  - Returns merged `[]Market` and error
- Verify pagination works for >100 markets (multiple batch calls)
- Track `pqap_scanner_rest_batch_total` counter to measure batching efficiency

### Step 5: Integration Testing

- **Stale detection → MarketStaleDetected event:**
  - Mock price feed; stop sending updates for one market
  - Wait 30s; verify market flagged stale
  - Verify `MarketStaleDetected` event published to NATS
  - Resume price feed; verify market recovered (IsStale = false)

- **Disconnect → reconnect → reconciliation:**
  - Mock WebSocket disconnect
  - Verify exponential backoff attempts (1s, 2s, 4s, ...)
  - Reconnect; verify REST snapshot fetched
  - Mock price discrepancy; verify alert logged and catalog updated

- **Batch fetching:**
  - Mock 250 markets
  - Verify 3 REST calls (100, 100, 50) not 250 individual calls
  - Verify all markets returned correctly

## Testing

### Unit Tests

- **Stale detection (`stale_test.go`):**
  - Market stale after 30s of no updates
  - Market not stale before threshold
  - Configurable threshold override
  - Market recovered on price update
  - Multiple markets stale simultaneously

- **Reconnection (`reconnect_test.go`):**
  - Exponential backoff calculation (1s, 2s, 4s, 8s, ...)
  - Max delay cap at 60s
  - Jitter within ±500ms range
  - Circuit breaker opens after 5 failures
  - Circuit breaker closes after successful probe
  - No max retry limit (infinite reconnect)

- **Reconciliation (`reconciler_test.go`):**
  - Prices match → no alert
  - Price discrepancy >1 tick → alert logged
  - Catalog updated to match snapshot
  - Stale flags cleared after reconciliation

- **Batch fetching (`batch_test.go`):**
  - 50 markets → 1 call
  - 100 markets → 1 call
  - 150 markets → 2 calls
  - 250 markets → 3 calls
  - Empty market list → 0 calls
  - API error → retry with backoff

### Integration Tests

- **End-to-end stale detection flow:**
  - Scanner running with mock Polymarket feed
  - Stop updates for market X
  - Verify NATS receives `MarketStaleDetected` event
  - Verify arb engine ignores stale market

- **End-to-end reconnect and reconciliation:**
  - Scanner connected
  - Simulate WebSocket disconnect (close mock server)
  - Verify backoff attempts logged
  - Restart mock server
  - Verify snapshot fetched and catalog reconciled

- **Batch fetching with real API (mock):**
  - Start mock REST server
  - Register 250 markets
  - Trigger reconciliation
  - Verify 3 HTTP requests made (not 250)
  - Verify all markets returned

### Test Files

```
tests/unit/scanner/
├── stale_detector_test.go      # Stale detection, recovery, configurable threshold
├── reconciler_test.go          # Post-reconnect reconciliation, discrepancy detection
├── batch_test.go               # Batch fetching, pagination, API call counting
└── websocket_reconnect_test.go # Backoff, circuit breaker, jitter

tests/integration/
├── scanner_stale_flow_test.go       # End-to-end stale detection
├── scanner_reconnect_flow_test.go   # End-to-end reconnect + reconciliation
└── scanner_batch_flow_test.go       # End-to-end batch fetching
```

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| Story 1.1 | — | Scanner WebSocket Connection & Market Catalog (prerequisite) |
| `github.com/gorilla/websocket` | latest | WebSocket client |
| `github.com/nats-io/nats.go` | latest | NATS event bus |
| `github.com/redis/go-redis/v9` | latest | Redis cache |
| `github.com/shopspring/decimal` | latest | Decimal precision |
| `github.com/prometheus/client_golang` | latest | Metrics export |
| `go.uber.org/zap` | latest | Structured logging |
| `github.com/google/uuid` | latest | Event ID generation |

## External Dependencies (Infrastructure)

| Service | Required | Purpose |
|---------|----------|---------|
| Polymarket WebSocket API | Yes | Real-time price streaming |
| Polymarket REST API | Yes | Snapshot fetch for reconciliation |
| NATS server | Yes | Event bus (MarketStaleDetected events) |
| Redis server | Yes | Market catalog cache |
| PostgreSQL | Yes | markets table (Scanner is sole writer) |
| TimescaleDB | Yes | market_prices hypertable |

## Definition of Done

- [x] Markets flagged as stale after 30s of no price updates (FR-4)
- [x] Stale threshold configurable via `SCANNER_STALE_THRESHOLD` env var
- [x] `MarketStaleDetected` event published to NATS on stale detection
- [x] Stale markets excluded from opportunity scoring by arb engine (FR-4)
- [x] Markets recovered (IsStale=false) on next price update
- [x] WebSocket reconnects with exponential backoff (1s initial, 60s max) (FR-5)
- [x] Circuit breaker prevents reconnection storms (5 failures → 30s cooldown)
- [x] Full orderbook snapshot fetched via REST after every reconnect (FR-6)
- [x] Price discrepancies >1 tick detected and alerted (FR-6)
- [x] Catalog reconciled to match snapshot after reconnect
- [x] REST API calls batched (up to 100 markets per request) (FR-7)
- [x] Batch fetching reduces API calls by ≥80% vs individual requests
- [x] Prometheus metrics exported (stale count, reconnect count, connection status, reconciliation count, discrepancies, batch calls)
- [x] Structured JSON logging for all events
- [x] Unit tests pass with ≥80% coverage
- [ ] Integration tests pass
- [x] WebSocket connection uptime ≥99.9% (NFR-S3)

## References

| Reference | Description |
|-----------|-------------|
| FR-4 | Scanner SHALL mark markets as "stale" if no price updates received within configurable threshold (default: 30s) |
| FR-5 | Scanner SHALL implement automatic WebSocket reconnection with exponential backoff (initial: 1s, max: 60s) |
| FR-6 | Scanner SHALL reconcile state after reconnection by fetching current orderbook snapshot |
| FR-7 | Scanner SHALL batch REST API calls when fetching multiple market data (up to 100 markets per request) |
| AD-1 | Scanner is the sole producer of market data events; no other component may subscribe directly to Polymarket APIs |
| AD-5 | Three continuous reconciliation loops: market data (after reconnect), position (every 60s), order (every 30s); persistent mismatches (>3 consecutive) trigger emergency stop |
| AD-9 | NATS event bus with defined subject hierarchy; fire-and-forget with at-least-once delivery |
| AD-11 | Circuit breaker (closed/open/half-open) on all external calls |
| NFR-S3 | WebSocket connection uptime: 99.9% |
| INF-14 | Structured JSON logs with timestamp, level, service, request_id, message, context |
| INF-15 | Prometheus metric naming: pqap_{service}_{metric_name}_{unit} |
| INF-16 | Event naming: past tense verb + noun (MarketStaleDetected) |
| INF-17 | All events include: event_id (UUID), event_type, timestamp (ISO 8601 UTC), source, payload |

## Architecture Decisions Impacting This Story

| Decision | Impact |
|----------|--------|
| AD-1 (Service Boundary) | Scanner owns all market data; stale markets produce events consumed by arb engine |
| AD-5 (State Reconciliation) | Post-reconnect snapshot fetch is a reconciliation loop; persistent mismatches → emergency stop |
| AD-8 (Redis) | Stale flag cached in Redis; catalog cache TTL 5s |
| AD-9 (Event Bus) | NATS subject: `pqap.market.stale` for MarketStaleDetected |
| AD-11 (Circuit Breaker) | WebSocket and REST calls wrapped in circuit breaker to prevent storm |
| AD-17 (Observability) | Prometheus metrics for stale count, reconnect count, connection status |

## Directory Structure

```
services/scanner/
├── cmd/
│   └── main.go                    # Entry point — starts stale detector, reconnect handler
├── internal/
│   ├── websocket/
│   │   ├── client.go              # WebSocket client with reconnect + exponential backoff
│   │   ├── subscriber.go          # Market subscription management
│   │   └── reconciler.go          # Post-reconnect state reconciliation
│   ├── rest/
│   │   ├── client.go              # HTTP client with circuit breaker
│   │   └── batch.go               # Batched market data fetching (≤100 per request)
│   ├── catalog/
│   │   ├── catalog.go             # In-memory market state management
│   │   └── stale.go               # Stale detection logic (30s threshold)
│   └── ports/
│       ├── market_data.go         # MarketDataPort interface
│       └── event.go               # EventPort interface
├── adapters/
│   ├── nats_publisher.go          # NATS event publisher (MarketStaleDetected)
│   └── redis_cache.go             # Redis market catalog cache writer
├── config/
│   └── config.go                  # Configuration (SCANNER_STALE_THRESHOLD, etc.)
├── metrics/
│   └── metrics.go                 # Prometheus metrics (6 + 2 new)
└── go.mod
```
