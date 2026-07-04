# Story 1.5: Execution Engine — Atomic YES+NO & Circuit Breaker

## Story

As a quant trader,
I want YES+NO arbitrage legs executed atomically and a circuit breaker to halt on repeated API failures,
So that I never end up with a half-filled arb position or blow up during an API outage.

## Status

ready-for-dev

## Acceptance Criteria

**Given** a YES+NO arbitrage opportunity is approved for execution
**When** the execution engine places both legs
**Then** both orders are placed within a 500ms window
**And** if one leg fails, the other is cancelled within 1 second
**And** if one leg is partially filled and the other fails, the partial fill is tracked and logged

**Given** the Polymarket CLOB API returns consecutive errors
**When** the error count reaches the configurable threshold (default: 5)
**Then** the circuit breaker trips and all trading is halted
**And** an alert is sent via Telegram (critical notification, bypasses throttling)
**And** trading remains halted until manual resume is initiated by the user
**And** the circuit breaker state is logged and queryable

## Technical Requirements

### Architecture Context

- **Service:** `execution-engine` (Go)
- **Paradigm:** Event-driven hexagonal architecture (ports & adapters)
- **Event bus:** NATS (JetStream for durable subscriptions)
- **Risk:** Pit Boss (Redis) — synchronous check before every trade (AD-4)
- **API:** Polymarket CLOB API — sole writer (AD-3)
- **Database:** PostgreSQL `trades` table (append-only, immutable) (AD-6)
- **Pattern:** Both legs of YES+NO arbitrage placed within 500ms (NFR-E3). If one leg fails, the other is cancelled within 1s. Circuit breaker on all CLOB API calls — 5 consecutive errors trips breaker, halts all trading, sends critical Telegram alert, requires manual resume (AD-11).

### Key Components to Implement

1. **Atomic YES+NO Executor** (`internal/executor/atomic.go`)
   - Coordinates placement of YES and NO legs within 500ms window (FR-23, NFR-E3)
   - Uses goroutines with `sync.WaitGroup` or `errgroup` for parallel placement
   - Both legs must succeed or both must be cancelled (atomic semantics)
   - On first leg failure: immediately cancel second leg (within 1s)
   - Track pair state: `PAIR_PENDING`, `PAIR_PLACING`, `PAIR_ONE_LEG`, `PAIR_FILLED`, `PAIR_PARTIAL`, `PAIR_CANCELLED`, `PAIR_FAILED`

2. **Leg Failure Handler** (`internal/executor/leg_handler.go`)
   - When YES leg fails after NO leg placed: cancel NO leg within 1s
   - When NO leg fails after YES leg placed: cancel YES leg within 1s
   - If one leg is partially filled and the other fails:
     - Track partial fill position
     - Log partial fill with quantity, price, and which leg failed
     - Publish `AtomicLegFailed` event with partial fill context
   - Cancel via CLOB API: `DELETE /order` with `client_order_id`
   - Track cancellation latency (target: <1s per acceptance criteria)

3. **Partial Fill Tracker** (`internal/executor/partial_tracker.go`)
   - Track partial fills for atomic pairs
   - Record: pair_id, leg (YES/NO), filled_qty, remaining_qty, fill_price, timestamp
   - Store in PostgreSQL `atomic_partial_fills` table
   - On pair completion: reconcile partial fills across both legs
   - On one-leg failure: log orphaned partial fill, alert user

4. **Circuit Breaker** (`internal/circuit_breaker/breaker.go`)
   - Wrap all CLOB API calls (order placement, cancellation, status polling)
   - States: **closed** (normal), **open** (tripped), **half-open** (probe)
   - Trip after N consecutive API errors (default: 5, configurable via `EXECUTION_CIRCUIT_BREAKER_THRESHOLD`)
   - Cooldown period: 60s (configurable via `EXECUTION_CIRCUIT_BREAKER_COOLDOWN`)
   - On trip:
     - Halt all trading immediately
     - Publish `CircuitBreakerTripped` event to NATS (`pqap.risk.circuit_breaker`)
     - Send critical Telegram alert (bypasses all throttling per FR-81, FR-82)
     - Set `pqap_execution_circuit_breaker_state` gauge to 1
     - Increment `pqap_execution_circuit_breaker_trips_total` counter
     - Log trip with: timestamp, error_count, last_error, consecutive_failures
   - Manual resume required: `POST /api/v1/execution/circuit-breaker/resume`
   - On resume: reset error counter, close circuit, resume trading
   - State queryable via Prometheus metrics and API endpoint

