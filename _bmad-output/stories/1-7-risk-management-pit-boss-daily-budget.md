# Story 1.7: Risk Management — Pit Boss & Daily Budget

## Story

As a quant trader,
I want a centralized Pit Boss that enforces daily loss limits and position limits before every trade,
So that the bot never loses more than I can afford in a single day.

## Status

ready-for-dev

## Acceptance Criteria

**Given** the risk manager service is running
**When** it initializes and periodically (every 30 seconds)
**Then** Pit Boss risk state keys are written to Redis with a 60-second TTL
**And** the state includes: daily budget remaining, max position per market (default: 10% of capital), max position per strategy (default: 20% of capital)

**Given** the Pit Boss state is in Redis
**When** the execution engine performs a synchronous risk check before a trade
**Then** the check completes within 10ms
**And** if daily loss limit (default: 2% of capital) is reached, the Pit Boss returns DENY
**And** if per-market position limit would be exceeded, the Pit Boss returns DENY
**And** if per-strategy position limit would be exceeded, the Pit Boss returns DENY
**And** all risk decisions (approve/deny) are logged to PostgreSQL `risk_events` table with full context
**And** the Pit Boss state in Redis is reconstructable from PostgreSQL on restart

## Technical Requirements

### Architecture Context

- **Service:** `risk-manager` (Go)
- **Paradigm:** Event-driven hexagonal architecture (ports & adapters)
- **Event bus:** NATS (JetStream for durable subscriptions)
- **Databases:** Redis (Pit Boss state, ephemeral cache), PostgreSQL (`risk_events` table — sole writer per AD-6)
- **Pit Boss Pattern:** Centralized risk authority — sole authority on whether a trade may proceed (AD-4). Lives as risk state keys in Redis with 60s TTL, refreshed every 30s. Synchronous GET by Execution Engine (<10ms). If TTL expires, trading halts (fail-safe).
- **Reconstruction:** All Redis state reconstructable from PostgreSQL on restart (AD-8). No data lives only in Redis.
- **Single Writer:** `risk_events` table written by Risk Management service only (AD-6).

### Key Components to Implement

1. **Pit Boss Engine** (`internal/pitboss/pitboss.go`)
   - Central risk evaluation engine
   - Evaluates all risk checks before returning ALLOW/DENY
   - Risk checks evaluated in order (fail-fast on first DENY):
     1. Daily budget remaining (2% of capital limit)
     2. Per-market position limit (10% of capital)
     3. Per-strategy position limit (20% of capital)
   - Returns `RiskDecision` with: decision (ALLOW/DENY), reason, context
   - All decisions logged to PostgreSQL `risk_events` table
   - Publishes `RiskStateUpdated` event to NATS after state changes
   - Publishes `RiskAlert` event when limits approached (80% threshold)

2. **Daily Budget Manager** (`internal/pitboss/daily_budget.go`)
   - Tracks daily realized PnL (sum of all closed/settled positions today)
   - Calculates daily budget remaining: `daily_budget_limit - abs(daily_loss)`
   - Default daily loss limit: 2% of total capital (configurable via `RISK_DAILY_LOSS_LIMIT_PCT`)
   - Resets at midnight UTC (new trading day)
   - Denies trades when daily loss limit reached
   - Monitors via `OrderFilled`, `PositionClosed` events from NATS
   - Publishes `DailyBudgetWarning` when 80% consumed

3. **Market Position Limiter** (`internal/pitboss/position_limit.go`)
   - Tracks total exposure per market across all positions
   - Calculates: `market_exposure = sum(position.quantity * position.current_price)` for all positions in market
   - Default limit: 10% of total capital (configurable via `RISK_MARKET_LIMIT_PCT`)
   - Denies trades that would exceed per-market limit
   - Updates on `PositionOpened`, `PositionClosed`, `PositionUpdated` events

4. **Strategy Position Limiter** (`internal/pitboss/strategy_limit.go`)
   - Tracks total exposure per strategy across all positions
   - Calculates: `strategy_exposure = sum(position.quantity * position.current_price)` for all positions in strategy
   - Default limit: 20% of total capital (configurable via `RISK_STRATEGY_LIMIT_PCT`)
   - Denies trades that would exceed per-strategy limit
   - Updates on `PositionOpened`, `PositionClosed`, `PositionUpdated` events

