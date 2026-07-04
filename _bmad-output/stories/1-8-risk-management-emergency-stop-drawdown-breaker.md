# Story 1.8: Risk Management — Emergency Stop & Drawdown Breaker

## Story

As a quant trader,
I want an emergency stop and drawdown circuit breaker that halt all trading on critical failures,
So that a bad market day or system failure doesn't wipe out my capital.

## Status

ready-for-dev

## Acceptance Criteria

### Drawdown Circuit Breaker

**Given** the risk manager is monitoring portfolio drawdown
**When** drawdown exceeds the configurable threshold (default: 10%)
**Then** all trading is halted immediately
**And** all open orders are cancelled
**And** a critical alert is sent via Telegram
**And** manual resume is required to restart trading

### Emergency Stop

**Given** a critical failure is detected (API death spiral, data corruption detected by reconciliation, daily budget exhausted)
**When** the emergency stop is triggered
**Then** all trading is halted within 1 second
**And** all open orders are cancelled
**And** an `EmergencyStop` event is published to NATS
**And** a critical Telegram notification is sent (bypasses all throttling)
**And** the emergency stop reason and full context are logged

## Technical Requirements

### Architecture Context

- **Service:** `risk-manager` (Go)
- **Paradigm:** Event-driven hexagonal architecture (ports & adapters)
- **Event bus:** NATS (JetStream for durable subscriptions)
- **Databases:** Redis (Pit Boss state, ephemeral cache), PostgreSQL (`risk_events` table — sole writer per AD-6)
- **Pit Boss Pattern:** Centralized risk authority — sole authority on whether a trade may proceed (AD-4). Emergency stop flag lives in Pit Boss Redis state. When `emergency_stop = true`, all risk checks return DENY.
- **Drawdown Calculation:** `drawdown = (peak_equity - current_equity) / peak_equity`. Peak equity tracked as running maximum of total capital.
- **Fail-safe:** If Pit Boss state TTL expires (60s), Execution Engine cannot read state → trading halts (AD-8).

### Key Components to Implement