5. **Circuit Breaker Resume Handler** (`internal/circuit_breaker/resume.go`)
   - API endpoint: `POST /api/v1/execution/circuit-breaker/resume`
   - Requires authentication (JWT)
   - Validates circuit is in OPEN state
   - Resets consecutive error counter to 0
   - Transitions to CLOSED state
   - Publishes `CircuitBreakerResumed` event
   - Logs resume with: timestamp, user_id, uptime_before_trip

6. **Telegram Alert on Trip** (`adapters/nats_publisher.go` → notification service)
   - Publish `NotificationRequest` event to NATS (`pqap.notification.send`)
   - Category: `CRITICAL`
   - Bypass throttling (critical notifications bypass all throttling per FR-82)
   - Message format: "⚠️ CIRCUIT BREAKER TRIPPED — {error_count} consecutive API errors. Last error: {error_detail}. Trading halted. Manual resume required."
   - Delivery target: <5s (NFR-N1)

7. **Order Audit Trail Enhancement** (`internal/logger/order_logger.go`)
   - Log atomic pair execution with: pair_id, yes_order_id, no_order_id, placement_latency_ms, both_filled, partial_fill_details
   - Log circuit breaker state changes: timestamp, old_state, new_state, error_count, trigger_reason
   - All entries to PostgreSQL `trades` and `risk_events` tables (append-only)

### Data Models

**AtomicPair (internal domain model):**
```go
type AtomicPair struct {
    ID              string          `json:"id"`               // UUID — pair ID
    OpportunityID   string          `json:"opportunity_id"`   // Link to opportunity
    MarketID        string          `json:"market_id"`
    YesOrderID      string          `json:"yes_order_id"`     // UUID — YES leg order
    NoOrderID       string          `json:"no_order_id"`      // UUID — NO leg order
    YesClientOrderID string         `json:"yes_client_order_id"`
    NoClientOrderID  string         `json:"no_client_order_id"`
    YesPrice        decimal.Decimal `json:"yes_price"`        // 4dp
    NoPrice         decimal.Decimal `json:"no_price"`         // 4dp
    YesSize         decimal.Decimal `json:"yes_size"`         // 8dp
    NoSize          decimal.Decimal `json:"no_size"`          // 8dp
    YesFilledQty    decimal.Decimal `json:"yes_filled_qty"`   // 8dp
    NoFilledQty     decimal.Decimal `json:"no_filled_qty"`    // 8dp
    Status          PairStatus      `json:"status"`           // PENDING, PLACING, ONE_LEG, FILLED, PARTIAL, CANCELLED, FAILED
    PlacementLatencyMs int64        `json:"placement_latency_ms"` // time to place both legs
    FailureReason   string          `json:"failure_reason"`   // empty if success
    FailedLeg       string          `json:"failed_leg"`       // "YES", "NO", or ""
    StrategyID      string          `json:"strategy_id"`
    AccountID       *string         `json:"account_id"`       // nullable, for future multi-account
    CreatedAt       time.Time       `json:"created_at"`       // UTC TIMESTAMPTZ
    CompletedAt     *time.Time      `json:"completed_at"`     // nullable
}
```

**CircuitBreakerState (internal domain model):**
```go
type CircuitBreakerState struct {
    State               string    `json:"state"`                // "CLOSED", "OPEN", "HALF_OPEN"
    ConsecutiveErrors   int       `json:"consecutive_errors"`
    LastError           string    `json:"last_error"`
    LastErrorTime       time.Time `json:"last_error_time"`
    TrippedAt           *time.Time `json:"tripped_at"`         // nullable
    ResumedAt           *time.Time `json:"resumed_at"`         // nullable
    TotalTrips          int64     `json:"total_trips"`
    CooldownRemainingMs int64     `json:"cooldown_remaining_ms"`
}
```