5. **Redis State Writer** (`internal/pitboss/redis_writer.go`)
   - Writes Pit Boss risk state to Redis every 30 seconds (configurable)
   - State keys with 60-second TTL (fail-safe: if TTL expires, trading halts)
   - Redis key structure:
     ```
     pqap:risk:state           # JSON — full Pit Boss state
     pqap:risk:daily_budget    # String — daily budget remaining (USD)
     pqap:risk:daily_loss      # String — daily realized loss (USD)
     pqap:risk:market:{id}     # String — market exposure (USD)
     pqap:risk:strategy:{id}   # String — strategy exposure (USD)
     pqap:risk:emergency_stop  # String — "true"/"false"
     ```
   - Pit Boss state JSON structure:
     ```json
     {
       "daily_budget_remaining": "980.00",
       "daily_loss": "20.00",
       "daily_loss_limit": "1000.00",
       "capital": "50000.00",
       "market_limits": {
         "market_123": { "exposure": "4500.00", "limit": "5000.00", "utilization": 0.90 },
         "market_456": { "exposure": "2000.00", "limit": "5000.00", "utilization": 0.40 }
       },
       "strategy_limits": {
         "simple_arb": { "exposure": "8000.00", "limit": "10000.00", "utilization": 0.80 },
         "cross_market": { "exposure": "3000.00", "limit": "10000.00", "utilization": 0.30 }
       },
       "emergency_stop": false,
       "updated_at": "2026-07-04T12:00:00Z"
     }
     ```
   - All monetary values use `decimal.Decimal` — never `float64` (INF-11)

6. **Risk Check Endpoint** (`internal/pitboss/check.go`)
   - Synchronous risk check for Execution Engine
   - Called via Redis GET (Execution Engine reads Pit Boss state directly from Redis)
   - Response time target: <10ms (NFR-R1)
   - The Execution Engine reads the pre-computed state from Redis and evaluates:
     - Is `emergency_stop` true? → DENY
     - Is `daily_budget_remaining` <= 0? → DENY
     - Would this trade exceed `market_limits[market_id].limit`? → DENY
     - Would this trade exceed `strategy_limits[strategy_id].limit`? → DENY
   - Returns ALLOW or DENY with reason string
   - All check results logged to `risk_events` table (async, after response)

7. **State Reconstructor** (`internal/pitboss/reconstructor.go`)
   - On service startup, reconstructs Pit Boss state from PostgreSQL
   - Queries `risk_events` table for current day's decisions
   - Queries `positions` table (via API or direct read) for current exposures
   - Queries `trades` table for daily realized PnL
   - Rebuilds Redis state from PostgreSQL data
   - Reconstruction completes before accepting new risk checks
   - Logs reconstruction with: source_count, duration, state_snapshot

8. **Risk Event Logger** (`internal/pitboss/logger.go`)
   - Logs every risk decision to PostgreSQL `risk_events` table
   - Single writer for `risk_events` table (AD-6)
   - Log entry includes:
     - `event_id` (UUID)
     - `timestamp` (UTC TIMESTAMPTZ)
     - `decision` (ALLOW/DENY)
     - `reason` (string — "daily_limit", "market_limit", "strategy_limit", "emergency_stop", "approved")
     - `market_id` (nullable)
     - `strategy_id` (nullable)
     - `trade_size` (NUMERIC — proposed trade size in USD)
     - `current_exposure` (NUMERIC — current exposure before trade)
     - `limit_value` (NUMERIC — limit that was checked)
     - `daily_budget_remaining` (NUMERIC)
     - `capital` (NUMERIC)
     - `context` (JSONB — additional context)
     - `account_id` (nullable, for future multi-account)
   - Append-only, immutable (no UPDATE/DELETE)
   - Queryable by: date range, decision, reason, market_id, strategy_id

### Data Models

**RiskDecision (internal domain model):**
```go
type RiskDecision struct {
    EventID             string          `json:"event_id"`               // UUID
    Timestamp           time.Time       `json:"timestamp"`              // UTC TIMESTAMPTZ
    Decision            string          `json:"decision"`               // "ALLOW" or "DENY"
    Reason              string          `json:"reason"`                 // "daily_limit", "market_limit", "strategy_limit", "emergency_stop", "approved"
    MarketID            *string         `json:"market_id"`              // nullable
    StrategyID          *string         `json:"strategy_id"`            // nullable
    TradeSize           decimal.Decimal `json:"trade_size"`             // 8dp — proposed trade size in USD
    CurrentExposure     decimal.Decimal `json:"current_exposure"`       // 8dp — current exposure before trade
    LimitValue          decimal.Decimal `json:"limit_value"`            // 8dp — limit that was checked
    DailyBudgetRemaining decimal.Decimal `json:"daily_budget_remaining"` // 8dp
    Capital             decimal.Decimal `json:"capital"`                // 8dp
    Context             map[string]interface{} `json:"context"`         // additional context
    AccountID           *string         `json:"account_id"`             // nullable, for future multi-account
}
```

