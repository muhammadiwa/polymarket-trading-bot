# Story 1.4: Execution Engine — Order Placement & Slippage Protection

## Story

As a quant trader,
I want the execution engine to place limit orders with slippage protection and risk validation,
So that trades are executed at favorable prices and never exceed my risk budget.

## Status

ready-for-dev

## Acceptance Criteria

**Given** the execution engine receives an `OpportunityDetected` event
**When** the engine prepares to execute the trade
**Then** a synchronous risk check is performed against the Pit Boss (Redis GET, < 10ms)
**And** if Pit Boss returns DENY, the order is rejected and logged with the denial reason
**And** if approved, a GTC limit order is placed via Polymarket CLOB API within 200ms of the decision
**And** every order gets a unique client order ID (UUID) for idempotency — no duplicate orders under any failure scenario
**And** if the price moves beyond slippage tolerance (default: 1%) before placement, the trade is rejected
**And** partial fills are tracked: filled quantity logged, remaining quantity handled per strategy (cancel or wait)
**And** every order attempt is logged with: timestamp, market, side, price, size, result, latency
**And** on fill, an `OrderFilled` event is published; on failure, `OrderFailed` is published

## Technical Requirements

### Architecture Context

- **Service:** `execution-engine` (Go)
- **Paradigm:** Event-driven hexagonal architecture (ports & adapters)
- **Event bus:** NATS (JetStream for durable subscriptions)
- **Risk:** Pit Boss (Redis) — synchronous check before every trade (AD-4)
- **API:** Polymarket CLOB API — sole writer (AD-3)
- **Database:** PostgreSQL `trades` table (append-only, immutable) (AD-6)
- **Pattern:** Execution Engine is the **sole writer** to Polymarket CLOB API. Subscribes to `OpportunityDetected` events, performs synchronous Pit Boss risk check, places orders, monitors fills, and emits order lifecycle events.

### Key Components to Implement

1. **NATS Subscriber** (`internal/ports/event.go` + `adapters/nats_subscriber.go`)
   - Subscribe to `pqap.opportunity.detected` (OpportunityDetected events)
   - Parse opportunity payload: opportunity_id, market_id, yes_price, no_price, spread, score, fill_probability, liquidity
   - Handle connection lifecycle (reconnect on disconnect)
   - Deduplicate by event_id (at-least-once delivery, idempotent consumers per AD-9)

2. **Pit Boss Risk Check** (`internal/executor/risk_check.go` + `adapters/redis_risk.go`)
   - Synchronous Redis GET on Pit Boss risk state keys
   - Check: daily budget remaining, per-market position limit, per-strategy position limit
   - Return ALLOW/DENY with reason
   - Target latency: <10ms (NFR-R1)
   - If DENY: reject order, log rejection reason, increment `pqap_execution_risk_denied_total`

3. **Order Placement** (`internal/executor/executor.go` + `adapters/polymarket_clob.go`)
   - Calculate optimal order size and price from opportunity data
   - Generate UUID v4 `client_order_id` for idempotency (FR-22, AD-3)
   - Place GTC limit order via Polymarket CLOB API (`POST /order`)
   - Track placement latency (target: <200ms per NFR-E1)
   - On API error: retry with backoff, increment circuit breaker counter

4. **Slippage Protection** (`internal/executor/slippage.go`)
   - Record opportunity price at detection time
   - Before placing order, check current market price
   - If `|current_price - opportunity_price| / opportunity_price > slippage_tolerance`:
     - Reject order with `ErrSlippageExceeded`
     - Log rejection with price delta
   - Configurable via `EXECUTION_SLIPPAGE_TOLERANCE` env var (default: 0.01 = 1%)
   - Per-strategy override supported

5. **Idempotent Order Placement** (`internal/executor/idempotency.go`)
   - Generate UUID v4 `client_order_id` per order attempt
   - Store `client_order_id` in memory (with TTL) to detect duplicates
   - If duplicate `client_order_id` detected: reject and log (NFR-E4: zero duplicate orders)
   - `client_order_id` passed to CLOB API for server-side dedup

6. **Fill Monitor** (`internal/monitor/fill_monitor.go`)
   - Poll CLOB API for order status after placement
   - Track filled quantity from order response
   - On partial fill: log filled_qty, remaining_qty; handle per strategy (cancel or wait)
   - On full fill: publish `OrderFilled` event
   - On timeout/cancel: publish `OrderCancelled` event
   - On failure: publish `OrderFailed` event