**CircuitBreakerTripped Event:**
```go
type CircuitBreakerTripped struct {
    EventID   string                    `json:"event_id"`   // UUID
    EventType string                    `json:"event_type"` // "CircuitBreakerTripped"
    Timestamp time.Time                 `json:"timestamp"`  // ISO 8601 UTC
    Source    string                    `json:"source"`     // "execution-engine"
    Payload   CircuitBreakerTrippedPayload `json:"payload"`
}

type CircuitBreakerTrippedPayload struct {
    ConsecutiveErrors int       `json:"consecutive_errors"`
    LastError         string    `json:"last_error"`
    LastErrorTime     time.Time `json:"last_error_time"`
    CooldownSeconds   int       `json:"cooldown_seconds"`
    Message           string    `json:"message"`
}
```

**AtomicLegFailed Event:**
```go
type AtomicLegFailed struct {
    EventID   string                 `json:"event_id"`   // UUID
    EventType string                 `json:"event_type"` // "AtomicLegFailed"
    Timestamp time.Time              `json:"timestamp"`  // ISO 8601 UTC
    Source    string                 `json:"source"`     // "execution-engine"
    Payload   AtomicLegFailedPayload `json:"payload"`
}

type AtomicLegFailedPayload struct {
    PairID          string          `json:"pair_id"`
    OpportunityID   string          `json:"opportunity_id"`
    MarketID        string          `json:"market_id"`
    FailedLeg       string          `json:"failed_leg"`       // "YES" or "NO"
    FailedOrderID   string          `json:"failed_order_id"`
    FailureReason   string          `json:"failure_reason"`
    SuccessfulLeg   string          `json:"successful_leg"`   // "YES" or "NO"
    SuccessfulOrderID string        `json:"successful_order_id"`
    SuccessfulFilledQty decimal.Decimal `json:"successful_filled_qty"`
    CancelledLeg    string          `json:"cancelled_leg"`    // "YES" or "NO"
    CancelledOrderID string         `json:"cancelled_order_id"`
    StrategyID      string          `json:"strategy_id"`
}
```

### API Endpoints

**Polymarket CLOB API (consumed):**
- `POST /order` — Place a new limit order
- `DELETE /order` — Cancel an existing order
- `GET /order/{order_id}` — Get order status (for fill monitoring)

**Internal API (produced):**
- `POST /api/v1/execution/circuit-breaker/resume` — Resume trading after circuit breaker trip (requires JWT auth)

**NATS Subjects (from AD-9):**
```
pqap.opportunity.detected       # Consumed: OpportunityDetected
pqap.order.placed               # Produced: OrderPlaced
pqap.order.filled               # Produced: OrderFilled
pqap.order.partial              # Produced: OrderPartialFill
pqap.order.cancelled            # Produced: OrderCancelled
pqap.order.failed               # Produced: OrderFailed
pqap.order.atomic_leg_failed    # Produced: AtomicLegFailed
pqap.risk.circuit_breaker       # Produced: CircuitBreakerTripped
pqap.risk.emergency             # Consumed: EmergencyStop (halt all trading)
pqap.notification.send          # Produced: NotificationRequest (critical Telegram alert)
```

### Prometheus Metrics (AD-17)

```
pqap_execution_atomic_pairs_total              # Counter — total atomic pair attempts
pqap_execution_atomic_pairs_filled_total       # Counter — fully filled pairs
pqap_execution_atomic_pairs_partial_total      # Counter — partial fill pairs
pqap_execution_atomic_pairs_cancelled_total    # Counter — cancelled pairs
pqap_execution_atomic_pairs_failed_total       # Counter — failed pairs
pqap_execution_atomic_placement_latency_ms     # Histogram — time to place both legs (target: <500ms)
pqap_execution_atomic_cancel_latency_ms        # Histogram — time to cancel other leg on failure (target: <1s)
pqap_execution_circuit_breaker_state           # Gauge — 0=closed, 1=open, 2=half-open
pqap_execution_circuit_breaker_trips_total     # Counter — total circuit breaker trips
pqap_execution_circuit_breaker_consecutive_errors # Gauge — current consecutive error count
pqap_execution_circuit_breaker_cooldown_remaining_ms # Gauge — cooldown remaining
```

## Implementation Guide

### Step 1: Atomic YES+NO Executor

- Receive `OpportunityDetected` event (after Pit Boss risk check and slippage check from Story 1.4)
- Validate opportunity is YES+NO arbitrage (not single-leg)
- Create `AtomicPair` record with status `PENDING`
- Generate UUID v4 `client_order_id` for both YES and NO legs
- Launch parallel goroutines using `errgroup`:
  ```
  g, ctx := errgroup.WithContext(ctx)
  g.Go(func() error { return placeLeg(ctx, yesOrder) })
  g.Go(func() error { return placeLeg(ctx, noOrder) })
  ```