**PitBossState (Redis state):**
```go
type PitBossState struct {
    DailyBudgetRemaining decimal.Decimal        `json:"daily_budget_remaining"`
    DailyLoss            decimal.Decimal        `json:"daily_loss"`
    DailyLossLimit       decimal.Decimal        `json:"daily_loss_limit"`
    Capital              decimal.Decimal        `json:"capital"`
    MarketLimits         map[string]LimitEntry  `json:"market_limits"`
    StrategyLimits       map[string]LimitEntry  `json:"strategy_limits"`
    EmergencyStop        bool                   `json:"emergency_stop"`
    UpdatedAt            time.Time              `json:"updated_at"`
}

type LimitEntry struct {
    Exposure  decimal.Decimal `json:"exposure"`
    Limit     decimal.Decimal `json:"limit"`
    Utilization float64       `json:"utilization"` // exposure / limit
}
```

**RiskCheckRequest (from Execution Engine):**
```go
type RiskCheckRequest struct {
    MarketID   string          `json:"market_id"`
    StrategyID string          `json:"strategy_id"`
    TradeSize  decimal.Decimal `json:"trade_size"`   // USD value of proposed trade
    Side       string          `json:"side"`          // "YES" or "NO"
}
```

**RiskCheckResponse (to Execution Engine):**
```go
type RiskCheckResponse struct {
    Decision string `json:"decision"` // "ALLOW" or "DENY"
    Reason   string `json:"reason"`   // human-readable reason
}
```

### Events

**RiskStateUpdated Event:**
```go
type RiskStateUpdated struct {
    EventID   string                  `json:"event_id"`
    EventType string                  `json:"event_type"` // "RiskStateUpdated"
    Timestamp time.Time               `json:"timestamp"`
    Source    string                  `json:"source"`     // "risk-manager"
    Payload   RiskStateUpdatedPayload `json:"payload"`
}

type RiskStateUpdatedPayload struct {
    DailyBudgetRemaining decimal.Decimal `json:"daily_budget_remaining"`
    DailyLoss            decimal.Decimal `json:"daily_loss"`
    Capital              decimal.Decimal `json:"capital"`
    MarketCount          int             `json:"market_count"`
    StrategyCount        int             `json:"strategy_count"`
    EmergencyStop        bool            `json:"emergency_stop"`
}
```

**RiskDecisionLogged Event:**
```go
type RiskDecisionLogged struct {
    EventID   string                      `json:"event_id"`
    EventType string                      `json:"event_type"` // "RiskDecisionLogged"
    Timestamp time.Time                   `json:"timestamp"`
    Source    string                      `json:"source"`     // "risk-manager"
    Payload   RiskDecisionLoggedPayload   `json:"payload"`
}

type RiskDecisionLoggedPayload struct {
    DecisionID          string          `json:"decision_id"`
    Decision            string          `json:"decision"`
    Reason              string          `json:"reason"`
    MarketID            *string         `json:"market_id"`
    StrategyID          *string         `json:"strategy_id"`
    TradeSize           decimal.Decimal `json:"trade_size"`
    DailyBudgetRemaining decimal.Decimal `json:"daily_budget_remaining"`
}
```

**DailyBudgetWarning Event:**
```go
type DailyBudgetWarning struct {
    EventID   string                      `json:"event_id"`
    EventType string                      `json:"event_type"` // "DailyBudgetWarning"
    Timestamp time.Time                   `json:"timestamp"`
    Source    string                      `json:"source"`     // "risk-manager"
    Payload   DailyBudgetWarningPayload   `json:"payload"`
}

type DailyBudgetWarningPayload struct {
    DailyLoss            decimal.Decimal `json:"daily_loss"`
    DailyLossLimit       decimal.Decimal `json:"daily_loss_limit"`
    Utilization          float64         `json:"utilization"` // daily_loss / daily_loss_limit
    BudgetRemaining      decimal.Decimal `json:"budget_remaining"`
}
```

### API Endpoints

**Internal API (produced):**
- `GET /api/v1/risk/state` — Get current Pit Boss state (read from Redis)
- `GET /api/v1/risk/daily-budget` — Get daily budget status
- `GET /api/v1/risk/limits` — Get all position limits (market + strategy)
- `GET /api/v1/risk/events` — Query risk decision log (filterable by date, decision, reason)
- `POST /api/v1/risk/emergency-stop` — Trigger emergency stop (requires JWT auth)
- `POST /api/v1/risk/resume` — Resume after emergency stop (requires JWT auth)
- `PUT /api/v1/risk/config` — Update risk parameters (requires JWT auth)