1. **Drawdown Circuit Breaker** (`internal/pitboss/drawdown.go`)
   - Monitors portfolio drawdown continuously
   - Drawdown formula: `drawdown = (peak_equity - current_equity) / peak_equity`
   - `peak_equity` = running maximum of `total_capital` (deposits + realized PnL + unrealized PnL)
   - `current_equity` = current `total_capital`
   - Default threshold: 10% of peak equity (configurable via `RISK_DRAWDOWN_LIMIT_PCT`)
   - When `drawdown >= threshold`:
     1. Set `emergency_stop = true` in Pit Boss state
     2. Write to Redis immediately (don't wait for 30s refresh)
     3. Publish `EmergencyStop` event to NATS with reason "drawdown_exceeded"
     4. Trigger open order cancellation (via Execution Engine)
     5. Send critical Telegram notification (bypasses throttling)
     6. Log to `risk_events` table with full context
   - Subscribes to `PositionUpdated`, `CapitalUpdated` events to track equity changes
   - Publishes `DrawdownWarning` when 80% of threshold reached

2. **Emergency Stop Coordinator** (`internal/emergency/emergency.go`)
   - Central emergency stop logic for all trigger sources
   - Trigger sources:
     - Drawdown circuit breaker tripped
     - API death spiral (API unreachable > 5 minutes, per AD-11)
     - Data corruption detected by reconciliation (persistent mismatches > 3 consecutive, per AD-5)
     - Daily budget exhausted (from Story 1.7)
     - Manual trigger via API (`POST /api/v1/risk/emergency-stop`)
   - Emergency stop flow (must complete within 1 second):
     1. Set `emergency_stop = true` in Pit Boss state (Redis)
     2. Write to Redis immediately (bypass 30s refresh cycle)
     3. Publish `EmergencyStop` event to NATS (`pqap.risk.emergency`)
     4. Publish `NotificationRequest` to NATS for critical Telegram alert
     5. Log emergency stop with reason and full context to `risk_events`
     6. Cancel all open orders (publish `CancelAllOrders` command to NATS)
     7. Update Prometheus metrics
   - Emergency stop is **idempotent** — multiple triggers are safe (deduplicate by checking current state)
   - Manual resume required: `POST /api/v1/risk/resume` (JWT auth required)

3. **Open Order Cancellation** (`internal/emergency/order_cancel.go`)
   - On emergency stop, cancel all open orders immediately
   - Publishes `CancelAllOrders` command to NATS (`pqap.order.cancel_all`)
   - Execution Engine subscribes and cancels all open orders via CLOB API
   - Waits up to 5 seconds for confirmation; logs any orders that couldn't be cancelled
   - Retries cancellation for orphaned orders (reconciliation loop, per AD-5)

4. **Drawdown Tracker** (`internal/pitboss/drawdown_tracker.go`)
   - Tracks peak equity and current equity
   - Subscribes to `PositionUpdated` and `CapitalUpdated` events
   - Calculates drawdown on every equity change
   - Stores state in Redis:
     ```
     pqap:risk:peak_equity       # String — peak equity (USD)
     pqap:risk:current_equity    # String — current equity (USD)
     pqap:risk:drawdown          # String — current drawdown (ratio, 0.0–1.0)
     pqap:risk:drawdown_limit    # String — drawdown threshold (ratio)
     ```
   - Publishes `DrawdownWarning` when drawdown reaches 80% of threshold
   - Publishes `DrawdownReset` when new peak equity is reached

5. **Critical Telegram Alert Bypass** (`internal/emergency/notifier.go`)
   - Emergency stop notifications bypass all throttling rules (FR-82 exception)
   - Sends immediately via `pqap.notification.send` NATS subject
   - Notification includes:
     - Emergency stop reason
     - Current drawdown percentage (if drawdown-triggered)
     - Current equity and peak equity
     - Daily PnL
     - Number of open orders cancelled
     - Timestamp of trigger
   - Notification severity: `CRITICAL`
   - Delivery target: < 5 seconds (NFR-N1)

6. **Resume Handler** (`internal/emergency/resume.go`)
   - Manual resume via `POST /api/v1/risk/resume` (JWT auth required)
   - Resume flow:
     1. Validate JWT token
     2. Clear `emergency_stop` flag in Pit Boss state
     3. Write to Redis immediately
     4. Log resume event to `risk_events`
     5. Publish `TradingResumed` event to NATS
     6. Send info Telegram notification: "Trading resumed"
   - Resume does NOT reset drawdown peak equity — drawdown tracking continues
   - If drawdown is still above threshold when resumed, emergency stop will trigger again immediately

7. **Emergency Stop State in Pit Boss** (update to `internal/pitboss/pitboss.go`)
   - Add `emergency_stop` field to Pit Boss state
   - Add `emergency_stop_reason` field (string)
   - Add `emergency_stop_timestamp` field (UTC TIMESTAMPTZ)
   - Update `Evaluate()` to check `emergency_stop` first (already in Story 1.7)
   - When `emergency_stop = true`, return DENY with reason "emergency_stop"

8. **Emergency Stop API Endpoints** (update to risk-manager API)
   - `POST /api/v1/risk/emergency-stop` — Trigger emergency stop manually (JWT auth)
   - `POST /api/v1/risk/resume` — Resume after emergency stop (JWT auth)
   - `GET /api/v1/risk/emergency-status` — Get current emergency stop status and reason

### Data Models

**EmergencyStopEvent (NATS event):**
```go
type EmergencyStopEvent struct {
    EventID   string                  `json:"event_id"`     // UUID
    EventType string                  `json:"event_type"`   // "EmergencyStop"
    Timestamp time.Time               `json:"timestamp"`    // ISO 8601 UTC
    Source    string                  `json:"source"`       // "risk-manager"
    Payload   EmergencyStopPayload    `json:"payload"`
}

type EmergencyStopPayload struct {
    Reason          string          `json:"reason"`            // "drawdown_exceeded", "api_death_spiral", "data_corruption", "daily_budget_exhausted", "manual"
    Drawdown        *decimal.Decimal `json:"drawdown"`         // current drawdown ratio (nullable if not drawdown-triggered)
    PeakEquity      *decimal.Decimal `json:"peak_equity"`      // peak equity in USD (nullable if not drawdown-triggered)
    CurrentEquity   *decimal.Decimal `json:"current_equity"`   // current equity in USD (nullable if not drawdown-triggered)
    DailyPnL        decimal.Decimal `json:"daily_pnl"`        // daily realized PnL
    OpenOrdersCount int             `json:"open_orders_count"` // number of open orders to cancel
    Context         map[string]interface{} `json:"context"`    // additional trigger context
}
```

**DrawdownWarning (NATS event):**
```go
type DrawdownWarning struct {
    EventID   string                 `json:"event_id"`
    EventType string                 `json:"event_type"` // "DrawdownWarning"
    Timestamp time.Time              `json:"timestamp"`
    Source    string                 `json:"source"`     // "risk-manager"
    Payload   DrawdownWarningPayload `json:"payload"`
}

type DrawdownWarningPayload struct {
    Drawdown      decimal.Decimal `json:"drawdown"`       // current drawdown ratio
    DrawdownLimit decimal.Decimal `json:"drawdown_limit"` // threshold
    PeakEquity    decimal.Decimal `json:"peak_equity"`
    CurrentEquity decimal.Decimal `json:"current_equity"`
    Utilization   float64         `json:"utilization"`    // drawdown / limit
}
```

**TradingResumed (NATS event):**
```go
type TradingResumed struct {
    EventID   string               `json:"event_id"`
    EventType string               `json:"event_type"` // "TradingResumed"
    Timestamp time.Time            `json:"timestamp"`
    Source    string               `json:"source"`     // "risk-manager"
    Payload   TradingResumedPayload `json:"payload"`
}

type TradingResumedPayload struct {
    PreviousReason string    `json:"previous_reason"` // reason for the emergency stop that was cleared
    ResumedBy      string    `json:"resumed_by"`      // user identifier
    ResumedAt      time.Time `json:"resumed_at"`
}
```

**PitBossState (updated fields):**
```go
type PitBossState struct {
    // ... existing fields from Story 1.7 ...
    EmergencyStop          bool       `json:"emergency_stop"`
    EmergencyStopReason    string     `json:"emergency_stop_reason"`
    EmergencyStopTimestamp *time.Time `json:"emergency_stop_timestamp"`
    PeakEquity             decimal.Decimal `json:"peak_equity"`
    CurrentEquity          decimal.Decimal `json:"current_equity"`
    Drawdown               decimal.Decimal `json:"drawdown"`         // ratio 0.0–1.0
    DrawdownLimit          decimal.Decimal `json:"drawdown_limit"`   // threshold ratio
}
```

### Events

**NATS Subjects:**
```
pqap.risk.emergency              # Produced: EmergencyStop
pqap.risk.drawdown_warning       # Produced: DrawdownWarning
pqap.risk.drawdown_reset         # Produced: DrawdownReset (new peak equity)
pqap.risk.trading_resumed        # Produced: TradingResumed
pqap.order.cancel_all            # Produced: CancelAllOrders (emergency order cancellation)
pqap.notification.send           # Produced: NotificationRequest (critical Telegram alert)
pqap.position.updated            # Consumed: PositionUpdated (equity tracking)
pqap.portfolio.capital_updated   # Consumed: CapitalUpdated (equity tracking)
pqap.risk.alert                  # Consumed: RiskAlert (from reconciliation — data corruption trigger)
```

### Prometheus Metrics (AD-17)

```
pqap_risk_drawdown_current                     # Gauge — current drawdown ratio (0.0–1.0)
pqap_risk_drawdown_limit                       # Gauge — drawdown threshold ratio
pqap_risk_drawdown_peak_equity_usd             # Gauge — peak equity
pqap_risk_drawdown_current_equity_usd          # Gauge — current equity
pqap_risk_emergency_stop_total                 # Counter — total emergency stops (by reason)
pqap_risk_emergency_stop_active                # Gauge — 1 if emergency stop active, 0 if not
pqap_risk_emergency_stop_duration_seconds      # Gauge — seconds since emergency stop activated
pqap_risk_order_cancel_latency_ms              # Histogram — order cancellation latency
pqap_risk_order_cancel_total                   # Counter — total orders cancelled by emergency stop
pqap_risk_resume_total                         # Counter — total trading resumes
pqap_risk_drawdown_warning_total               # Counter — total drawdown warnings
```

## Implementation Guide

### Step 1: Database Schema Updates

Add emergency stop fields to `risk_events` context (JSONB):
```sql
-- No schema change needed — context JSONB field captures all emergency stop details
-- Example context for emergency stop:
-- {
--   "trigger": "drawdown_exceeded",
--   "drawdown": "0.1234",
--   "peak_equity": "50000.00",
--   "current_equity": "43830.00",
--   "daily_pnl": "-150.00",
--   "open_orders_cancelled": 3
-- }
```

### Step 2: Drawdown Tracker

- Subscribe to `pqap.position.updated` and `pqap.portfolio.capital_updated` NATS subjects
- On each equity change:
  1. Update `current_equity`
  2. If `current_equity > peak_equity`: update `peak_equity` (new high), publish `DrawdownReset`
  3. Calculate `drawdown = (peak_equity - current_equity) / peak_equity`
  4. Write to Redis: `pqap:risk:drawdown`, `pqap:risk:peak_equity`, `pqap:risk:current_equity`
  5. If `drawdown >= 0.80 * drawdown_limit`: publish `DrawdownWarning`
  6. If `drawdown >= drawdown_limit`: trigger emergency stop
- Update `pqap_risk_drawdown_current` gauge
- Update `pqap_risk_drawdown_peak_equity_usd` gauge
- Update `pqap_risk_drawdown_current_equity_usd` gauge

### Step 3: Emergency Stop Coordinator

- Central function: `func (e *EmergencyStopper) Trigger(reason string, context map[string]interface{})`
- Flow (must complete within 1 second):
  1. Check if already in emergency stop (idempotent — if yes, log and return)
  2. Set `emergency_stop = true` in Pit Boss state
  3. Set `emergency_stop_reason` and `emergency_stop_timestamp`
  4. Write to Redis immediately (`SET pqap:risk:emergency_stop "true" EX 60`)
  5. Publish `EmergencyStop` event to NATS
  6. Publish `CancelAllOrders` command to NATS
  7. Publish `NotificationRequest` to NATS (critical, bypasses throttling)
  8. Log to `risk_events` table
  9. Update Prometheus metrics
- Subscribe to trigger sources:
  - `pqap.risk.alert` (data corruption from reconciliation)
  - Internal drawdown exceeded signal
  - Internal API death spiral signal (from circuit breaker monitoring)
  - Internal daily budget exhausted signal (from Story 1.7)
  - Manual trigger via API

### Step 4: Open Order Cancellation

- Publish `CancelAllOrders` to NATS subject `pqap.order.cancel_all`
- Execution Engine subscribes and:
  1. Fetches all open orders from internal state
  2. Cancels each order via Polymarket CLOB API
  3. Logs cancellation results
  4. Publishes `OrderCancelled` for each cancelled order
- Risk Manager monitors cancellation completion:
  - Waits up to 5 seconds for confirmation
  - Logs any orders that couldn't be cancelled
  - Orphaned orders caught by reconciliation loop (AD-5)

### Step 5: Critical Telegram Alert

- Publish `NotificationRequest` to NATS with:
  - `severity`: "CRITICAL"
  - `bypass_throttling`: true
  - `message`: formatted emergency stop details
- Message format:
  ```
  🚨 EMERGENCY STOP TRIGGERED

  Reason: {reason}
  Time: {timestamp UTC}

  Drawdown: {drawdown}% (limit: {limit}%)
  Peak Equity: ${peak_equity}
  Current Equity: ${current_equity}
  Daily PnL: ${daily_pnl}

  Open Orders Cancelled: {count}

  Manual resume required via dashboard or API.
  ```
- Notification service delivers immediately (bypasses all throttling per FR-82)

### Step 6: Resume Handler

- `POST /api/v1/risk/resume` endpoint:
  1. Validate JWT token (require auth)
  2. Check if emergency stop is active (if not, return 400)
  3. Clear `emergency_stop` flag in Pit Boss state
  4. Clear `emergency_stop_reason` and `emergency_stop_timestamp`
  5. Write to Redis immediately
  6. Log resume event to `risk_events`
  7. Publish `TradingResumed` event to NATS
  8. Publish info `NotificationRequest` to NATS: "Trading resumed"
  9. Return 200 with resume details
- If drawdown is still above threshold after resume, emergency stop will trigger again on next equity check

### Step 7: Update Pit Boss State

- Add to Pit Boss state JSON:
  ```json
  {
    "emergency_stop": false,
    "emergency_stop_reason": "",
    "emergency_stop_timestamp": null,
    "peak_equity": "50000.00",
    "current_equity": "48500.00",
    "drawdown": "0.03",
    "drawdown_limit": "0.10"
  }
  ```
- Update `Evaluate()` to check emergency stop first (already done in Story 1.7)
- When `emergency_stop = true`, all checks return DENY with reason "emergency_stop"

### Step 8: Redis State Keys

Update Redis key structure:
```
pqap:risk:emergency_stop       # String — "true"/"false"
pqap:risk:emergency_reason     # String — trigger reason
pqap:risk:emergency_timestamp  # String — ISO 8601 UTC
pqap:risk:peak_equity          # String — peak equity (USD)
pqap:risk:current_equity       # String — current equity (USD)
pqap:risk:drawdown             # String — current drawdown ratio
pqap:risk:drawdown_limit       # String — threshold ratio
```

### Step 9: State Reconstruction

- On service startup, reconstruct drawdown state from PostgreSQL:
  1. Query `risk_events` for most recent `EmergencyStop` event (check if still active)
  2. Query `risk_events` for most recent `TradingResumed` event (determine current state)
  3. Query `positions` table for current equity
  4. Query `trades` table for peak equity history (running max of daily equity)
  5. Rebuild drawdown state
  6. Write to Redis
- If last event was `EmergencyStop` (no matching `TradingResumed`), system starts in emergency stop state

### Step 10: API Death Spiral Detection

- Monitor circuit breaker state for Polymarket API
- If circuit breaker is open for > 5 minutes (configurable via `RISK_API_TIMEOUT_MINUTES`):
  - Trigger emergency stop with reason "api_death_spiral"
  - Context includes: circuit breaker open duration, consecutive failures
- Subscribe to circuit breaker state changes from Execution Engine

### Step 11: Data Corruption Detection

- Subscribe to `pqap.risk.alert` NATS subject
- Reconciliation services (Scanner, Position Manager, Execution Engine) publish alerts on persistent mismatches
- If persistent mismatches > 3 consecutive (per AD-5):
  - Trigger emergency stop with reason "data_corruption"
  - Context includes: reconciliation type, mismatch details, consecutive count

## Testing

### Unit Tests

- **Drawdown tracker (`drawdown_tracker_test.go`):**
  - Drawdown calculated correctly from equity changes
  - Peak equity updates on new high
  - Warning published at 80% of threshold
  - Emergency stop triggered when threshold exceeded
  - DrawdownReset published on new peak equity

- **Emergency stop coordinator (`emergency_test.go`):**
  - Emergency stop sets flag in Pit Boss state
  - Emergency stop publishes EmergencyStop event
  - Emergency stop triggers order cancellation
  - Emergency stop sends critical Telegram alert
  - Emergency stop completes within 1 second
  - Idempotent — multiple triggers are safe
  - Resume clears flag and publishes TradingResumed

- **Order cancellation (`order_cancel_test.go`):**
  - CancelAllOrders event published on emergency stop
  - Cancellation results logged
  - Timeout after 5 seconds

- **Critical notifier (`notifier_test.go`):**
  - Notification bypasses throttling
  - Message includes all required fields
  - Severity set to CRITICAL

- **Resume handler (`resume_test.go`):**
  - Resume clears emergency stop flag
  - Resume requires JWT auth
  - Resume publishes TradingResumed event
  - Resume does not reset peak equity

### Integration Tests

- **Full emergency stop flow:**
  - Trigger emergency stop via drawdown exceeded
  - Verify all trading halted within 1 second
  - Verify open orders cancelled
  - Verify EmergencyStop event published
  - Verify critical Telegram notification sent
  - Verify reason and context logged to risk_events

- **Drawdown circuit breaker:**
  - Simulate equity decline exceeding threshold
  - Verify drawdown calculated correctly
  - Verify emergency stop triggers at threshold
  - Verify warning at 80% threshold

- **Resume flow:**
  - Trigger emergency stop
  - Resume via API
  - Verify trading resumes
  - Verify drawdown tracking continues (peak not reset)
  - Verify emergency stop re-triggers if drawdown still exceeded

- **API death spiral detection:**
  - Simulate API circuit breaker open for > 5 minutes
  - Verify emergency stop triggered with reason "api_death_spiral"

- **Data corruption detection:**
  - Simulate persistent reconciliation mismatches (> 3)
  - Verify emergency stop triggered with reason "data_corruption"

- **Manual trigger:**
  - Trigger emergency stop via API
  - Verify same flow as automatic triggers

### Test Files

```
tests/unit/risk-manager/
├── drawdown_tracker_test.go       # Drawdown calculation, peak tracking, warning thresholds
├── emergency_test.go              # Emergency stop trigger, idempotency, flow completion
├── order_cancel_test.go           # Order cancellation command, timeout handling
├── notifier_test.go               # Critical notification bypass, message format
├── resume_test.go                 # Resume handler, auth, peak equity behavior
└── drawdown_circuit_breaker_test.go  # Threshold enforcement, integration with Pit Boss

tests/integration/
├── emergency_stop_flow_test.go    # Full emergency stop end-to-end
├── drawdown_circuit_breaker_test.go  # Drawdown monitoring end-to-end
├── resume_flow_test.go            # Resume end-to-end
├── api_death_spiral_test.go       # API timeout trigger end-to-end
├── data_corruption_test.go        # Reconciliation mismatch trigger end-to-end
└── manual_emergency_stop_test.go  # Manual API trigger end-to-end
```

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| Story 1.7 | — | Pit Boss & Daily Budget (prerequisite — Pit Boss state, emergency_stop flag, risk_events table) |
| Story 1.4 | — | Execution Engine — Order Placement (prerequisite — consumes CancelAllOrders command) |
| Story 1.6 | — | Position Tracking & PnL (prerequisite — produces PositionUpdated events for equity tracking) |
| Story 1.10 | — | Telegram Notifications (prerequisite — delivers critical alerts) |
| `github.com/nats-io/nats.go` | latest | NATS event bus |
| `github.com/redis/go-redis/v9` | latest | Redis client for Pit Boss state |
| `github.com/shopspring/decimal` | latest | Decimal precision (monetary values) |
| `github.com/prometheus/client_golang` | latest | Metrics export |
| `go.uber.org/zap` | latest | Structured logging |
| `github.com/google/uuid` | latest | Event ID generation |
| `github.com/jackc/pgx/v5` | latest | PostgreSQL driver |

## External Dependencies (Infrastructure)

| Service | Required | Purpose |
|---------|----------|---------|
| NATS server | Yes | Event bus (consume equity events; produce emergency/resume events) |
| Redis | Yes | Pit Boss state (emergency stop flag, drawdown state) |
| PostgreSQL | Yes | risk_events table (source of truth for emergency stop history) |
| Telegram Bot API | Yes | Critical emergency alerts (via notification service) |

## Definition of Done

- [ ] Drawdown circuit breaker triggers at configurable threshold (default: 10%) (FR-42)
- [ ] Emergency stop halts all trading within 1 second (FR-44)
- [ ] All open orders cancelled on emergency stop (FR-44)
- [ ] EmergencyStop event published to NATS (FR-44, AD-9)
- [ ] Critical Telegram notification sent, bypasses all throttling (FR-44, FR-82)
- [ ] Emergency stop reason and full context logged to risk_events (FR-47, NFR-R4)
- [ ] Manual resume required to restart trading (FR-42, FR-44)
- [ ] API death spiral detection (> 5 min unreachable) triggers emergency stop (AD-11)
- [ ] Data corruption detection (persistent reconciliation mismatches > 3) triggers emergency stop (AD-5)
- [ ] Daily budget exhausted triggers emergency stop (AD-11)
- [ ] Manual emergency stop via API supported (FR-53)
- [ ] Emergency stop state in Pit Boss Redis state (AD-4)
- [ ] State reconstructable from PostgreSQL on restart (AD-8)
- [ ] Prometheus metrics exported (AD-17)
- [ ] Structured JSON logging (INF-14)
- [ ] Decimal precision for all monetary values (INF-11)
- [ ] All timestamps UTC as TIMESTAMPTZ (INF-12)
- [ ] Unit tests pass with ≥80% coverage
- [ ] Integration tests pass

## References

| Reference | Description |
|-----------|-------------|
| FR-42 | System SHALL implement drawdown circuit breaker: halt if drawdown exceeds threshold (default: 10%) |
| FR-44 | System SHALL implement emergency stop: immediate halt on critical failures (API death spiral, data corruption) |
| FR-53 | Dashboard SHALL provide quick actions: emergency stop, pause/resume, risk param adjustment |
| FR-82 | Center SHALL support notification throttling (max 10 per minute for non-critical) — emergency bypasses |
| AD-4 | Pit Boss is the sole authority on whether a trade may proceed; lives as risk state keys in Redis with 60s TTL |
| AD-5 | State Reconciliation: persistent mismatches (> 3 consecutive) trigger emergency stop |
| AD-8 | Redis is ephemeral cache/coordination only; all Redis state reconstructable from PostgreSQL |
| AD-9 | NATS is primary event bus; fire-and-forget with at-least-once delivery; consumers idempotent by event UUID |
| AD-11 | Emergency stop on: API unreachable >5min, data corruption, daily budget exhausted, drawdown breaker tripped |
| AD-17 | Prometheus metrics on /metrics for all services |
| NFR-N1 | Critical notification latency: within 5s |
| NFR-N2 | Critical notification delivery rate: 99.9% |
| NFR-R4 | Risk decision auditability: complete log with full context |
| INF-11 | Decimal precision: all monetary values use Decimal (never float64); prices 4dp, quantities 8dp, PnL 8dp |
| INF-12 | All timestamps UTC as TIMESTAMPTZ; display timezone configurable |
| INF-14 | Structured JSON logs with timestamp, level, service, request_id, message, context |
| INF-15 | Prometheus metric naming: pqap_{service}_{metric_name}_{unit} |
| INF-16 | Event naming: past tense verb + noun (EmergencyStop, DrawdownWarning, TradingResumed) |
| INF-17 | All events include: event_id (UUID), event_type, timestamp (ISO 8601 UTC), source, payload |

## Architecture Decisions Impacting This Story

| Decision | Impact |
|----------|--------|
| AD-4 (Pit Boss) | Emergency stop flag lives in Pit Boss state. When set, all risk checks return DENY. |
| AD-5 (Reconciliation) | Persistent mismatches (> 3 consecutive) trigger emergency stop. Reconciliation alerts feed into emergency coordinator. |
| AD-8 (Redis) | Emergency stop state in Redis with 60s TTL. If TTL expires, trading halts (fail-safe). State reconstructable from PostgreSQL. |
| AD-9 (NATS) | EmergencyStop event published to NATS. CancelAllOrders command published to NATS. All consumers idempotent. |
| AD-10 (Communication) | Emergency stop: async (NATS publish). Resume: sync (API call with JWT auth). |
| AD-11 (Error Handling) | Global emergency stop on: API unreachable >5min, data corruption, daily budget exhausted, drawdown breaker tripped. |
| AD-17 (Observability) | Prometheus metrics for drawdown, emergency stop, order cancellation, resume. |
| INF-11 (Decimal) | All monetary values use `decimal.Decimal` — never `float64`. |
| INF-12 (Time) | All timestamps UTC as `TIMESTAMPTZ`. |

## Directory Structure

```
services/risk-manager/
├── cmd/
│   └── main.go                           # Entry point — starts all components
├── internal/
│   ├── pitboss/
│   │   ├── pitboss.go                    # Core risk evaluation (updated with emergency stop check)
│   │   ├── daily_budget.go               # Daily loss tracking (from Story 1.7)
│   │   ├── position_limit.go             # Per-market position limits (from Story 1.7)
│   │   ├── strategy_limit.go             # Per-strategy position limits (from Story 1.7)
│   │   ├── drawdown.go                   # Drawdown circuit breaker
│   │   ├── drawdown_tracker.go           # Peak equity tracking, drawdown calculation
│   │   ├── redis_writer.go               # Redis state writer (updated with drawdown fields)
│   │   ├── reconstructor.go              # State reconstruction (updated with drawdown state)
│   │   ├── logger.go                     # Risk event logging (updated with emergency context)
│   │   └── check.go                      # Synchronous risk check endpoint
│   ├── emergency/
│   │   ├── emergency.go                  # Emergency stop coordinator
│   │   ├── order_cancel.go               # Open order cancellation on emergency stop
│   │   ├── notifier.go                   # Critical Telegram alert bypass
│   │   └── resume.go                     # Resume handler
│   └── ports/
│       ├── risk_state.go                 # RiskStatePort interface (Redis)
│       └── event.go                      # EventPort interface (NATS)
├── adapters/
│   ├── redis_writer.go                   # Redis adapter (updated with emergency/drawdown keys)
│   ├── postgres_repo.go                  # PostgreSQL adapter (risk_events)
│   ├── nats_subscriber.go                # NATS subscriber (equity events, reconciliation alerts)
│   └── nats_publisher.go                 # NATS publisher (emergency, drawdown, resume events)
├── config/
│   └── config.go                         # Configuration (updated with drawdown/emergency params)
├── metrics/
│   └── metrics.go                        # Prometheus metrics (updated with drawdown/emergency metrics)
└── go.mod
```

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `RISK_DRAWDOWN_LIMIT_PCT` | `0.10` | Drawdown circuit breaker threshold as ratio (10%) |
| `RISK_DRAWDOWN_WARNING_THRESHOLD` | `0.80` | Utilization threshold for drawdown warning (80% of limit) |
| `RISK_API_TIMEOUT_MINUTES` | `5` | Minutes API must be unreachable before emergency stop |
| `RISK_RECON_MISMATCH_LIMIT` | `3` | Consecutive reconciliation mismatches before emergency stop |
| `RISK_ORDER_CANCEL_TIMEOUT` | `5s` | Max time to wait for order cancellation confirmation |
| `RISK_EMERGENCY_STOP_TTL` | `60s` | TTL for emergency stop Redis key |
| `RISK_DAILY_LOSS_LIMIT_PCT` | `0.02` | Daily loss limit as % of capital (from Story 1.7) |
| `RISK_MARKET_LIMIT_PCT` | `0.10` | Max position per market as % of capital (from Story 1.7) |
| `RISK_STRATEGY_LIMIT_PCT` | `0.20` | Max position per strategy as % of capital (from Story 1.7) |
| `RISK_STATE_REFRESH_INTERVAL` | `30s` | How often to refresh Pit Boss state in Redis (from Story 1.7) |
| `RISK_STATE_TTL` | `60s` | TTL for Pit Boss state keys in Redis (from Story 1.7) |
| `CAPITAL_TOTAL` | `10000.00` | Total capital in USD (used for limit calculations) |
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection string |
| `POSTGRES_URL` | `postgres://localhost:5432/pqap` | PostgreSQL connection string |
| `JWT_SECRET` | — | JWT secret for API authentication (resume/emergency-stop endpoints) |