- Record `start_time = time.Now()` before launching goroutines
- Wait for both goroutines with timeout (500ms per NFR-E3)
- On both success: update pair status to `FILLED`, record `placement_latency_ms`
- On timeout: cancel any placed legs, update status to `FAILED`
- Log pair execution to PostgreSQL

### Step 2: Leg Failure Handler

- In `placeLeg()` goroutine, if CLOB API returns error:
  - Return error immediately to errgroup
  - errgroup cancels context → other goroutine detects cancellation
- On first leg failure detected:
  - Record `cancel_start = time.Now()`
  - Cancel the other leg via `DELETE /order` (if it was placed)
  - Track cancellation latency (target: <1s)
  - Update pair status based on fill state:
    - Neither leg filled → `CANCELLED`
    - One leg partially filled → `PARTIAL` (log orphaned partial fill)
    - One leg fully filled → `FAILED` (critical — log and alert)
  - Publish `AtomicLegFailed` event
  - Log to `trades` table with full context

### Step 3: Partial Fill Tracking

- Monitor fill status for both legs during atomic execution
- If one leg receives partial fill and the other fails:
  - Record partial fill details: pair_id, leg, filled_qty, remaining_qty, fill_price
  - Store in PostgreSQL `atomic_partial_fills` table
  - Log as `PARTIAL` status in `trades` table
  - The partially filled position becomes an orphaned position — position manager will track it
  - Publish `AtomicLegFailed` with partial fill context
  - Alert user via Telegram (warning level)
- On pair completion (both filled): reconcile partial fills if any

### Step 4: Circuit Breaker Implementation

- Wrap all CLOB API calls in circuit breaker (order placement, cancellation, status polling)
- Track consecutive errors in memory (atomic counter)
- On each API error:
  - Increment `pqap_execution_circuit_breaker_consecutive_errors` gauge
  - Record `last_error` and `last_error_time`
  - If `consecutive_errors >= threshold`:
    - Trip circuit breaker
    - Set `pqap_execution_circuit_breaker_state` to 1 (open)
    - Increment `pqap_execution_circuit_breaker_trips_total`
    - Publish `CircuitBreakerTripped` event
    - Send critical Telegram alert via `pqap.notification.send`
    - Log trip to `risk_events` table
    - Set `tripped_at` timestamp
- On circuit open:
  - All subsequent CLOB API calls fail immediately with `ErrCircuitBreakerOpen`
  - All order attempts rejected with reason "circuit_breaker_open"
  - `cooldown_remaining_ms` decrements in real-time
- After cooldown expires:
  - Transition to `HALF_OPEN`
  - Allow one probe request
  - If probe succeeds: close circuit, reset counter, resume trading
  - If probe fails: reopen circuit, restart cooldown

### Step 5: Telegram Alert on Trip

- On circuit breaker trip, publish `NotificationRequest` to NATS:
  ```go
  notification := NotificationRequest{
      EventID:   uuid.New().String(),
      EventType: "NotificationRequest",
      Timestamp: time.Now().UTC(),
      Source:    "execution-engine",
      Payload: NotificationRequestPayload{
          Category:  "CRITICAL",
          Title:     "CIRCUIT BREAKER TRIPPED",
          Message:   fmt.Sprintf("⚠️ %d consecutive API errors. Last error: %s. Trading halted. Manual resume required.", errorCount, lastError),
          Channel:   "telegram",
          Priority:  "high",
          BypassThrottle: true,  // Critical notifications bypass throttling (FR-82)
      },
  }
  ```
- Delivery target: <5s (NFR-N1)
- Notification service handles delivery and retry

### Step 6: Manual Resume

- Expose API endpoint: `POST /api/v1/execution/circuit-breaker/resume`
- Request body: `{ "reason": "string" }` (optional)
- Authentication: JWT token required
- Validation:
  - Circuit must be in `OPEN` state
  - User must be authenticated