7. **Order Audit Trail** (`internal/logger/order_logger.go`)
   - Log every order attempt to PostgreSQL `trades` table (append-only, immutable)
   - Fields: timestamp (UTC TIMESTAMPTZ), market_id, side, price, quantity, fill status, PnL, strategy_id, latency_ms, client_order_id, result, error_reason
   - All monetary values use Decimal precision (prices: 4dp, quantities: 8dp, PnL: 8dp) (INF-11)
   - `account_id` column included (nullable, default null) for future multi-account (INF-18)

8. **NATS Publisher** (`adapters/nats_publisher.go`)
   - Publish `OrderPlaced` to `pqap.order.placed` on successful order placement
   - Publish `OrderFilled` to `pqap.order.filled` on full fill
   - Publish `OrderPartialFill` to `pqap.order.partial` on partial fill
   - Publish `OrderFailed` to `pqap.order.failed` on failure
   - Event schema: `event_id` (UUID), `event_type`, `timestamp` (ISO 8601 UTC), `source` ("execution-engine"), `payload` (INF-17)

9. **Circuit Breaker** (`internal/circuit_breaker/breaker.go`)
   - Wrap CLOB API calls in circuit breaker (AD-11)
   - States: closed (normal), open (tripped), half-open (probe)
   - Trip after N consecutive API errors (default: 5) (FR-21)
   - Cooldown period: 60s (configurable)
   - On trip: halt all trading, send critical Telegram alert, require manual resume
   - Track state in Prometheus gauge

### Data Models

**Order (internal domain model):**
```go
type Order struct {
    ID              string          `json:"id"`               // UUID — internal order ID
    ClientOrderID   string          `json:"client_order_id"`  // UUID — for CLOB API idempotency
    OpportunityID   string          `json:"opportunity_id"`   // Link back to opportunity
    MarketID        string          `json:"market_id"`
    Side            string          `json:"side"`             // "BUY" or "SELL"
    Price           decimal.Decimal `json:"price"`            // 4dp
    Size            decimal.Decimal `json:"size"`             // 8dp
    FilledQty       decimal.Decimal `json:"filled_qty"`       // 8dp
    RemainingQty    decimal.Decimal `json:"remaining_qty"`    // 8dp
    Status          OrderStatus     `json:"status"`           // PENDING, PLACED, PARTIAL_FILL, FILLED, CANCELLED, FAILED
    TimeInForce     string          `json:"time_in_force"`    // GTC (default), FOK, GTD, FAK
    LatencyMs       int64           `json:"latency_ms"`       // order placement latency
    RiskCheckResult string          `json:"risk_check_result"` // "ALLOW" or "DENY" with reason
    SlippageCheck   string          `json:"slippage_check"`   // "PASS" or "FAIL" with delta
    ErrorReason     string          `json:"error_reason"`     // empty if success
    StrategyID      string          `json:"strategy_id"`
    AccountID       *string         `json:"account_id"`       // nullable, for future multi-account
    PlacedAt        time.Time       `json:"placed_at"`        // UTC TIMESTAMPTZ
    FilledAt        *time.Time      `json:"filled_at"`        // nullable
}
```

**OrderFilled Event:**
```go
type OrderFilled struct {
    EventID   string           `json:"event_id"`   // UUID
    EventType string           `json:"event_type"` // "OrderFilled"
    Timestamp time.Time        `json:"timestamp"`   // ISO 8601 UTC
    Source    string           `json:"source"`      // "execution-engine"
    Payload   OrderFilledPayload `json:"payload"`
}

type OrderFilledPayload struct {
    OrderID        string          `json:"order_id"`
    ClientOrderID  string          `json:"client_order_id"`
    OpportunityID  string          `json:"opportunity_id"`
    MarketID       string          `json:"market_id"`
    Side           string          `json:"side"`
    Price          decimal.Decimal `json:"price"`
    FilledQty      decimal.Decimal `json:"filled_qty"`
    LatencyMs      int64           `json:"latency_ms"`
    StrategyID     string          `json:"strategy_id"`
}
```

