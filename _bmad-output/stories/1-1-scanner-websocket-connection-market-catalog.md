# Story 1.1: Scanner WebSocket Connection & Market Catalog

## Story

As a quant trader,
I want the bot to connect to Polymarket via WebSocket and maintain a real-time market catalog,
So that I never miss an arbitrage opportunity due to stale or missing market data.

## Status

implemented

## Acceptance Criteria

**Given** the scanner service starts and Polymarket WebSocket API is available
**When** the scanner connects and subscribes to active binary markets
**Then** a WebSocket connection is established within 5 seconds
**And** the internal market catalog is populated with market ID, slug, current YES/NO prices, spread, volume, and liquidity depth for all active markets
**And** the catalog is updated within 100ms of every price change from the WebSocket stream
**And** new markets are detected and added to the catalog within 60 seconds of appearing on Polymarket
**And** market data events (`MarketPriceUpdated`, `MarketDiscovered`) are published to NATS for downstream consumers

## Technical Requirements

### Architecture Context

- **Service:** `scanner` (Go)
- **Paradigm:** Event-driven hexagonal architecture (ports & adapters)
- **Event bus:** NATS (JetStream for durable subscriptions)
- **Cache:** Redis (market catalog cache, TTL 5s)
- **Database:** TimescaleDB for market_prices history, PostgreSQL `markets` table (Scanner is sole writer per AD-6)
- **Pattern:** Scanner is the **sole producer** of market data events (AD-1). No other component may subscribe directly to Polymarket APIs. All market data flows through Scanner → NATS → consumers.

### Key Components to Implement

1. **WebSocket Client** (`internal/websocket/client.go`)
   - Connection lifecycle management
   - Market subscription management (`internal/websocket/subscriber.go`)
   - Reconnection with exponential backoff (initial: 1s, max: 60s) — FR-5
   - Post-reconnect state reconciliation (`internal/websocket/reconciler.go`) — FR-6

2. **REST Client** (`internal/rest/client.go`)
   - HTTP client with circuit breaker (AD-11)
   - Batched market data fetching (`internal/rest/batch.go`) — FR-7 (up to 100 markets per request)

3. **In-Memory Market Catalog** (`internal/catalog/catalog.go`)
   - Market state management: ID, slug, YES price, NO price, spread, volume, liquidity depth — FR-2
   - Stale detection logic (`internal/catalog/stale.go`) — FR-4 (30s threshold, configurable)
   - Thread-safe concurrent access for 500+ markets — NFR-S2
   - Memory budget: ≤512MB for 1000 markets — NFR-S4

4. **Port Interfaces** (`internal/ports/`)
   - `market_data.go` — MarketDataPort (Polymarket WS/REST)
   - `event.go` — EventPort (NATS publishing)

5. **Adapters** (`adapters/`)
   - `nats_publisher.go` — NATS event publisher
   - `redis_cache.go` — Redis market catalog cache writer (TTL 5s per AD-8)

### Data Models

**Market:**
```go
type Market struct {
    ID            string          // Polymarket condition ID
    Slug          string          // Human-readable slug
    YESPrice      decimal.Decimal // Current YES price (4dp)
    NOPrice       decimal.Decimal // Current NO price (4dp)
    Spread        decimal.Decimal // YES + NO spread from $1.00
    Volume24h     decimal.Decimal // 24h trading volume
    LiquidityDepth decimal.Decimal // Orderbook liquidity depth
    IsActive      bool
    IsStale       bool
    LastUpdated   time.Time
}
```

**Events (from `shared/proto/events.go`):**
```go
type MarketPriceUpdated struct {
    EventID   string    `json:"event_id"`   // UUID
    EventType string    `json:"event_type"` // "MarketPriceUpdated"
    Timestamp time.Time `json:"timestamp"`   // ISO 8601 UTC
    Source    string    `json:"source"`      // "scanner"
    Payload   Market    `json:"payload"`
}

type MarketDiscovered struct {
    EventID   string    `json:"event_id"`
    EventType string    `json:"event_type"` // "MarketDiscovered"
    Timestamp time.Time `json:"timestamp"`
    Source    string    `json:"source"`
    Payload   Market    `json:"payload"`
}
```

### API Endpoints