- On resume:
  - Reset `consecutive_errors` to 0
  - Clear `last_error`
  - Set `pqap_execution_circuit_breaker_state` to 0 (closed)
  - Set `resumed_at` timestamp
  - Publish `CircuitBreakerResumed` event
  - Log resume to `risk_events` table
  - Trading automatically resumes on next opportunity

### Step 7: State Queryability

- Prometheus metrics (always current):
  - `pqap_execution_circuit_breaker_state` — 0=closed, 1=open, 2=half-open
  - `pqap_execution_circuit_breaker_consecutive_errors` — current error count
  - `pqap_execution_circuit_breaker_cooldown_remaining_ms` — cooldown timer
- API endpoint: `GET /api/v1/execution/circuit-breaker/status`
  - Returns full `CircuitBreakerState` JSON
  - Includes: state, consecutive_errors, last_error, tripped_at, cooldown_remaining_ms

### Step 8: Event Publishing

- All events follow standard schema (INF-17):
  - `event_id`: UUID v4
  - `event_type`: past tense verb + noun (INF-16)
  - `timestamp`: ISO 8601 UTC
  - `source`: "execution-engine"
  - `payload`: event-specific JSON
- New events for this story:
  - `AtomicLegFailed` → `pqap.order.atomic_leg_failed` (on leg failure during atomic pair)
  - `CircuitBreakerTripped` → `pqap.risk.circuit_breaker` (on breaker trip)
  - `CircuitBreakerResumed` → `pqap.risk.circuit_breaker` (on manual resume)
  - `NotificationRequest` → `pqap.notification.send` (critical Telegram alert)
- Fire-and-forget with at-least-once delivery (AD-9)
- Consumers idempotent by event_id

### Step 9: Order Audit Trail Enhancement

- Extend PostgreSQL `trades` table with atomic pair context:
  ```sql
  ALTER TABLE trades ADD COLUMN pair_id UUID DEFAULT NULL;
  ALTER TABLE trades ADD COLUMN leg TEXT DEFAULT NULL;        -- "YES", "NO", or NULL
  ALTER TABLE trades ADD COLUMN pair_status TEXT DEFAULT NULL; -- pair-level status
  ```
- New table for partial fill tracking:
  ```sql
  CREATE TABLE atomic_partial_fills (
      id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      pair_id         UUID NOT NULL,
      leg             TEXT NOT NULL,        -- "YES" or "NO"
      filled_qty      NUMERIC(10,8) NOT NULL,
      remaining_qty   NUMERIC(10,8) NOT NULL,
      fill_price      NUMERIC(10,4) NOT NULL,
      order_id        UUID NOT NULL,
      client_order_id UUID NOT NULL,
      market_id       TEXT NOT NULL,
      strategy_id     TEXT NOT NULL,
      account_id      UUID DEFAULT NULL,
      created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
  );
  ```
- New table for circuit breaker events:
  ```sql
  CREATE TABLE circuit_breaker_events (
      id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      event_type      TEXT NOT NULL,        -- "TRIPPED", "RESUMED"
      state_from      TEXT NOT NULL,
      state_to        TEXT NOT NULL,
      consecutive_errors INTEGER,
      last_error      TEXT,
      cooldown_seconds INTEGER,
      user_id         TEXT DEFAULT NULL,     -- for resume events
      reason          TEXT DEFAULT '',
      created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
  );
  ```

## Testing

### Unit Tests

- **Atomic executor (`atomic_test.go`):**
  - Both legs placed within 500ms → pair FILLED
  - First leg fails → second leg cancelled within 1s
  - Both legs fail → pair CANCELLED
  - Timeout (500ms exceeded) → legs cancelled, pair FAILED
  - Concurrent placement correctness (race conditions)

- **Leg failure handler (`leg_handler_test.go`):**
  - YES fails, NO placed → NO cancelled, pair CANCELLED
  - NO fails, YES placed → YES cancelled, pair CANCELLED
  - YES fails, NO partially filled → partial tracked, pair PARTIAL
  - Cancel latency < 1s verified

- **Partial fill tracker (`partial_tracker_test.go`):**
  - One leg partial, other fails → partial recorded
  - Both legs partial → both recorded
  - Partial fill reconciliation on pair completion

- **Circuit breaker (`breaker_test.go`):**
  - Closed → Open after 5 consecutive failures
  - Open → requests fail immediately with `ErrCircuitBreakerOpen`
  - Half-open → probe after cooldown
  - Successful probe → close
  - Failed probe → reopen
  - Manual resume → close, reset counter
  - Concurrent access safety