**NATS Subjects:**
```
pqap.order.filled                # Consumed: OrderFilled (from execution-engine)
pqap.position.opened             # Consumed: PositionOpened (from position-manager)
pqap.position.closed             # Consumed: PositionClosed (from position-manager)
pqap.position.updated            # Consumed: PositionUpdated (from position-manager)
pqap.risk.state_updated          # Produced: RiskStateUpdated
pqap.risk.decision_logged        # Produced: RiskDecisionLogged
pqap.risk.alert                  # Produced: RiskAlert (limit warnings)
pqap.risk.emergency              # Produced: EmergencyStop
pqap.risk.daily_budget_warning   # Produced: DailyBudgetWarning
pqap.notification.send           # Produced: NotificationRequest (Telegram alerts)
```

### Prometheus Metrics (AD-17)

```
pqap_risk_check_latency_ms                  # Histogram — risk check latency (target: <10ms)
pqap_risk_check_total                       # Counter — total risk checks performed
pqap_risk_check_denied_total                # Counter — total risk checks denied (by reason)
pqap_risk_daily_budget_remaining_usd        # Gauge — daily budget remaining
pqap_risk_daily_loss_usd                    # Gauge — daily realized loss
pqap_risk_daily_loss_limit_usd              # Gauge — daily loss limit
pqap_risk_market_exposure_usd               # Gauge — per-market exposure (labels: market_id)
pqap_risk_strategy_exposure_usd             # Gauge — per-strategy exposure (labels: strategy_id)
pqap_risk_market_utilization                # Gauge — per-market utilization ratio (labels: market_id)
pqap_risk_strategy_utilization              # Gauge — per-strategy utilization ratio (labels: strategy_id)
pqap_risk_state_refresh_total               # Counter — total Redis state refreshes
pqap_risk_state_refresh_latency_ms          # Histogram — Redis state write latency
pqap_risk_reconstruction_duration_ms        # Histogram — state reconstruction time on startup
pqap_risk_emergency_stop_total              # Counter — total emergency stops triggered
pqap_risk_events_logged_total               # Counter — total risk events logged to PostgreSQL
```

## Implementation Guide

### Step 1: Database Schema

Create `risk_events` table:
```sql
CREATE TABLE risk_events (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision                TEXT NOT NULL,            -- "ALLOW" or "DENY"
    reason                  TEXT NOT NULL,            -- "daily_limit", "market_limit", "strategy_limit", "emergency_stop", "approved"
    market_id               TEXT DEFAULT NULL,
    strategy_id             TEXT DEFAULT NULL,
    trade_size              NUMERIC(18,8) NOT NULL DEFAULT 0,
    current_exposure        NUMERIC(18,8) NOT NULL DEFAULT 0,
    limit_value             NUMERIC(18,8) NOT NULL DEFAULT 0,
    daily_budget_remaining  NUMERIC(18,8) NOT NULL,
    capital                 NUMERIC(18,8) NOT NULL,
    context                 JSONB DEFAULT '{}',
    account_id              UUID DEFAULT NULL,        -- nullable, for future multi-account
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_risk_events_created_at ON risk_events(created_at);
CREATE INDEX idx_risk_events_decision ON risk_events(decision);
CREATE INDEX idx_risk_events_reason ON risk_events(reason);
CREATE INDEX idx_risk_events_market_id ON risk_events(market_id);
CREATE INDEX idx_risk_events_strategy_id ON risk_events(strategy_id);
CREATE INDEX idx_risk_events_account_id ON risk_events(account_id);
```

### Step 2: Pit Boss Engine — Risk Evaluation

- Core risk evaluation function:
  ```go
  func (pb *PitBoss) Evaluate(req RiskCheckRequest) RiskDecision {
      // 1. Check emergency stop
      if pb.state.EmergencyStop {
          return pb.deny(req, "emergency_stop")
      }

      // 2. Check daily budget
      if pb.state.DailyBudgetRemaining.LessThanOrEqual(decimal.Zero) {
          return pb.deny(req, "daily_limit")
      }

      // 3. Check per-market limit
      marketExposure := pb.state.MarketLimits[req.MarketID].Exposure
      marketLimit := pb.state.MarketLimits[req.MarketID].Limit
      if marketExposure.Add(req.TradeSize).GreaterThan(marketLimit) {
          return pb.deny(req, "market_limit")
      }

      // 4. Check per-strategy limit
      strategyExposure := pb.state.StrategyLimits[req.StrategyID].Exposure
      strategyLimit := pb.state.StrategyLimits[req.StrategyID].Limit
      if strategyExposure.Add(req.TradeSize).GreaterThan(strategyLimit) {
          return pb.deny(req, "strategy_limit")
      }

      // 5. All checks passed
      return pb.allow(req)
  }
  ```