**OrderFailed Event:**
```go
type OrderFailed struct {
    EventID   string           `json:"event_id"`   // UUID
    EventType string           `json:"event_type"` // "OrderFailed"
    Timestamp time.Time        `json:"timestamp"`   // ISO 8601 UTC
    Source    string           `json:"source"`      // "execution-engine"
    Payload   OrderFailedPayload `json:"payload"`
}

type OrderFailedPayload struct {
    OrderID        string `json:"order_id"`
    ClientOrderID  string `json:"client_order_id"`
    OpportunityID  string `json:"opportunity_id"`
    MarketID       string `json:"market_id"`
    Reason         string `json:"reason"`       // "risk_denied", "slippage_exceeded", "api_error", "circuit_breaker_open"
    ErrorDetail    string `json:"error_detail"`
    StrategyID     string `json:"strategy_id"`
}
```

### API Endpoints

**Polymarket CLOB API (consumed):**
- `POST /order` — Place a new limit order
- `DELETE /order` — Cancel an existing order
- `GET /order/{order_id}` — Get order status (for fill monitoring)

**NATS Subjects (from AD-9):**
```
pqap.opportunity.detected   # Consumed: OpportunityDetected
pqap.order.placed           # Produced: OrderPlaced
pqap.order.filled           # Produced: OrderFilled
pqap.order.partial          # Produced: OrderPartialFill
pqap.order.cancelled        # Produced: OrderCancelled
pqap.order.failed           # Produced: OrderFailed
pqap.risk.emergency         # Consumed: EmergencyStop (halt all trading)
```

### Prometheus Metrics (AD-17)

```
pqap_execution_orders_placed_total         # Counter — total orders placed
pqap_execution_orders_filled_total         # Counter — total orders filled
pqap_execution_orders_failed_total         # Counter — total orders failed
pqap_execution_orders_cancelled_total      # Counter — total orders cancelled
pqap_execution_risk_denied_total           # Counter — orders denied by Pit Boss
pqap_execution_slippage_rejected_total     # Counter — orders rejected due to slippage
pqap_execution_order_latency_ms            # Histogram — order placement latency (target: <200ms)
pqap_execution_risk_check_latency_ms       # Histogram — Pit Boss risk check latency (target: <10ms)
pqap_execution_fill_latency_ms             # Histogram — time from placement to fill
pqap_execution_circuit_breaker_state       # Gauge — 0=closed, 1=open, 2=half-open
pqap_execution_circuit_breaker_trips_total # Counter — total circuit breaker trips
pqap_execution_active_orders               # Gauge — currently open orders
pqap_execution_partial_fills_total         # Counter — partial fill events
pqap_execution_duplicate_rejected_total    # Counter — duplicate client_order_id rejections
```

## Implementation Guide

### Step 1: NATS Subscriber

- Subscribe to `pqap.opportunity.detected` (JetStream durable subscription)
- Parse `OpportunityDetected` event payload
- Deduplicate by `event_id` — if already processed, skip (at-least-once delivery, idempotent per AD-9)
- On receive: record `start_time = time.Now()` for latency tracking
- Handle NATS connection lifecycle: reconnect on disconnect, update `pqap_execution_nats_connection_status` gauge
- If `EmergencyStop` event received on `pqap.risk.emergency`: halt all processing, cancel open orders

### Step 2: Pit Boss Risk Check

- Synchronous Redis GET on Pit Boss risk state keys
- Keys to check:
  - `pqap:risk:daily_budget_remaining` — if ≤ 0, DENY
  - `pqap:risk:market_position:{market_id}` — if would exceed per-market limit, DENY
  - `pqap:risk:strategy_position:{strategy_id}` — if would exceed per-strategy limit, DENY
- Return: `{allowed: bool, reason: string}`
- Target: <10ms latency (NFR-R1)
- Log risk decision to PostgreSQL `risk_events` table with full context (FR-47)
- If DENY:
  - Increment `pqap_execution_risk_denied_total`
  - Publish `OrderFailed` with reason "risk_denied"
  - Log to `trades` table
  - Skip to next opportunity

### Step 3: Slippage Check

- Record opportunity price at detection time (from `OpportunityDetected` payload)
- Fetch current market price (from Redis market catalog cache, TTL 5s)
- Calculate slippage: `delta = |current_price - opportunity_price| / opportunity_price`
- If `delta > slippage_tolerance`:
  - Reject with `ErrSlippageExceeded`
  - Increment `pqap_execution_slippage_rejected_total`
  - Publish `OrderFailed` with reason "slippage_exceeded"
  - Log to `trades` table
- Configurable: `EXECUTION_SLIPPAGE_TOLERANCE` env var (default: 0.01 = 1%)