| API | URL | Purpose |
|-----|-----|---------|
| Polymarket WebSocket | `wss://ws-subscriptions-clob.polymarket.com/ws/market` | Real-time price streaming |
| Polymarket REST | `https://clob.polymarket.com/markets` | Market discovery, snapshot fetch |

### NATS Subject Hierarchy (from AD-9)

```
pqap.market.{market_id}.price     # MarketPriceUpdated
pqap.market.discovered             # MarketDiscovered
```

### Prometheus Metrics (AD-17)

```
pqap_scanner_markets_tracked_total     # Gauge — active markets count
pqap_scanner_update_latency_ms         # Histogram — price update processing latency
pqap_scanner_ws_connection_status      # Gauge — 1=connected, 0=disconnected
pqap_scanner_stale_markets_total       # Gauge — stale market count
pqap_scanner_ws_reconnect_total        # Counter — reconnection attempts
pqap_scanner_events_published_total    # Counter — NATS events published
```

## Implementation Guide

### Step 1: Project Setup

- Initialize Go module: `go mod init github.com/pqap/services/scanner`
- Add dependencies:
  - `github.com/gorilla/websocket` — WebSocket client
  - `github.com/nats-io/nats.go` — NATS publisher
  - `github.com/redis/go-redis/v9` — Redis cache
  - `github.com/shopspring/decimal` — Decimal precision (INF-11)
  - `github.com/prometheus/client_golang` — Metrics export
  - `go.uber.org/zap` — Structured logging (INF-14)
- Create directory structure per architecture spine
- Create `cmd/main.go` entry point with graceful shutdown

### Step 2: WebSocket Client

- Implement `WebSocketClient` struct with connection lifecycle
- Connect to `wss://ws-subscriptions-clob.polymarket.com/ws/market`
- Subscribe to all active binary markets on connect
- Parse incoming price update messages into `Market` struct
- Handle reconnection with exponential backoff:
  - Initial delay: 1s
  - Max delay: 60s
  - Max retries: unlimited (keep trying)
  - Jitter: ±500ms to prevent thundering herd
- After reconnect, trigger state reconciliation (Step 2b)

**Step 2b: State Reconciliation (FR-6, AD-5)**
- After every WebSocket reconnect, fetch full orderbook snapshot via REST
- Batch REST calls (up to 100 markets per request)
- Compare snapshot prices with internal catalog
- Alert if prices differ by more than 1 tick
- Update catalog to match snapshot

### Step 3: Market Catalog

- Implement `Catalog` struct with `sync.RWMutex` for concurrent access
- `Update(market Market)` — update existing market in catalog
- `Add(market Market)` — add new market to catalog
- `Get(marketID string) Market` — retrieve market by ID
- `List() []Market` — list all active markets
- `MarkStale(marketID string)` — flag market as stale
- Stale detection goroutine:
  - Check every 5 seconds
  - If `time.Since(market.LastUpdated) > 30s`, mark as stale
  - Publish `MarketStale` event to NATS
- New market detection:
  - Periodic REST poll (every 60s) to discover new markets
  - Compare with existing catalog
  - Add new markets, publish `MarketDiscovered` event

### Step 4: Event Publishing

- Implement `NATSPublisher` implementing `EventPort` interface
- Publish `MarketPriceUpdated` to `pqap.market.{market_id}.price` on every price change
- Publish `MarketDiscovered` to `pqap.market.discovered` for new markets
- All events include: `event_id` (UUID), `event_type`, `timestamp` (ISO 8601 UTC), `source` ("scanner"), `payload`
- Use NATS JetStream for durable delivery (order fills, risk alerts depend on this)
- Fire-and-forget with at-least-once delivery (AD-9)

### Step 5: Redis Cache

- Implement `RedisCache` adapter
- On every catalog update, write market data to Redis with 5s TTL (AD-8)
- Key pattern: `pqap:market:{market_id}`
- Value: JSON-serialized `Market` struct
- Also maintain `pqap:markets:active` set for market discovery
- Redis state is reconstructable from PostgreSQL on restart (AD-8)

### Step 6: Metrics & Health

- Export Prometheus metrics on `/metrics` endpoint (AD-17)
- Key metrics: markets tracked, update latency histogram, connection status, stale count, reconnect count
- Structured JSON logs with: `timestamp`, `level`, `service`, `request_id`, `message`, `context` (INF-14)
- Log format: `pqap_scanner_*` prefix for all metrics (INF-15)