- All decisions logged to `risk_events` table (async, after response)
- Publish `RiskDecisionLogged` event to NATS

### Step 3: Daily Budget Manager

- Subscribe to `pqap.order.filled` and `pqap.position.closed` NATS subjects
- Track daily realized PnL:
  - On `PositionClosed`: add `realized_pnl` to daily loss if negative
  - Calculate: `daily_budget_remaining = daily_loss_limit - abs(daily_loss)`
- Default daily loss limit: 2% of total capital (configurable)
- Reset at midnight UTC (new trading day)
- Deny trades when `daily_budget_remaining <= 0`
- Publish `DailyBudgetWarning` when 80% consumed:
  - `utilization = abs(daily_loss) / daily_loss_limit`
  - If `utilization >= 0.80`: publish warning, send Telegram notification
- Update `pqap_risk_daily_budget_remaining_usd` gauge
- Update `pqap_risk_daily_loss_usd` gauge

### Step 4: Market Position Limiter

- Subscribe to `pqap.position.opened`, `pqap.position.closed`, `pqap.position.updated` NATS subjects
- Track exposure per market:
  - `market_exposure = sum(position.quantity * position.current_price)` for all open positions in market
  - Update on every position change event
- Default limit: 10% of total capital (configurable via `RISK_MARKET_LIMIT_PCT`)
- Deny trades that would exceed limit:
  - `if market_exposure + trade_size > limit: DENY`
- Update `pqap_risk_market_exposure_usd` gauge per market
- Update `pqap_risk_market_utilization` gauge per market

### Step 5: Strategy Position Limiter

- Subscribe to `pqap.position.opened`, `pqap.position.closed`, `pqap.position.updated` NATS subjects
- Track exposure per strategy:
  - `strategy_exposure = sum(position.quantity * position.current_price)` for all open positions in strategy
  - Update on every position change event
- Default limit: 20% of total capital (configurable via `RISK_STRATEGY_LIMIT_PCT`)
- Deny trades that would exceed limit:
  - `if strategy_exposure + trade_size > limit: DENY`
- Update `pqap_risk_strategy_exposure_usd` gauge per strategy
- Update `pqap_risk_strategy_utilization` gauge per strategy

### Step 6: Redis State Writer

- Start refresh goroutine on service startup
- Every 30 seconds (configurable via `RISK_STATE_REFRESH_INTERVAL`):
  1. Calculate current state:
     - Daily budget remaining from daily budget manager
     - Market exposures from position limiter
     - Strategy exposures from strategy limiter
     - Emergency stop flag
  2. Write to Redis with 60-second TTL:
     ```
     SET pqap:risk:state <json> EX 60
     SET pqap:risk:daily_budget <value> EX 60
     SET pqap:risk:daily_loss <value> EX 60
     SET pqap:risk:emergency_stop <value> EX 60
     ```
  3. Write per-market and per-strategy keys:
     ```
     SET pqap:risk:market:<id> <exposure> EX 60
     SET pqap:risk:strategy:<id> <exposure> EX 60
     ```
  4. Publish `RiskStateUpdated` event to NATS
  5. Update `pqap_risk_state_refresh_total` counter
  6. Record refresh latency in `pqap_risk_state_refresh_latency_ms` histogram
- Fail-safe: if TTL expires, Execution Engine sees stale/missing state → trading halts

### Step 7: Synchronous Risk Check (<10ms)

- The Execution Engine performs risk check by reading Pit Boss state directly from Redis
- Risk check flow (in Execution Engine):
  1. `GET pqap:risk:state` from Redis (<1ms)
  2. Parse JSON state
  3. Evaluate checks (in-memory, <1ms):
     - Is `emergency_stop` true? → DENY
     - Is `daily_budget_remaining <= 0`? → DENY
     - Would trade exceed `market_limits[market_id].limit`? → DENY
     - Would trade exceed `strategy_limits[strategy_id].limit`? → DENY
  4. Return decision (<10ms total)
- The Risk Manager writes pre-computed state; Execution Engine reads and evaluates
- Risk check result logged to `risk_events` table asynchronously (after order decision)

### Step 8: State Reconstruction on Restart

- On service startup, before accepting risk checks:
  1. Query `risk_events` table for today's decisions
  2. Query `positions` table for current open positions
  3. Query `trades` table for today's realized PnL
  4. Rebuild state:
     - Calculate daily loss from closed positions today
     - Calculate market exposures from open positions
     - Calculate strategy exposures from open positions
     - Check for emergency stop events today
  5. Write reconstructed state to Redis
  6. Log reconstruction with: source_count, duration, state_snapshot
  7. Update `pqap_risk_reconstruction_duration_ms` histogram