- **Telegram alert (`notification_test.go`):**
  - Circuit breaker trip → NotificationRequest published
  - Category is CRITICAL
  - BypassThrottle is true
  - Message format correct

### Integration Tests

- **Atomic YES+NO → both filled:**
  - Mock CLOB API returns success for both legs
  - Verify both orders placed within 500ms
  - Verify pair status FILLED
  - Verify trade logged with pair context

- **Atomic YES+NO → one leg fails:**
  - Mock CLOB API fails for YES leg, succeeds for NO leg
  - Verify NO leg cancelled
  - Verify pair status CANCELLED
  - Verify AtomicLegFailed event published
  - Verify cancellation latency < 1s

- **Atomic YES+NO → partial fill + failure:**
  - Mock CLOB API returns partial fill for YES, error for NO
  - Verify YES leg cancelled
  - Verify partial fill tracked in `atomic_partial_fills`
  - Verify pair status PARTIAL
  - Verify Telegram warning sent

- **Circuit breaker → halt and resume:**
  - Mock CLOB API to return 5 consecutive errors
  - Verify circuit breaker trips
  - Verify all subsequent orders rejected
  - Verify Telegram alert sent
  - Call resume endpoint
  - Verify trading resumes

- **Circuit breaker → state queryable:**
  - Trip circuit breaker
  - Query Prometheus metrics
  - Verify state = 1 (open)
  - Verify consecutive_errors = 5
  - Query API endpoint
  - Verify full state returned

### Test Files

```
tests/unit/execution-engine/
├── atomic_test.go                  # Atomic YES+NO execution (parallel, timeout, success)
├── leg_handler_test.go             # Leg failure handling (cancel, partial, latency)
├── partial_tracker_test.go         # Partial fill tracking (record, reconcile)
├── breaker_test.go                 # Circuit breaker state transitions (closed/open/half-open)
├── resume_test.go                  # Manual resume handler (auth, validation, reset)
├── notification_test.go            # Telegram alert on trip (category, bypass, format)
└── atomic_logger_test.go           # Atomic pair audit trail logging

tests/integration/
├── execution_atomic_both_filled_test.go    # Both legs succeed
├── execution_atomic_one_leg_fails_test.go  # One leg fails → cancel other
├── execution_atomic_partial_fill_test.go   # Partial fill + failure
├── execution_circuit_breaker_trip_test.go  # 5 errors → trip → halt
├── execution_circuit_breaker_resume_test.go # Manual resume
└── execution_circuit_breaker_query_test.go # State queryability
```

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| Story 1.1 | — | Scanner WebSocket Connection & Market Catalog (prerequisite) |
| Story 1.2 | — | Market Scanner — Stale Detection, Reconnect & Batching (prerequisite) |
| Story 1.3 | — | Arbitrage Detection & Opportunity Scoring (prerequisite — produces OpportunityDetected) |
| Story 1.4 | — | Execution Engine — Order Placement & Slippage Protection (prerequisite — base execution flow) |
| Story 1.7 | — | Risk Management — Pit Boss & Daily Budget (prerequisite — provides risk state in Redis) |
| Story 1.10 | — | Telegram Notifications (prerequisite — delivers circuit breaker alerts) |
| `github.com/nats-io/nats.go` | latest | NATS event bus |
| `github.com/shopspring/decimal` | latest | Decimal precision (prices, quantities) |
| `github.com/prometheus/client_golang` | latest | Metrics export |
| `go.uber.org/zap` | latest | Structured logging |
| `github.com/google/uuid` | latest | Pair ID, event ID generation |
| `github.com/jackc/pgx/v5` | latest | PostgreSQL driver |
| `github.com/redis/go-redis/v9` | latest | Redis client (Pit Boss check) |
| `github.com/sony/gobreaker` | latest | Circuit breaker implementation |
| `golang.org/x/sync/errgroup` | latest | Parallel leg placement with error propagation |

## External Dependencies (Infrastructure)

| Service | Required | Purpose |
|---------|----------|---------|
| NATS server | Yes | Event bus (consume OpportunityDetected, produce order/risk events) |
| Redis | Yes | Pit Boss risk state (synchronous GET, <10ms) |
| PostgreSQL | Yes | trades, atomic_partial_fills, circuit_breaker_events tables |
| Polymarket CLOB API | Yes | Order placement, cancellation, status polling |
| Telegram Bot API | Yes | Critical circuit breaker alerts (via notification service) |