### Step 4: Order Placement

- Calculate order parameters:
  - `price`: current market price (or opportunity price if within slippage tolerance)
  - `size`: based on strategy position sizing and available capital
  - `side`: from opportunity (BUY YES + BUY NO for arb)
- Generate UUID v4 `client_order_id`
- Check for duplicate `client_order_id` in memory cache (with 5min TTL)
- If duplicate: increment `pqap_execution_duplicate_rejected_total`, skip
- Place GTC limit order via CLOB API: `POST /order`
  - Body: `{market_id, side, price, size, client_order_id, time_in_force: "GTC"}`
  - Track latency: `latency_ms = time.Since(start_time).Milliseconds()`
- On success: record `pqap_execution_order_latency_ms` histogram
- On API error: increment circuit breaker counter, retry with exponential backoff (max 3 retries)

### Step 5: Fill Monitoring

- After order placement, poll CLOB API for order status
- Poll interval: 100ms (configurable)
- Poll timeout: 30s (configurable)
- On full fill:
  - Update order status to `FILLED`
  - Record `filled_at` timestamp
  - Publish `OrderFilled` event to `pqap.order.filled`
  - Record `pqap_execution_fill_latency_ms` histogram
  - Record `pqap_execution_orders_filled_total` counter
- On partial fill:
  - Update order status to `PARTIAL_FILL`
  - Log `filled_qty` and `remaining_qty`
  - Publish `OrderPartialFill` event to `pqap.order.partial`
  - Record `pqap_execution_partial_fills_total` counter
  - Strategy decision: cancel remaining or wait (configurable per strategy)
- On cancel/timeout:
  - Update order status to `CANCELLED`
  - Publish `OrderCancelled` event to `pqap.order.cancelled`
  - Record `pqap_execution_orders_cancelled_total` counter

### Step 6: Order Audit Trail

- Write to PostgreSQL `trades` table (append-only, immutable per AD-6, FR-65)
- Table schema:
  ```sql
  CREATE TABLE trades (
      id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      order_id        UUID NOT NULL,
      client_order_id UUID NOT NULL UNIQUE,
      opportunity_id  UUID NOT NULL,
      market_id       TEXT NOT NULL,
      side            TEXT NOT NULL,
      price           NUMERIC(10,4) NOT NULL,
      size            NUMERIC(10,8) NOT NULL,
      filled_qty      NUMERIC(10,8) NOT NULL DEFAULT 0,
      fill_status     TEXT NOT NULL,  -- PENDING, PLACED, PARTIAL_FILL, FILLED, CANCELLED, FAILED
      pnl             NUMERIC(10,8) DEFAULT NULL,
      strategy_id     TEXT NOT NULL,
      latency_ms      INTEGER NOT NULL,
      risk_check      TEXT NOT NULL,
      slippage_check  TEXT NOT NULL,
      error_reason    TEXT DEFAULT '',
      account_id      UUID DEFAULT NULL,  -- for future multi-account (INF-18)
      placed_at       TIMESTAMPTZ NOT NULL,
      filled_at       TIMESTAMPTZ DEFAULT NULL,
      created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
  );
  ```
- No UPDATE or DELETE operations permitted (FR-65)
- All monetary values use Decimal precision (prices: 4dp, quantities: 8dp, PnL: 8dp) (INF-11)
- All timestamps UTC as TIMESTAMPTZ (INF-12)

### Step 7: Event Publishing

- All events follow standard schema (INF-17):
  - `event_id`: UUID v4
  - `event_type`: past tense verb + noun (INF-16)
  - `timestamp`: ISO 8601 UTC
  - `source`: "execution-engine"
  - `payload`: event-specific JSON
- Events to publish:
  - `OrderPlaced` → `pqap.order.placed` (on successful CLOB API response)
  - `OrderFilled` → `pqap.order.filled` (on full fill)
  - `OrderPartialFill` → `pqap.order.partial` (on partial fill)
  - `OrderCancelled` → `pqap.order.cancelled` (on cancel/timeout)
  - `OrderFailed` → `pqap.order.failed` (on any failure)
- Fire-and-forget with at-least-once delivery (AD-9)
- Consumers idempotent by event_id

### Step 8: Circuit Breaker

- Wrap all CLOB API calls in circuit breaker (AD-11)
- States:
  - **Closed** (normal): requests pass through, failure counter increments on error
  - **Open** (tripped): after 5 consecutive failures (configurable), all requests fail immediately for 60s cooldown
  - **Half-open** (probe): after cooldown, one probe request; success → close, fail → reopen