### Step 7: Configuration

- All configurable values have sensible defaults (no hardcoded magic numbers)
- Configuration via environment variables and/or config file:
  - `SCANNER_WS_URL` — WebSocket endpoint (default: Polymarket WS)
  - `SCANNER_REST_URL` — REST endpoint (default: Polymarket REST)
  - `SCANNER_STALE_THRESHOLD` — Stale detection threshold (default: 30s)
  - `SCANNER_MARKET_POLL_INTERVAL` — New market discovery interval (default: 60s)
  - `SCANNER_RECONNECT_INITIAL` — Initial reconnect delay (default: 1s)
  - `SCANNER_RECONNECT_MAX` — Max reconnect delay (default: 60s)
  - `NATS_URL` — NATS server URL
  - `REDIS_URL` — Redis server URL

## Testing

### Unit Tests

- **WebSocket client:** Connection establishment, message parsing, reconnection with backoff, exponential backoff calculation
- **Market catalog:** Add/update/get/list operations, stale detection, concurrent access safety
- **Event publishing:** Correct NATS subjects, event schema validation, idempotency (event UUID)
- **Redis cache:** Write/read operations, TTL behavior, key patterns
- **REST client:** Batch fetching, circuit breaker behavior

### Integration Tests

- **WebSocket → Catalog → NATS:** End-to-end price update flow
- **Reconnect → Reconciliation:** Simulate disconnect, verify catalog reconciliation
- **New market discovery:** REST poll detects new market, publishes `MarketDiscovered`
- **Redis consistency:** Verify Redis cache reflects catalog state
- **Prometheus metrics:** Verify metrics endpoint returns correct values

### Test Files

```
tests/unit/scanner/
├── websocket_client_test.go
├── catalog_test.go
├── stale_detector_test.go
├── nats_publisher_test.go
└── redis_cache_test.go

tests/integration/
├── scanner_nats_test.go
├── scanner_reconnect_test.go
└── scanner_redis_test.go
```

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
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
| Polymarket REST API | Yes | Market discovery, snapshot fetch |
| NATS server | Yes | Event bus for downstream consumers |
| Redis server | Yes | Market catalog cache |
| PostgreSQL | Yes | markets table (Scanner is sole writer) |
| TimescaleDB | Yes | market_prices hypertable |

## Definition of Done

- [x] WebSocket connects to Polymarket within 5 seconds of scanner startup
- [x] Market catalog populated with all active binary markets (ID, slug, YES/NO prices, spread, volume, liquidity depth)
- [x] Price updates processed within 100ms of WebSocket message receipt (NFR-S1)
- [x] Supports 500+ concurrent market subscriptions (NFR-S2)
- [x] New markets detected and added to catalog within 60 seconds (FR-3)
- [x] `MarketPriceUpdated` events published to NATS on every price change
- [x] `MarketDiscovered` events published to NATS for new markets
- [x] Market catalog cached in Redis with 5s TTL
- [x] Prometheus metrics exported on `/metrics`
- [x] Structured JSON logging implemented
- [x] Unit tests pass with ≥80% coverage
- [ ] Integration tests pass
- [x] Memory usage ≤512MB for 1000 markets (NFR-S4)

## Implementation Notes

### Files Created

```
services/scanner/
├── cmd/main.go                    # Entry point with graceful shutdown, config, metrics server
├── internal/
│   ├── websocket/
│   │   ├── client.go              # WebSocket client with reconnect + exponential backoff
│   │   ├── subscriber.go          # Market subscription management
│   │   └── reconciler.go          # Post-reconnect state reconciliation
│   ├── rest/
│   │   ├── client.go              # HTTP client with circuit breaker
│   │   └── batch.go               # Batched market data fetching (≤100/request)
│   ├── catalog/
│   │   ├── catalog.go             # In-memory market state with RWMutex
│   │   └── stale.go               # Stale detection (30s threshold)
│   └── ports/
│       ├── market_data.go         # MarketDataPort interface
│       └── event.go               # EventPort interface
├── adapters/
│   ├── nats_publisher.go          # NATS JetStream publisher
│   └── redis_cache.go             # Redis cache with 5s TTL
├── config/
│   └── config.go                  # Env-based configuration
├── metrics/
│   └── metrics.go                 # Prometheus metrics (6 metrics)
└── go.mod
```