## Definition of Done

- [ ] YES+NO atomic execution: both legs placed within 500ms window (FR-23, NFR-E3)
- [ ] Leg failure handling: other leg cancelled within 1s (FR-23)
- [ ] Partial fill tracking: orphaned partial fills logged and tracked (FR-20)
- [ ] Circuit breaker: trips after 5 consecutive API errors (FR-21, AD-11)
- [ ] Circuit breaker: halts all trading on trip (FR-21)
- [ ] Circuit breaker: sends critical Telegram alert on trip (bypasses throttling)
- [ ] Circuit breaker: manual resume required to restart trading (FR-21)
- [ ] Circuit breaker: state queryable via Prometheus and API (AD-17)
- [ ] Atomic pair audit trail in PostgreSQL (FR-24, FR-62)
- [ ] Circuit breaker events logged in PostgreSQL (FR-47)
- [ ] Prometheus metrics exported (AD-17)
- [ ] Structured JSON logging (INF-14)
- [ ] Decimal precision for all monetary values (INF-11)
- [ ] All timestamps UTC as TIMESTAMPTZ (INF-12)
- [ ] Unit tests pass with ≥80% coverage
- [ ] Integration tests pass

## References

| Reference | Description |
|-----------|-------------|
| FR-20 | Engine SHALL handle partial fills: track filled quantity and decide cancel/wait based on strategy |
| FR-21 | Engine SHALL implement circuit breaker: halt trading after N consecutive API errors (default: 5) |
| FR-23 | Engine SHALL execute both legs of YES+NO arbitrage atomically (both succeed or both cancel) |
| FR-24 | Engine SHALL log every order attempt with: timestamp, market, side, price, size, result, latency |
| FR-81 | Center SHALL categorize notifications: critical, warning, info, debug |
| FR-82 | Center SHALL support notification throttling (max 10 per minute for non-critical) |
| AD-3 | Execution Engine is the sole writer to Polymarket CLOB API; every order gets a unique client order ID (UUID) for idempotency |
| AD-11 | Circuit breaker (closed/open/half-open) on all external calls; global emergency stop on critical failures |
| AD-17 | Observability: Prometheus metrics, Grafana dashboards, structured JSON logs |
| NFR-E3 | YES+NO atomic execution window: within 500ms |
| NFR-N1 | Critical notification latency: within 5s |
| NFR-N2 | Critical notification delivery rate: 99.9% |
| INF-11 | Decimal precision: all monetary values use Decimal (never float64); prices 4dp, quantities 8dp, PnL 8dp |
| INF-12 | All timestamps UTC as TIMESTAMPTZ; display timezone configurable |
| INF-14 | Structured JSON logs with timestamp, level, service, request_id, message, context |
| INF-15 | Prometheus metric naming: pqap_{service}_{metric_name}_{unit} |
| INF-16 | Event naming: past tense verb + noun (CircuitBreakerTripped, AtomicLegFailed) |
| INF-17 | All events include: event_id (UUID), event_type, timestamp (ISO 8601 UTC), source, payload |
| INF-18 | Include account_id as nullable column in all relevant tables from day one |

## Architecture Decisions Impacting This Story

| Decision | Impact |
|----------|--------|
| AD-3 (Service Boundary) | Execution Engine is sole CLOB API writer. Every order gets UUID client_order_id for idempotency. Both legs of YES+NO arb placed within 500ms; if one fails, other cancelled within 1s. |
| AD-11 (Circuit Breaker) | 5 consecutive API errors → trip, halt trading, alert, manual resume. Three states: closed/open/half-open. Cooldown period before probe. |
| AD-17 (Observability) | Prometheus metrics for atomic pair success/failure, circuit breaker state, placement latency, cancellation latency. |
| AD-9 (Event Bus) | Consumes `pqap.opportunity.detected`, produces order lifecycle events + `CircuitBreakerTripped` + `AtomicLegFailed`. Fire-and-forget, at-least-once, idempotent. |
| AD-10 (Sync vs Async) | Leg placement: parallel async within 500ms window. Pit Boss check: sync before pair. CLOB API: sync HTTP per leg. NATS: async publish. |
| AD-6 (Data Ownership) | trades table extended with pair_id, leg, pair_status. New atomic_partial_fills and circuit_breaker_events tables. |
| INF-11 (Decimal) | All prices, quantities, PnL use `decimal.Decimal` — never `float64`. |
| INF-18 (Multi-Account) | `account_id` nullable column in new tables from day one. |