- On trip:
  - Set `pqap_execution_circuit_breaker_state` gauge to 1
  - Increment `pqap_execution_circuit_breaker_trips_total`
  - Publish `RiskAlert` to NATS
  - Send critical Telegram alert (bypasses throttling)
  - All subsequent order attempts rejected until manual resume
- State queryable via Prometheus metrics

## Testing

### Unit Tests

- **Pit Boss risk check (`risk_check_test.go`):**
  - ALLOW when budget available and limits not exceeded
  - DENY when daily budget exhausted
  - DENY when per-market position limit exceeded
  - DENY when per-strategy position limit exceeded
  - Latency < 10ms verified
  - Redis connection failure → fail-safe (DENY)

- **Slippage protection (`slippage_test.go`):**
  - Price within tolerance → PASS
  - Price beyond tolerance → FAIL with delta logged
  - Configurable tolerance via env var
  - Per-strategy override works
  - Edge case: exactly at tolerance → PASS (>= not >)

- **Order placement (`executor_test.go`):**
  - GTC limit order placed correctly
  - UUID client_order_id generated
  - Latency tracked
  - CLOB API response parsed correctly
  - API error → retry with backoff

- **Idempotency (`idempotency_test.go`):**
  - Unique client_order_id per order
  - Duplicate client_order_id detected and rejected
  - Memory cache TTL works correctly

- **Fill monitoring (`fill_monitor_test.go`):**
  - Full fill → OrderFilled event published
  - Partial fill → logged, OrderPartialFill published
  - Cancel → OrderCancelled published
  - Timeout → handled correctly

- **Circuit breaker (`breaker_test.go`):**
  - Closed → Open after 5 consecutive failures
  - Open → requests fail immediately
  - Half-open → probe after cooldown
  - Successful probe → close

- **Event publishing (`publisher_test.go`):**
  - Event schema correct (event_id, event_type, timestamp, source, payload)
  - event_id is UUID
  - timestamp is UTC
  - source is "execution-engine"

### Integration Tests

- **OpportunityDetected → OrderFilled flow:**
  - Mock NATS server + mock CLOB API
  - Publish OpportunityDetected
  - Verify risk check performed
  - Verify order placed via CLOB API
  - Verify OrderFilled published after fill
  - Verify trade logged to PostgreSQL

- **Pit Boss DENY → order rejected:**
  - Set Pit Boss state to DENY
  - Publish OpportunityDetected
  - Verify no order placed
  - Verify OrderFailed published with reason "risk_denied"
  - Verify trade logged with DENY result

- **Slippage exceeded → order rejected:**
  - Set market price beyond slippage tolerance
  - Publish OpportunityDetected
  - Verify no order placed
  - Verify OrderFailed published with reason "slippage_exceeded"

- **Partial fill → tracked correctly:**
  - Mock CLOB API returns partial fill
  - Verify partial fill logged
  - Verify OrderPartialFill published
  - Verify remaining quantity handling

- **Circuit breaker → halt trading:**
  - Mock CLOB API to return 5 consecutive errors
  - Verify circuit breaker trips
  - Verify subsequent orders rejected
  - Verify Telegram alert sent

- **Duplicate order → rejected:**
  - Send same OpportunityDetected twice quickly
  - Verify only one order placed
  - Verify duplicate rejected counter incremented

### Test Files