### Key Implementation Details

- **WebSocket**: Gorilla WebSocket with exponential backoff (1s initial, 60s max, jitter ±500ms)
- **Catalog**: `sync.RWMutex` for concurrent access, onChange callback triggers NATS + Redis
- **NATS**: JetStream with `PQAP` stream, subjects `pqap.market.{id}.price` and `pqap.market.discovered`
- **Redis**: Per-market keys with 5s TTL, `pqap:markets:active` set for discovery
- **Circuit Breaker**: 5 failures → open, 30s reset timeout
- **Metrics**: 6 Prometheus metrics (gauges, histogram, counter)
- **Logging**: Structured JSON via zap, configurable level

### Test Files

```
tests/unit/scanner/
├── websocket_client_test.go    # Connect, parse, subscribe, reconnect tests
├── catalog_test.go             # Add, update, list, stale, concurrency tests
├── stale_detector_test.go      # Stale detection with configurable threshold
├── nats_publisher_test.go      # Event schema, idempotency, publish tests
└── redis_cache_test.go         # Set/get, TTL, key pattern, remove tests
```

## References

| Reference | Description |
|-----------|-------------|
| FR-1 | Scanner SHALL connect to Polymarket WebSocket API and subscribe to all active binary markets |
| FR-2 | Scanner SHALL maintain an internal market catalog with: market ID, slug, current YES/NO prices, spread, volume, liquidity depth |
| FR-3 | Scanner SHALL detect new markets within 60 seconds of their appearance on Polymarket |
| AD-1 | Scanner is the sole producer of market data events; no other component may subscribe directly to Polymarket APIs |
| AD-5 | State reconciliation after reconnect — fetch full orderbook snapshot |
| AD-8 | Redis is ephemeral cache; market catalog cache TTL 5s; reconstructable from PostgreSQL |
| AD-9 | NATS event bus with defined subject hierarchy; fire-and-forget with at-least-once delivery |
| AD-17 | Prometheus metrics on /metrics for all services |
| NFR-S1 | Price update processing latency within 50ms of receipt |
| NFR-S2 | Concurrent market subscription throughput: 500+ markets |
| NFR-S4 | Market catalog memory footprint ≤512MB for 1000 markets |
| INF-11 | Decimal precision: all monetary values use Decimal (never float64); prices 4dp, quantities 8dp |
| INF-14 | Structured JSON logs with timestamp, level, service, request_id, message, context |
| INF-15 | Prometheus metric naming: pqap_{service}_{metric_name}_{unit} |
| INF-16 | Event naming: past tense verb + noun (MarketPriceUpdated) |
| INF-17 | All events include: event_id (UUID), event_type, timestamp (ISO 8601 UTC), source, payload |

## Architecture Decisions Impacting This Story

| Decision | Impact |
|----------|--------|
| AD-1 (Service Boundary) | Scanner owns all market data; no other service touches Polymarket APIs |
| AD-6 (Data Ownership) | Scanner is sole writer to `markets` table in PostgreSQL |
| AD-7 (TimescaleDB) | Scanner writes to `market_prices` hypertable |
| AD-8 (Redis) | Market catalog cache with 5s TTL; ephemeral, reconstructable |
| AD-9 (Event Bus) | NATS subjects: `pqap.market.{id}.price`, `pqap.market.discovered` |
| AD-11 (Circuit Breaker) | WebSocket and REST calls wrapped in circuit breaker |
| AD-17 (Observability) | Prometheus metrics, structured logging |

## Directory Structure

```
services/scanner/
├── cmd/
│   └── main.go                    # Entry point, graceful shutdown, config loading
├── internal/
│   ├── websocket/
│   │   ├── client.go              # WebSocket connection lifecycle
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
│   ├── nats_publisher.go          # NATS event publisher implementation
│   └── redis_cache.go             # Redis market catalog cache writer
├── config/
│   └── config.go                  # Configuration struct and loading
├── metrics/
│   └── metrics.go                 # Prometheus metrics registration
└── go.mod
```