## Directory Structure

```
services/execution-engine/
├── cmd/
│   └── main.go                           # Entry point — starts subscriber, executor, monitor, breaker
├── internal/
│   ├── executor/
│   │   ├── executor.go                   # Core execution flow (receive → check → place → monitor)
│   │   ├── atomic.go                     # Atomic YES+NO execution (parallel placement, 500ms window)
│   │   ├── leg_handler.go                # Leg failure handling (cancel other leg, track partial fills)
│   │   ├── partial_tracker.go            # Partial fill tracking for atomic pairs
│   │   ├── risk_check.go                 # Pit Boss risk check logic
│   │   ├── slippage.go                   # Slippage protection
│   │   └── idempotency.go               # Duplicate client_order_id detection
│   ├── monitor/
│   │   └── fill_monitor.go              # Fill monitoring (poll CLOB API for status)
│   ├── circuit_breaker/
│   │   ├── breaker.go                   # API circuit breaker (closed/open/half-open)
│   │   └── resume.go                    # Manual resume handler (API endpoint)
│   ├── logger/
│   │   ├── order_logger.go              # PostgreSQL trades table logging
│   │   ├── atomic_logger.go             # Atomic pair and partial fill logging
│   │   └── breaker_logger.go            # Circuit breaker event logging
│   └── ports/
│       ├── order.go                     # OrderPort interface (CLOB API)
│       ├── risk.go                      # RiskPort interface (Pit Boss)
│       └── event.go                     # EventPort interface (NATS)
├── adapters/
│   ├── polymarket_clob.go               # Polymarket CLOB API adapter
│   ├── redis_risk.go                    # Pit Boss risk check adapter (Redis)
│   ├── nats_subscriber.go               # NATS subscriber (OpportunityDetected)
│   ├── nats_publisher.go                # NATS publisher (order/risk/notification events)
│   └── postgres_repo.go                 # PostgreSQL adapter (trades, partial fills, breaker events)
├── config/
│   └── config.go                        # Configuration (atomic timeout, breaker threshold, cooldown)
├── metrics/
│   └── metrics.go                       # Prometheus metrics (12 metrics)
└── go.mod
```

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `EXECUTION_ATOMIC_TIMEOUT_MS` | `500` | Max time to place both YES+NO legs (NFR-E3) |
| `EXECUTION_LEG_CANCEL_TIMEOUT_MS` | `1000` | Max time to cancel other leg on failure |
| `EXECUTION_CIRCUIT_BREAKER_THRESHOLD` | `5` | Consecutive API errors to trip breaker (FR-21) |
| `EXECUTION_CIRCUIT_BREAKER_COOLDOWN` | `60s` | Cooldown period before half-open probe |
| `EXECUTION_CIRCUIT_BREAKER_PROBE_TIMEOUT` | `5s` | Timeout for half-open probe request |
| `EXECUTION_SLIPPAGE_TOLERANCE` | `0.01` | Slippage tolerance (1% = 0.01) |
| `EXECUTION_TIME_IN_FORCE` | `GTC` | Default order time-in-force |
| `EXECUTION_FILL_POLL_INTERVAL` | `100ms` | Fill monitoring poll interval |
| `EXECUTION_FILL_POLL_TIMEOUT` | `30s` | Fill monitoring poll timeout |
| `EXECUTION_MAX_RETRIES` | `3` | Max retries on API error (before circuit breaker) |
| `EXECUTION_RETRY_BACKOFF_INITIAL` | `100ms` | Initial retry backoff |
| `EXECUTION_RETRY_BACKOFF_MAX` | `5s` | Max retry backoff |
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |
| `REDIS_URL` | `localhost:6379` | Redis URL (Pit Boss) |
| `POSTGRES_URL` | `postgres://localhost:5432/pqap` | PostgreSQL connection string |
| `POLYMARKET_CLOB_URL` | `https://clob.polymarket.com` | Polymarket CLOB API base URL |