- Reconstruction must complete before service is healthy

### Step 9: Emergency Stop Integration

- Subscribe to `pqap.risk.emergency` NATS subject (from other services)
- On emergency stop:
  1. Set `emergency_stop = true` in Pit Boss state
  2. Write to Redis immediately (don't wait for 30s refresh)
  3. All subsequent risk checks return DENY with reason "emergency_stop"
  4. Log to `risk_events` table
  5. Send critical Telegram notification
  6. Update `pqap_risk_emergency_stop_total` counter
- Manual resume via API: `POST /api/v1/risk/resume`
- Resume clears `emergency_stop` flag and logs resume event

### Step 10: Event Publishing

- All events follow standard schema (INF-17):
  - `event_id`: UUID v4
  - `event_type`: past tense verb + noun (INF-16)
  - `timestamp`: ISO 8601 UTC
  - `source`: "risk-manager"
  - `payload`: event-specific JSON
- Events produced:
  - `RiskStateUpdated` → `pqap.risk.state_updated`
  - `RiskDecisionLogged` → `pqap.risk.decision_logged`
  - `DailyBudgetWarning` → `pqap.risk.daily_budget_warning`
  - `RiskAlert` → `pqap.risk.alert`
  - `EmergencyStop` → `pqap.risk.emergency`
  - `NotificationRequest` → `pqap.notification.send`
- Fire-and-forget with at-least-once delivery (AD-9)
- Consumers idempotent by event_id

## Testing

### Unit Tests

- **Pit Boss engine (`pitboss_test.go`):**
  - ALLOW when all limits within bounds
  - DENY when daily budget exhausted
  - DENY when market position limit would be exceeded
  - DENY when strategy position limit would be exceeded
  - DENY when emergency stop active
  - All decisions include correct reason
  - All decisions logged to risk_events

- **Daily budget manager (`daily_budget_test.go`):**
  - Daily loss tracked correctly from PositionClosed events
  - Budget remaining calculated correctly
  - Warning published at 80% utilization
  - Reset at midnight UTC
  - Deny when budget exhausted

- **Market position limiter (`position_limit_test.go`):**
  - Exposure calculated correctly from positions
  - Deny when trade would exceed limit
  - Allow when trade within limit
  - Exposure updates on position changes

- **Strategy position limiter (`strategy_limit_test.go`):**
  - Exposure calculated correctly from positions
  - Deny when trade would exceed limit
  - Allow when trade within limit
  - Exposure updates on position changes

- **Redis state writer (`redis_writer_test.go`):**
  - State written every 30s
  - TTL set to 60s
  - All keys present in state JSON
  - State reflects current limits

- **State reconstructor (`reconstructor_test.go`):**
  - State reconstructed from PostgreSQL on startup
  - Daily loss calculated from trades table
  - Market exposures calculated from positions table
  - Strategy exposures calculated from positions table
  - Reconstruction logged with duration

- **Risk event logger (`logger_test.go`):**
  - All decisions logged to risk_events table
  - All required fields populated
  - No NULL values for required fields
  - Append-only (no updates/deletes)

### Integration Tests

- **Risk check flow (<10ms):**
  - Write Pit Boss state to Redis
  - Execute risk check from Execution Engine perspective
  - Verify response time <10ms
  - Verify correct ALLOW/DENY decision

- **Daily budget enforcement:**
  - Create positions with losses exceeding daily limit
  - Verify DENY with reason "daily_limit"
  - Verify Telegram alert sent
  - Verify daily_budget_warning event published

- **Market limit enforcement:**
  - Create positions approaching market limit
  - Verify DENY when trade would exceed limit
  - Verify correct market_id in denial reason

- **Strategy limit enforcement:**
  - Create positions approaching strategy limit
  - Verify DENY when trade would exceed limit
  - Verify correct strategy_id in denial reason

- **State reconstruction:**
  - Create risk events and positions in PostgreSQL
  - Restart risk-manager service
  - Verify state reconstructed correctly
  - Verify Redis state matches PostgreSQL data

- **Emergency stop flow:**
  - Trigger emergency stop
  - Verify all subsequent checks return DENY
  - Verify critical Telegram alert sent
  - Resume and verify checks return ALLOW

### Test Files

```
tests/unit/risk-manager/
├── pitboss_test.go                 # Core risk evaluation (ALLOW/DENY logic)
├── daily_budget_test.go            # Daily loss tracking and limit enforcement
├── position_limit_test.go          # Per-market position limit enforcement
├── strategy_limit_test.go          # Per-strategy position limit enforcement
├── redis_writer_test.go            # Redis state writing (TTL, refresh interval)
├── reconstructor_test.go           # State reconstruction from PostgreSQL
├── logger_test.go                  # Risk event logging to PostgreSQL
└── check_test.go                   # Synchronous risk check latency

tests/integration/
├── risk_check_flow_test.go         # Full risk check flow (<10ms)
├── risk_daily_budget_test.go       # Daily budget enforcement end-to-end
├── risk_market_limit_test.go       # Market limit enforcement end-to-end
├── risk_strategy_limit_test.go     # Strategy limit enforcement end-to-end
├── risk_reconstruction_test.go     # State reconstruction on restart
└── risk_emergency_stop_test.go     # Emergency stop flow end-to-end
```

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| Story 1.1 | — | Scanner WebSocket Connection & Market Catalog (prerequisite — produces MarketPriceUpdated) |
| Story 1.4 | — | Execution Engine — Order Placement & Slippage Protection (prerequisite — consumes Pit Boss state) |
| Story 1.6 | — | Position Tracking & PnL Calculation (prerequisite — produces PositionOpened/Closed/Updated) |
| Story 1.10 | — | Telegram Notifications (prerequisite — delivers risk alerts) |
| `github.com/nats-io/nats.go` | latest | NATS event bus |
| `github.com/redis/go-redis/v9` | latest | Redis client for Pit Boss state |
| `github.com/shopspring/decimal` | latest | Decimal precision (monetary values) |
| `github.com/prometheus/client_golang` | latest | Metrics export |
| `go.uber.org/zap` | latest | Structured logging |
| `github.com/google/uuid` | latest | Event ID, decision ID generation |
| `github.com/jackc/pgx/v5` | latest | PostgreSQL driver |

## External Dependencies (Infrastructure)

| Service | Required | Purpose |
|---------|----------|---------|
| NATS server | Yes | Event bus (consume position/order events; produce risk events) |
| Redis | Yes | Pit Boss state (ephemeral cache, 60s TTL, reconstructable) |
| PostgreSQL | Yes | risk_events table (source of truth for risk decisions) |
| Telegram Bot API | Yes | Risk alerts, daily budget warnings (via notification service) |

## Definition of Done

- [ ] Pit Boss risk state written to Redis every 30s with 60s TTL (FR-46, AD-4, AD-8)
- [ ] State includes: daily budget remaining, market limits, strategy limits (FR-38, FR-39)
- [ ] Risk check completes within 10ms (NFR-R1)
- [ ] Daily loss limit (2% of capital) enforced (FR-38)
- [ ] Per-market position limit (10% of capital) enforced (FR-39)
- [ ] Per-strategy position limit (20% of capital) enforced (FR-33)
- [ ] All risk decisions logged to PostgreSQL risk_events table (FR-47, NFR-R4)
- [ ] Pit Boss state reconstructable from PostgreSQL on restart (AD-8)
- [ ] Emergency stop integration (FR-44)
- [ ] Prometheus metrics exported (AD-17)
- [ ] Structured JSON logging (INF-14)
- [ ] Decimal precision for all monetary values (INF-11)
- [ ] All timestamps UTC as TIMESTAMPTZ (INF-12)
- [ ] account_id nullable column in risk_events table (INF-18)
- [ ] Unit tests pass with ≥80% coverage
- [ ] Integration tests pass

## References

| Reference | Description |
|-----------|-------------|
| FR-38 | System SHALL enforce daily loss limit (default: 2% of capital, configurable) |
| FR-39 | System SHALL enforce max position per market (default: 10% of capital, configurable) |
| FR-45 | Pit Boss SHALL be consulted before every trade; trades rejected if Pit Boss returns deny |
| FR-46 | System SHALL maintain risk state in Redis for cross-component access |
| FR-47 | System SHALL log all risk decisions (approve/deny) with full context |
| AD-4 | Pit Boss is the sole authority on whether a trade may proceed; lives as risk state keys in Redis with 60s TTL |
| AD-8 | Redis is ephemeral cache/coordination only; all Redis state reconstructable from PostgreSQL |
| AD-6 | Data Ownership: risk_events table written by Risk Management only |
| NFR-R1 | Risk check latency: within 10ms of trade request |
| NFR-R2 | Risk state consistency: via Redis across all components |
| NFR-R3 | Pit Boss availability: 99.99% |
| NFR-R4 | Risk decision auditability: complete log with full context |
| INF-11 | Decimal precision: all monetary values use Decimal (never float64); prices 4dp, quantities 8dp, PnL 8dp |
| INF-12 | All timestamps UTC as TIMESTAMPTZ; display timezone configurable |
| INF-14 | Structured JSON logs with timestamp, level, service, request_id, message, context |
| INF-15 | Prometheus metric naming: pqap_{service}_{metric_name}_{unit} |
| INF-16 | Event naming: past tense verb + noun (RiskStateUpdated, RiskDecisionLogged) |
| INF-17 | All events include: event_id (UUID), event_type, timestamp (ISO 8601 UTC), source, payload |
| INF-18 | Include account_id as nullable column in all relevant tables from day one |

## Architecture Decisions Impacting This Story

| Decision | Impact |
|----------|--------|
| AD-4 (Pit Boss) | Centralized risk authority — sole authority on trade approval. Lives in Redis with 60s TTL. Risk Management is sole writer. Execution Engine is read-only consumer. |
| AD-6 (Data Ownership) | risk_events table — sole writer: risk-manager. Append-only, immutable. |
| AD-8 (Redis) | Redis is ephemeral cache. Pit Boss state TTL 60s, refreshed every 30s. If TTL expires, trading halts (fail-safe). All state reconstructable from PostgreSQL. |
| AD-10 (Communication) | Execution → Pit Boss: Sync RPC (Redis GET, <10ms). Risk Management → Redis: Sync write. Risk events → NATS: Async publish. |
| AD-11 (Error Handling) | Emergency stop on: daily budget exhausted, drawdown breaker tripped, API unreachable >5min, data corruption. |
| AD-17 (Observability) | Prometheus metrics for risk checks, daily budget, market/strategy exposures, state refreshes. |
| INF-11 (Decimal) | All monetary values use `decimal.Decimal` — never `float64`. |
| INF-12 (Time) | All timestamps UTC as `TIMESTAMPTZ`. |
| INF-18 (Multi-Account) | `account_id` nullable column in risk_events table from day one. |

## Directory Structure

```
services/risk-manager/
├── cmd/
│   └── main.go                           # Entry point — starts Pit Boss, refresh loop, subscriber
├── internal/
│   ├── pitboss/
│   │   ├── pitboss.go                    # Core risk evaluation engine (ALLOW/DENY logic)
│   │   ├── daily_budget.go               # Daily loss tracking and limit enforcement
│   │   ├── position_limit.go             # Per-market position limit enforcement
│   │   ├── strategy_limit.go             # Per-strategy position limit enforcement
│   │   ├── redis_writer.go               # Redis state writer (30s refresh, 60s TTL)
│   │   ├── reconstructor.go              # State reconstruction from PostgreSQL on restart
│   │   ├── logger.go                     # Risk event logging to PostgreSQL
│   │   └── check.go                      # Synchronous risk check endpoint
│   ├── emergency/
│   │   └── emergency.go                  # Emergency stop logic
│   └── ports/
│       ├── risk_state.go                 # RiskStatePort interface (Redis)
│       └── event.go                      # EventPort interface (NATS)
├── adapters/
│   ├── redis_writer.go                   # Redis adapter (Pit Boss state read/write)
│   ├── postgres_repo.go                  # PostgreSQL adapter (risk_events read/write)
│   ├── nats_subscriber.go                # NATS subscriber (position/order events)
│   └── nats_publisher.go                 # NATS publisher (risk events)
├── config/
│   └── config.go                         # Configuration (limits, intervals, thresholds)
├── metrics/
│   └── metrics.go                        # Prometheus metrics (16 metrics)
└── go.mod
```

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `RISK_DAILY_LOSS_LIMIT_PCT` | `0.02` | Daily loss limit as % of capital (FR-38) |
| `RISK_MARKET_LIMIT_PCT` | `0.10` | Max position per market as % of capital (FR-39) |
| `RISK_STRATEGY_LIMIT_PCT` | `0.20` | Max position per strategy as % of capital (FR-33) |
| `RISK_STATE_REFRESH_INTERVAL` | `30s` | How often to refresh Pit Boss state in Redis (AD-4) |
| `RISK_STATE_TTL` | `60s` | TTL for Pit Boss state keys in Redis (AD-4, AD-8) |
| `RISK_DAILY_BUDGET_WARNING_THRESHOLD` | `0.80` | Utilization threshold for daily budget warning |
| `RISK_RECONSTRUCTION_TIMEOUT` | `30s` | Max time for state reconstruction on startup |
| `CAPITAL_TOTAL` | `10000.00` | Total capital in USD (used for limit calculations) |
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection string |
| `POSTGRES_URL` | `postgres://localhost:5432/pqap` | PostgreSQL connection string |