```
tests/unit/execution-engine/
├── risk_check_test.go         # Pit Boss risk check (allow/deny, latency)
├── slippage_test.go           # Slippage protection (within/beyond tolerance)
├── executor_test.go           # Order placement (GTC, UUID, latency)
├── idempotency_test.go        # Duplicate detection
├── fill_monitor_test.go       # Fill monitoring (full, partial, cancel)
├── breaker_test.go            # Circuit breaker state transitions
├── publisher_test.go          # Event schema validation
└── order_logger_test.go       # PostgreSQL trades table logging

tests/integration/
├── execution_full_flow_test.go    # OpportunityDetected → OrderFilled end-to-end
├── execution_risk_deny_test.go    # Pit Boss DENY → rejected
├── execution_slippage_test.go     # Slippage exceeded → rejected
├── execution_partial_fill_test.go # Partial fill → tracked
├── execution_circuit_breaker_test.go # Circuit breaker → halt
└── execution_idempotency_test.go  # Duplicate order → rejected
```

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| Story 1.1 | — | Scanner WebSocket Connection & Market Catalog (prerequisite) |
| Story 1.2 | — | Market Scanner — Stale Detection, Reconnect & Batching (prerequisite) |
| Story 1.3 | — | Arbitrage Detection & Opportunity Scoring (prerequisite — produces OpportunityDetected) |
| Story 1.7 | — | Risk Management — Pit Boss & Daily Budget (prerequisite — provides risk state in Redis) |
| `github.com/nats-io/nats.go` | latest | NATS event bus |
| `github.com/shopspring/decimal` | latest | Decimal precision (prices, quantities) |
| `github.com/prometheus/client_golang` | latest | Metrics export |
| `go.uber.org/zap` | latest | Structured logging |
| `github.com/google/uuid` | latest | Client order ID and event ID generation |
| `github.com/jackc/pgx/v5` | latest | PostgreSQL driver |
| `github.com/redis/go-redis/v9` | latest | Redis client (Pit Boss check) |
| `github.com/sony/gobreaker` | latest | Circuit breaker implementation |

## External Dependencies (Infrastructure)

| Service | Required | Purpose |
|---------|----------|---------|
| NATS server | Yes | Event bus (consume OpportunityDetected, produce order events) |
| Redis | Yes | Pit Boss risk state (synchronous GET, <10ms) |
| PostgreSQL | Yes | trades table (order audit trail, append-only) |
| Polymarket CLOB API | Yes | Order placement (POST /order), cancellation (DELETE /order), status (GET /order) |

## Definition of Done

- [ ] GTC limit orders placed via CLOB API (FR-17)
- [ ] Pit Boss consulted before every trade, <10ms latency (FR-18, FR-45, NFR-R1)
- [ ] Slippage protection working, configurable default 1% (FR-19)
- [ ] Idempotent order placement via UUID client order ID (FR-22, AD-3)
- [ ] Partial fills tracked and logged (FR-20)
- [ ] Order audit trail complete in PostgreSQL (FR-24, FR-62, FR-65)
- [ ] OrderFilled/OrderFailed events published to NATS (AD-9)
- [ ] Order placement latency <200ms (NFR-E1)
- [ ] 99.9% order placement success rate excluding API outages (NFR-E2)
- [ ] Zero duplicate orders under any failure scenario (NFR-E4)
- [ ] Circuit breaker trips after 5 consecutive API errors (FR-21)
- [ ] Prometheus metrics exported (AD-17)
- [ ] Structured JSON logging (INF-14)
- [ ] Decimal precision for all monetary values (INF-11)
- [ ] Unit tests pass with ≥80% coverage
- [ ] Integration tests pass

## References

| Reference | Description |
|-----------|-------------|
| FR-17 | Engine SHALL place limit orders (GTC) by default with configurable time-in-force override |
| FR-18 | Engine SHALL validate order against risk budget before placement |
| FR-19 | Engine SHALL implement slippage protection: reject if price moves beyond tolerance (default: 1%) |
| FR-20 | Engine SHALL handle partial fills: track filled quantity and decide cancel/wait based on strategy |
| FR-22 | Engine SHALL implement idempotent order placement (client order ID prevents duplicates) |
| FR-24 | Engine SHALL log every order attempt with: timestamp, market, side, price, size, result, latency |
| AD-3 | Execution Engine is the sole writer to Polymarket CLOB API; every order gets a unique client order ID (UUID) for idempotency |
| AD-10 | Sync patterns: Execution → Pit Boss (Redis GET, <10ms), Execution → CLOB API |
| AD-11 | Circuit breaker (closed/open/half-open) on all external calls |
| NFR-E1 | Order placement latency: within 200ms of decision |
| NFR-E2 | Order placement success rate: 99.9% (excluding API outages) |
| NFR-E4 | Zero duplicate orders under any failure scenario |
| NFR-R1 | Risk check latency: within 10ms of trade request |
| INF-11 | Decimal precision: all monetary values use Decimal (never float64); prices 4dp, quantities 8dp, PnL 8dp |
| INF-12 | All timestamps UTC as TIMESTAMPTZ; display timezone configurable |
| INF-14 | Structured JSON logs with timestamp, level, service, request_id, message, context |
| INF-15 | Prometheus metric naming: pqap_{service}_{metric_name}_{unit} |
| INF-16 | Event naming: past tense verb + noun (OrderFilled, OrderFailed) |
| INF-17 | All events include: event_id (UUID), event_type, timestamp (ISO 8601 UTC), source, payload |
| INF-18 | Include account_id as nullable column in all relevant tables from day one |

## Architecture Decisions Impacting This Story

| Decision | Impact |
|----------|--------|
| AD-3 (Service Boundary) | Execution Engine is sole CLOB API writer. Every order gets UUID client_order_id for idempotency. |
| AD-4 (Pit Boss) | Synchronous Redis GET before every trade. Returns ALLOW/DENY. No trade bypasses Pit Boss. |
| AD-6 (Data Ownership) | trades table written by Execution Engine only. Append-only, immutable. |
| AD-9 (Event Bus) | Consumes `pqap.opportunity.detected`, produces order lifecycle events. Fire-and-forget, at-least-once, idempotent. |
| AD-10 (Sync vs Async) | Execution → Pit Boss: sync Redis GET. Execution → CLOB API: sync HTTP. Execution → NATS: async publish. |
| AD-11 (Circuit Breaker) | 5 consecutive API errors → trip, halt trading, alert, manual resume. |
| AD-17 (Observability) | Prometheus metrics for order count, latency, circuit breaker state, risk denials. |
| INF-11 (Decimal) | All prices, quantities, PnL use `decimal.Decimal` — never `float64`. |
| INF-18 (Multi-Account) | `account_id` nullable column in trades table from day one. |

## Directory Structure

```
services/execution-engine/
├── cmd/
│   └── main.go                       # Entry point — starts subscriber, executor, monitor
├── internal/
│   ├── executor/
│   │   ├── executor.go               # Core execution flow (receive → check → place → monitor)
│   │   ├── risk_check.go             # Pit Boss risk check logic
│   │   ├── slippage.go               # Slippage protection
│   │   └── idempotency.go            # Duplicate client_order_id detection
│   ├── monitor/
│   │   └── fill_monitor.go           # Fill monitoring (poll CLOB API for status)
│   ├── circuit_breaker/
│   │   └── breaker.go                # API circuit breaker (closed/open/half-open)
│   ├── logger/
│   │   └── order_logger.go           # PostgreSQL trades table logging
│   └── ports/
│       ├── order.go                  # OrderPort interface (CLOB API)
│       ├── risk.go                   # RiskPort interface (Pit Boss)
│       └── event.go                  # EventPort interface (NATS)
├── adapters/
│   ├── polymarket_clob.go            # Polymarket CLOB API adapter
│   ├── redis_risk.go                 # Pit Boss risk check adapter (Redis)
│   ├── nats_subscriber.go            # NATS subscriber (OpportunityDetected)
│   ├── nats_publisher.go             # NATS publisher (order lifecycle events)
│   └── postgres_repo.go              # PostgreSQL adapter (trades table)
├── config/
│   └── config.go                     # Configuration (slippage, circuit breaker, polling)
├── metrics/
│   └── metrics.go                    # Prometheus metrics (14 metrics)
└── go.mod
```

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `EXECUTION_SLIPPAGE_TOLERANCE` | `0.01` | Slippage tolerance (1% = 0.01) |
| `EXECUTION_TIME_IN_FORCE` | `GTC` | Default order time-in-force |
| `EXECUTION_CIRCUIT_BREAKER_THRESHOLD` | `5` | Consecutive errors to trip circuit breaker |
| `EXECUTION_CIRCUIT_BREAKER_COOLDOWN` | `60s` | Circuit breaker cooldown period |
| `EXECUTION_FILL_POLL_INTERVAL` | `100ms` | Fill monitoring poll interval |
| `EXECUTION_FILL_POLL_TIMEOUT` | `30s` | Fill monitoring poll timeout |
| `EXECUTION_MAX_RETRIES` | `3` | Max retries on API error |
| `EXECUTION_RETRY_BACKOFF_INITIAL` | `100ms` | Initial retry backoff |
| `EXECUTION_RETRY_BACKOFF_MAX` | `5s` | Max retry backoff |
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |
| `REDIS_URL` | `localhost:6379` | Redis URL (Pit Boss) |
| `POSTGRES_URL` | `postgres://localhost:5432/pqap` | PostgreSQL connection string |
| `POLYMARKET_CLOB_URL` | `https://clob.polymarket.com` | Polymarket CLOB API base URL |
