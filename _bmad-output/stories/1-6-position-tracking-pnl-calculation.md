# Story 1.6: Position Tracking & PnL Calculation

## Story

As a quant trader,
I want the bot to track all open positions with real-time PnL and reconcile with Polymarket,
So that I always know my true exposure and profit/loss.

## Status

ready-for-dev

## Acceptance Criteria

**Given** an `OrderFilled` event is received
**When** the position manager processes it
**Then** a new position is created with: market, side, entry price, current price, quantity, unrealized PnL
**And** the position is stored in the PostgreSQL `positions` table

**Given** positions are open and prices are updating
**When** a `MarketPriceUpdated` event is received
**Then** unrealized PnL is recalculated within 1 second using current market prices
**And** if position exceeds configured limits, an alert is sent within 5 seconds

**Given** the position manager is running
**When** every 60 seconds elapse
**Then** position state is reconciled with the Polymarket API
**And** any discrepancy is detected, alerted, and logged
**And** persistent mismatches (>3 consecutive) trigger emergency stop

**Given** a market resolution is detected
**When** the position manager processes it
**Then** the position is automatically settled, PnL is finalized, and the position moves to history

**And** manual position exit (close at market) is supported — exit order placed within 1 second of command

## Technical Requirements

### Architecture Context

- **Service:** `position-manager` (Go)
- **Paradigm:** Event-driven hexagonal architecture (ports & adapters)
- **Event bus:** NATS (JetStream for durable subscriptions)
- **Database:** PostgreSQL `positions` table — sole writer (AD-6)
- **Reconciliation:** Every 60s against Polymarket API; persistent mismatches (>3 consecutive) trigger emergency stop (AD-5)
- **Pattern:** Position lifecycle: Open → Monitor → Close → Settle. Real-time PnL updates within 1s of price change (NFR-P2). Position state accuracy within 1% of Polymarket API (NFR-P1). Mismatch detection within 60s (NFR-P3).

### Key Components to Implement

1. **Position Tracker** (`internal/tracker/tracker.go`)
   - Listens to `OrderFilled` events from NATS (`pqap.order.filled`)
   - Creates new position records with: market_id, side (YES/NO), entry_price, current_price, quantity, unrealized_pnl, status
   - Stores positions in PostgreSQL `positions` table
   - Listens to `MarketPriceUpdated` events (`pqap.market.{market_id}.price`)
   - Recalculates unrealized PnL within 1s of price change (NFR-P2)
   - Publishes `PositionOpened` event on new position (`pqap.position.opened`)
   - Publishes `PositionUpdated` event on PnL recalculation (`pqap.position.updated`)
   - Tracks position status: `OPEN`, `MONITORING`, `CLOSING`, `CLOSED`, `SETTLED`

2. **PnL Calculator** (`internal/tracker/pnl.go`)
   - Unrealized PnL formula:
     - For YES position: `(current_price - entry_price) * quantity`
     - For NO position: `(current_price - entry_price) * quantity`
     - Where prices are in range [0.0000, 1.0000] (4 decimal places)
   - Realized PnL calculated on settlement: `(exit_price - entry_price) * quantity`
   - All monetary values use `decimal.Decimal` — never `float64` (INF-11)
   - Prices: 4 decimal places, Quantities: 8 decimal places, PnL: 8 decimal places (INF-11)
   - Update latency target: within 1s of price change (NFR-P2)

3. **Position Reconciler** (`internal/tracker/reconciler.go`)
   - Runs every 60 seconds (configurable via `POSITION_RECONCILIATION_INTERVAL`)
   - Fetches position state from Polymarket API (`GET /positions`)
   - Compares with internal PostgreSQL state
   - Checks: quantity, side, market_id match
   - On discrepancy:
     - Log discrepancy with full context (internal vs API state)
     - Publish `PositionReconciliationMismatch` event
     - Send warning notification via Telegram
     - Increment `consecutive_mismatches` counter
     - If `consecutive_mismatches > 3`: trigger emergency stop (AD-5)
   - On match: reset `consecutive_mismatches` counter to 0
   - Accuracy target: within 1% of Polymarket API (NFR-P1)

4. **Market Resolution Detector** (`internal/tracker/resolution.go`)
   - Listens to `MarketResolved` events from NATS (`pqap.market.resolved`)
   - Detects when a market with open positions has resolved
   - Automatically settles positions:
     - Calculate final PnL based on resolution outcome (YES wins = $1.00, NO wins = $0.00)
     - Update position status to `SETTLED`
     - Record realized PnL
     - Move position to `position_history` table
   - Publish `PositionClosed` event with settlement details
   - Settlement latency target: within 5s of resolution detection

5. **Manual Position Exit** (`internal/tracker/exit.go`)
   - API endpoint: `POST /api/v1/positions/{position_id}/exit`
   - Places market exit order via Execution Engine
   - Exit order placed within 1s of command (FR-30)
   - On exit:
     - Position status changes to `CLOSING`
     - Exit order published to NATS for Execution Engine
     - On fill: calculate realized PnL, update status to `CLOSED`, move to history
   - Supports "close at market" — uses current market price for exit
   - Logs exit with: timestamp, exit_price, realized_pnl, reason ("manual_exit")

6. **Position Limit Alert** (`internal/tracker/limit_alert.go`)
   - Monitors position exposure against configured limits
   - Limits checked:
     - Per-market position limit (default: 10% of capital, configurable)
     - Per-strategy position limit (default: 20% of capital, configurable)
     - Total capital utilization limit
   - On limit breach:
     - Publish `RiskAlert` event to NATS (`pqap.risk.alert`)
     - Send warning notification via Telegram within 5s (FR-29)
     - Log breach with: limit_type, current_value, threshold, position_id
   - Does NOT auto-close positions — only alerts (Risk Manager handles enforcement)

### Data Models

**Position (internal domain model):**
```go
type Position struct {
    ID              string          `json:"id"`               // UUID — position ID
    MarketID        string          `json:"market_id"`        // Polymarket market ID
    MarketSlug      string          `json:"market_slug"`      // Human-readable market slug
    Side            string          `json:"side"`             // "YES" or "NO"
    EntryPrice      decimal.Decimal `json:"entry_price"`      // 4dp — price at entry
    CurrentPrice    decimal.Decimal `json:"current_price"`    // 4dp — latest market price
    Quantity        decimal.Decimal `json:"quantity"`         // 8dp — position size
    UnrealizedPnL   decimal.Decimal `json:"unrealized_pnl"`   // 8dp — mark-to-market PnL
    RealizedPnL     decimal.Decimal `json:"realized_pnl"`     // 8dp — PnL on close/settle
    Status          PositionStatus  `json:"status"`           // OPEN, MONITORING, CLOSING, CLOSED, SETTLED
    StrategyID      string          `json:"strategy_id"`      // Strategy that opened position
    EntryOrderID    string          `json:"entry_order_id"`   // UUID — order that filled
    ExitOrderID     *string         `json:"exit_order_id"`    // UUID — exit order (nullable)
    OpenedAt        time.Time       `json:"opened_at"`        // UTC TIMESTAMPTZ
    ClosedAt        *time.Time      `json:"closed_at"`        // UTC TIMESTAMPTZ (nullable)
    SettledAt       *time.Time      `json:"settled_at"`       // UTC TIMESTAMPTZ (nullable)
    AccountID       *string         `json:"account_id"`       // nullable, for future multi-account
    CreatedAt       time.Time       `json:"created_at"`       // UTC TIMESTAMPTZ
    UpdatedAt       time.Time       `json:"updated_at"`       // UTC TIMESTAMPTZ
}
```

**PositionHistory (archived positions):**
```go
type PositionHistory struct {
    ID              string          `json:"id"`
    MarketID        string          `json:"market_id"`
    MarketSlug      string          `json:"market_slug"`
    Side            string          `json:"side"`
    EntryPrice      decimal.Decimal `json:"entry_price"`
    ExitPrice       decimal.Decimal `json:"exit_price"`
    Quantity        decimal.Decimal `json:"quantity"`
    RealizedPnL     decimal.Decimal `json:"realized_pnl"`
    StrategyID      string          `json:"strategy_id"`
    EntryOrderID    string          `json:"entry_order_id"`
    ExitOrderID     *string         `json:"exit_order_id"`
    ExitReason      string          `json:"exit_reason"`      // "manual", "resolution", "limit_breach"
    OpenedAt        time.Time       `json:"opened_at"`
    ClosedAt        time.Time       `json:"closed_at"`
    AccountID       *string         `json:"account_id"`
}
```

**PositionReconciliationState:**
```go
type ReconciliationState struct {
    LastReconciledAt     time.Time `json:"last_reconciled_at"`
    ConsecutiveMismatches int      `json:"consecutive_mismatches"`
    TotalMismatches      int64     `json:"total_mismatches"`
    TotalReconciliations int64     `json:"total_reconciliations"`
}
```

### Events

**PositionOpened Event:**
```go
type PositionOpened struct {
    EventID   string              `json:"event_id"`   // UUID
    EventType string              `json:"event_type"` // "PositionOpened"
    Timestamp time.Time           `json:"timestamp"`  // ISO 8601 UTC
    Source    string              `json:"source"`     // "position-manager"
    Payload   PositionOpenedPayload `json:"payload"`
}

type PositionOpenedPayload struct {
    PositionID  string          `json:"position_id"`
    MarketID    string          `json:"market_id"`
    MarketSlug  string          `json:"market_slug"`
    Side        string          `json:"side"`
    EntryPrice  decimal.Decimal `json:"entry_price"`
    Quantity    decimal.Decimal `json:"quantity"`
    StrategyID  string          `json:"strategy_id"`
    EntryOrderID string         `json:"entry_order_id"`
    AccountID   *string         `json:"account_id"`
}
```

**PositionUpdated Event:**
```go
type PositionUpdated struct {
    EventID   string               `json:"event_id"`
    EventType string               `json:"event_type"` // "PositionUpdated"
    Timestamp time.Time            `json:"timestamp"`
    Source    string               `json:"source"`     // "position-manager"
    Payload   PositionUpdatedPayload `json:"payload"`
}

type PositionUpdatedPayload struct {
    PositionID    string          `json:"position_id"`
    MarketID      string          `json:"market_id"`
    CurrentPrice  decimal.Decimal `json:"current_price"`
    UnrealizedPnL decimal.Decimal `json:"unrealized_pnl"`
    UpdatedAt     time.Time       `json:"updated_at"`
}
```

**PositionClosed Event:**
```go
type PositionClosed struct {
    EventID   string              `json:"event_id"`
    EventType string              `json:"event_type"` // "PositionClosed"
    Timestamp time.Time           `json:"timestamp"`
    Source    string              `json:"source"`     // "position-manager"
    Payload   PositionClosedPayload `json:"payload"`
}

type PositionClosedPayload struct {
    PositionID   string          `json:"position_id"`
    MarketID     string          `json:"market_id"`
    Side         string          `json:"side"`
    EntryPrice   decimal.Decimal `json:"entry_price"`
    ExitPrice    decimal.Decimal `json:"exit_price"`
    Quantity     decimal.Decimal `json:"quantity"`
    RealizedPnL  decimal.Decimal `json:"realized_pnl"`
    ExitReason   string          `json:"exit_reason"`
    StrategyID   string          `json:"strategy_id"`
    AccountID    *string         `json:"account_id"`
}
```

**PositionReconciliationMismatch Event:**
```go
type PositionReconciliationMismatch struct {
    EventID   string                           `json:"event_id"`
    EventType string                           `json:"event_type"` // "PositionReconciliationMismatch"
    Timestamp time.Time                        `json:"timestamp"`
    Source    string                           `json:"source"`     // "position-manager"
    Payload   PositionReconciliationMismatchPayload `json:"payload"`
}

type PositionReconciliationMismatchPayload struct {
    PositionID            string `json:"position_id"`
    InternalQuantity      string `json:"internal_quantity"`
    APIQuantity            string `json:"api_quantity"`
    InternalSide          string `json:"internal_side"`
    APISide               string `json:"api_side"`
    ConsecutiveMismatches int    `json:"consecutive_mismatches"`
    MismatchType          string `json:"mismatch_type"` // "quantity", "side", "missing", "extra"
}
```

### API Endpoints

**Polymarket API (consumed):**
- `GET /positions` — Fetch all positions for reconciliation
- `GET /markets/{market_id}` — Fetch market resolution status

**Internal API (produced):**
- `GET /api/v1/positions` — List all open positions
- `GET /api/v1/positions/{position_id}` — Get position details
- `POST /api/v1/positions/{position_id}/exit` — Manual position exit (requires JWT auth)
- `GET /api/v1/positions/history` — List closed/settled positions
- `GET /api/v1/positions/reconciliation/status` — Get reconciliation state

**NATS Subjects:**
```
pqap.order.filled                # Consumed: OrderFilled (from execution-engine)
pqap.market.{market_id}.price    # Consumed: MarketPriceUpdated (from scanner)
pqap.market.resolved             # Consumed: MarketResolved (from scanner)
pqap.position.opened             # Produced: PositionOpened
pqap.position.updated            # Produced: PositionUpdated
pqap.position.closed             # Produced: PositionClosed
pqap.position.reconciliation_mismatch # Produced: PositionReconciliationMismatch
pqap.risk.alert                  # Produced: RiskAlert (limit breaches)
pqap.notification.send           # Produced: NotificationRequest (Telegram alerts)
pqap.risk.emergency              # Produced: EmergencyStop (after 3 consecutive mismatches)
```

### Prometheus Metrics (AD-17)

```
pqap_position_open_total                    # Counter — total positions opened
pqap_position_closed_total                  # Counter — total positions closed
pqap_position_settled_total                 # Counter — total positions settled (resolution)
pqap_position_active_count                  # Gauge — current open positions
pqap_position_pnl_update_latency_ms         # Histogram — PnL recalculation latency (target: <1s)
pqap_position_unrealized_pnl_usd            # Gauge — total unrealized PnL across all positions
pqap_position_realized_pnl_usd_total        # Counter — cumulative realized PnL
pqap_position_reconciliation_total          # Counter — total reconciliation runs
pqap_position_reconciliation_mismatches_total # Counter — total mismatches detected
pqap_position_reconciliation_consecutive    # Gauge — current consecutive mismatch count
pqap_position_exit_latency_ms               # Histogram — manual exit order latency (target: <1s)
pqap_position_limit_breach_total            # Counter — total limit breach alerts
```

## Implementation Guide

### Step 1: Database Schema

Create `positions` table:
```sql
CREATE TABLE positions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    market_id       TEXT NOT NULL,
    market_slug     TEXT NOT NULL,
    side            TEXT NOT NULL,            -- "YES" or "NO"
    entry_price     NUMERIC(10,4) NOT NULL,
    current_price   NUMERIC(10,4) NOT NULL,
    quantity        NUMERIC(18,8) NOT NULL,
    unrealized_pnl  NUMERIC(18,8) NOT NULL DEFAULT 0,
    realized_pnl    NUMERIC(18,8) NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'OPEN',  -- OPEN, MONITORING, CLOSING, CLOSED, SETTLED
    strategy_id     TEXT NOT NULL,
    entry_order_id  UUID NOT NULL,
    exit_order_id   UUID DEFAULT NULL,
    opened_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at       TIMESTAMPTZ DEFAULT NULL,
    settled_at      TIMESTAMPTZ DEFAULT NULL,
    account_id      UUID DEFAULT NULL,        -- nullable, for future multi-account
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_positions_market_id ON positions(market_id);
CREATE INDEX idx_positions_status ON positions(status);
CREATE INDEX idx_positions_strategy_id ON positions(strategy_id);
CREATE INDEX idx_positions_account_id ON positions(account_id);
```

Create `position_history` table:
```sql
CREATE TABLE position_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    market_id       TEXT NOT NULL,
    market_slug     TEXT NOT NULL,
    side            TEXT NOT NULL,
    entry_price     NUMERIC(10,4) NOT NULL,
    exit_price      NUMERIC(10,4) NOT NULL,
    quantity        NUMERIC(18,8) NOT NULL,
    realized_pnl    NUMERIC(18,8) NOT NULL,
    strategy_id     TEXT NOT NULL,
    entry_order_id  UUID NOT NULL,
    exit_order_id   UUID DEFAULT NULL,
    exit_reason     TEXT NOT NULL,            -- "manual", "resolution", "limit_breach"
    opened_at       TIMESTAMPTZ NOT NULL,
    closed_at       TIMESTAMPTZ NOT NULL,
    account_id      UUID DEFAULT NULL,
    archived_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_position_history_market_id ON position_history(market_id);
CREATE INDEX idx_position_history_strategy_id ON position_history(strategy_id);
CREATE INDEX idx_position_history_closed_at ON position_history(closed_at);
```

Create `position_reconciliation_log` table:
```sql
CREATE TABLE position_reconciliation_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    position_id     UUID DEFAULT NULL,
    market_id       TEXT NOT NULL,
    mismatch_type   TEXT NOT NULL,            -- "quantity", "side", "missing", "extra"
    internal_state  JSONB NOT NULL,
    api_state       JSONB NOT NULL,
    consecutive_mismatches INTEGER NOT NULL,
    resolved        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_recon_log_created_at ON position_reconciliation_log(created_at);
```

### Step 2: Position Tracker — OrderFilled Handler

- Subscribe to `pqap.order.filled` NATS subject
- On `OrderFilled` event:
  - Extract: market_id, market_slug, side, fill_price, filled_qty, order_id, strategy_id
  - Calculate initial unrealized PnL (0 at entry)
  - Insert into `positions` table with status `OPEN`
  - Publish `PositionOpened` event to `pqap.position.opened`
  - Log position creation with full context
  - Increment `pqap_position_open_total` counter
  - Set `pqap_position_active_count` gauge

### Step 3: PnL Calculator — Real-time Updates

- Subscribe to `pqap.market.{market_id}.price` NATS subjects (wildcard: `pqap.market.*.price`)
- On `MarketPriceUpdated` event:
  - Query all open positions for this `market_id`
  - For each position:
    - Calculate unrealized PnL:
      ```
      if side == "YES":
          unrealized_pnl = (current_price - entry_price) * quantity
      else:  # NO
          unrealized_pnl = (current_price - entry_price) * quantity
      ```
    - Update `current_price` and `unrealized_pnl` in PostgreSQL
    - Publish `PositionUpdated` event
    - Record PnL update latency in `pqap_position_pnl_update_latency_ms` histogram
  - Update `pqap_position_unrealized_pnl_usd` gauge (sum of all unrealized PnL)
- Latency target: within 1s of price change (NFR-P2)

### Step 4: Position Reconciler — 60s Loop

- Start reconciliation goroutine on service startup
- Every 60 seconds (configurable):
  - Fetch all positions from Polymarket API (`GET /positions`)
  - Fetch all open positions from PostgreSQL
  - Compare:
    - For each internal position: find matching API position
    - Check: quantity, side, market_id
    - Log any discrepancy to `position_reconciliation_log`
  - On mismatch:
    - Publish `PositionReconciliationMismatch` event
    - Increment `consecutive_mismatches` counter
    - Send warning Telegram notification
    - If `consecutive_mismatches > 3`:
      - Trigger emergency stop (`pqap.risk.emergency`)
      - Send critical Telegram alert
      - Log emergency trigger with full context
  - On match:
    - Reset `consecutive_mismatches` to 0
  - Update `pqap_position_reconciliation_total` counter
  - Update `pqap_position_reconciliation_mismatches_total` counter
  - Update `pqap_position_reconciliation_consecutive` gauge

### Step 5: Market Resolution Detector

- Subscribe to `pqap.market.resolved` NATS subject
- On `MarketResolved` event:
  - Query all open positions for this `market_id`
  - For each position:
    - Determine resolution outcome:
      - If market resolved YES: exit_price = 1.0000 for YES positions, 0.0000 for NO positions
      - If market resolved NO: exit_price = 0.0000 for YES positions, 1.0000 for NO positions
    - Calculate realized PnL: `(exit_price - entry_price) * quantity`
    - Update position status to `SETTLED`
    - Set `settled_at` timestamp
    - Move to `position_history` table with `exit_reason = "resolution"`
    - Publish `PositionClosed` event
    - Log settlement with full context
    - Increment `pqap_position_settled_total` counter
    - Update `pqap_position_realized_pnl_usd_total` counter

### Step 6: Manual Position Exit

- Expose API endpoint: `POST /api/v1/positions/{position_id}/exit`
- Authentication: JWT token required
- On request:
  - Validate position exists and is in `OPEN` or `MONITORING` status
  - Update position status to `CLOSING`
  - Create exit order request:
    ```go
    exitOrder := ExitOrderRequest{
        PositionID:   positionID,
        MarketID:     position.MarketID,
        Side:         position.Side,
        Quantity:     position.Quantity,
        OrderType:    "MARKET",  // Close at market
        Reason:       "manual_exit",
    }
    ```
  - Publish exit order to NATS for Execution Engine to process
  - Record `exit_order_id` on position
  - On exit fill (received via `OrderFilled`):
    - Calculate realized PnL: `(exit_price - entry_price) * quantity`
    - Update position status to `CLOSED`
    - Set `closed_at` timestamp
    - Move to `position_history` with `exit_reason = "manual"`
    - Publish `PositionClosed` event
  - Record exit latency in `pqap_position_exit_latency_ms` histogram

### Step 7: Position Limit Alerts

- On every PnL recalculation (Step 3):
  - Check position against configured limits:
    - Per-market limit: `position.value > total_capital * market_limit_pct`
    - Per-strategy limit: `strategy_exposure > total_capital * strategy_limit_pct`
  - If limit breached:
    - Publish `RiskAlert` event to `pqap.risk.alert`
    - Publish `NotificationRequest` to Telegram (warning level)
    - Log breach to `risk_events` table
    - Increment `pqap_position_limit_breach_total` counter
  - Alert delivery target: within 5s (FR-29)

### Step 8: Event Publishing

- All events follow standard schema (INF-17):
  - `event_id`: UUID v4
  - `event_type`: past tense verb + noun (INF-16)
  - `timestamp`: ISO 8601 UTC
  - `source`: "position-manager"
  - `payload`: event-specific JSON
- Events produced:
  - `PositionOpened` → `pqap.position.opened`
  - `PositionUpdated` → `pqap.position.updated`
  - `PositionClosed` → `pqap.position.closed`
  - `PositionReconciliationMismatch` → `pqap.position.reconciliation_mismatch`
  - `RiskAlert` → `pqap.risk.alert`
  - `NotificationRequest` → `pqap.notification.send`
  - `EmergencyStop` → `pqap.risk.emergency` (on 3+ consecutive mismatches)
- Fire-and-forget with at-least-once delivery (AD-9)
- Consumers idempotent by event_id

## Testing

### Unit Tests

- **Position tracker (`tracker_test.go`):**
  - OrderFilled creates position with correct fields
  - Position stored in PostgreSQL with correct status
  - PositionOpened event published
  - Multiple fills for same market create separate positions

- **PnL calculator (`pnl_test.go`):**
  - YES position: unrealized PnL = (current - entry) * quantity
  - NO position: unrealized PnL = (current - entry) * quantity
  - PnL update latency < 1s
  - Decimal precision maintained (8dp)
  - Edge cases: price at 0.0000, price at 1.0000

- **Reconciler (`reconciler_test.go`):**
  - Match: consecutive_mismatches reset to 0
  - Mismatch: consecutive_mismatches incremented
  - 3+ consecutive mismatches: emergency stop triggered
  - Reconciliation log written to PostgreSQL
  - Telegram alert sent on mismatch

- **Resolution detector (`resolution_test.go`):**
  - YES resolution: YES position settled at 1.0000, NO at 0.0000
  - NO resolution: YES position settled at 0.0000, NO at 1.0000
  - Realized PnL calculated correctly
  - Position moved to history
  - PositionClosed event published

- **Manual exit (`exit_test.go`):**
  - Exit order placed within 1s
  - Position status changes to CLOSING
  - On fill: realized PnL calculated, status CLOSED
  - Position moved to history with exit_reason = "manual"
  - Invalid position ID returns 404
  - Closed position cannot be exited (returns 400)

- **Limit alerts (`limit_alert_test.go`):**
  - Per-market limit breach: alert sent
  - Per-strategy limit breach: alert sent
  - Alert delivery within 5s
  - No alert when within limits

### Integration Tests

- **OrderFilled → Position created:**
  - Publish OrderFilled event to NATS
  - Verify position created in PostgreSQL
  - Verify PositionOpened event published
  - Verify all fields correct

- **Price update → PnL recalculation:**
  - Create position
  - Publish MarketPriceUpdated event
  - Verify PnL updated in PostgreSQL within 1s
  - Verify PositionUpdated event published

- **Reconciliation → mismatch detection:**
  - Create position with known state
  - Mock Polymarket API to return different state
  - Run reconciliation
  - Verify mismatch detected and logged
  - Verify Telegram alert sent

- **Reconciliation → emergency stop:**
  - Create 4 consecutive mismatches
  - Verify emergency stop triggered after 3rd mismatch
  - Verify EmergencyStop event published
  - Verify critical Telegram alert sent

- **Market resolution → settlement:**
  - Create YES and NO positions for same market
  - Publish MarketResolved event (YES wins)
  - Verify YES position settled at 1.0000
  - Verify NO position settled at 0.0000
  - Verify both moved to history
  - Verify PositionClosed events published

- **Manual exit → close at market:**
  - Create position
  - Call POST /api/v1/positions/{id}/exit
  - Verify exit order placed within 1s
  - Simulate fill
  - Verify position closed and moved to history

### Test Files

```
tests/unit/position-manager/
├── tracker_test.go                 # Position lifecycle (open, close, settle)
├── pnl_test.go                     # PnL calculation (unrealized, realized, precision)
├── reconciler_test.go              # API reconciliation (match, mismatch, emergency)
├── resolution_test.go              # Market resolution detection and settlement
├── exit_test.go                    # Manual position exit (API, order, fill)
└── limit_alert_test.go             # Position limit breach alerts

tests/integration/
├── position_order_filled_test.go   # OrderFilled → position creation flow
├── position_price_update_test.go   # Price update → PnL recalculation flow
├── position_reconciliation_test.go # Reconciliation loop → mismatch detection
├── position_resolution_test.go     # Market resolution → settlement flow
├── position_manual_exit_test.go    # Manual exit → order placement → fill
└── position_limit_alert_test.go    # Limit breach → alert delivery
```

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| Story 1.1 | — | Scanner WebSocket Connection & Market Catalog (prerequisite — produces MarketPriceUpdated) |
| Story 1.2 | — | Market Scanner — Stale Detection, Reconnect & Batching (prerequisite) |
| Story 1.4 | — | Execution Engine — Order Placement & Slippage Protection (prerequisite — produces OrderFilled) |
| Story 1.5 | — | Execution Engine — Atomic YES+NO & Circuit Breaker (prerequisite — handles exit orders) |
| Story 1.7 | — | Risk Management — Pit Boss & Daily Budget (prerequisite — provides capital limits) |
| Story 1.10 | — | Telegram Notifications (prerequisite — delivers reconciliation alerts) |
| `github.com/nats-io/nats.go` | latest | NATS event bus |
| `github.com/shopspring/decimal` | latest | Decimal precision (prices, quantities, PnL) |
| `github.com/prometheus/client_golang` | latest | Metrics export |
| `go.uber.org/zap` | latest | Structured logging |
| `github.com/google/uuid` | latest | Position ID, event ID generation |
| `github.com/jackc/pgx/v5` | latest | PostgreSQL driver |

## External Dependencies (Infrastructure)

| Service | Required | Purpose |
|---------|----------|---------|
| NATS server | Yes | Event bus (consume OrderFilled, MarketPriceUpdated, MarketResolved; produce position events) |
| PostgreSQL | Yes | positions, position_history, position_reconciliation_log tables |
| Polymarket API | Yes | Position reconciliation (GET /positions), market resolution status |
| Telegram Bot API | Yes | Reconciliation alerts, limit breach warnings (via notification service) |

## Definition of Done

- [ ] OrderFilled creates position with correct fields in PostgreSQL (FR-25)
- [ ] PnL recalculated within 1s of MarketPriceUpdated event (FR-28, NFR-P2)
- [ ] Position state accuracy within 1% of Polymarket API (NFR-P1)
- [ ] Reconciliation runs every 60s (FR-26, NFR-P3)
- [ ] Mismatch detection, alerting, and logging (FR-26)
- [ ] 3+ consecutive mismatches trigger emergency stop (AD-5)
- [ ] Market resolution auto-settles positions (FR-27)
- [ ] Manual exit places order within 1s (FR-30)
- [ ] Position limit alerts sent within 5s of breach (FR-29)
- [ ] Prometheus metrics exported (AD-17)
- [ ] Structured JSON logging (INF-14)
- [ ] Decimal precision for all monetary values (INF-11)
- [ ] All timestamps UTC as TIMESTAMPTZ (INF-12)
- [ ] `account_id` nullable column in all tables (INF-18)
- [ ] Unit tests pass with ≥80% coverage
- [ ] Integration tests pass

## References

| Reference | Description |
|-----------|-------------|
| FR-25 | Manager SHALL track all open positions with: market, side, entry price, current price, quantity, unrealized PnL |
| FR-26 | Manager SHALL reconcile position state with Polymarket API every 60 seconds |
| FR-27 | Manager SHALL detect market resolution and automatically settle positions |
| FR-28 | Manager SHALL calculate unrealized PnL using current market prices |
| FR-29 | Manager SHALL alert when position exceeds configured limits |
| FR-30 | Manager SHALL support manual position exit (close at market) |
| AD-5 | State Reconciliation: position reconciliation every 60s; persistent mismatches (>3 consecutive) trigger emergency stop |
| AD-6 | Data Ownership: positions table written by Position Manager only |
| AD-9 | Event Bus: NATS with defined subject hierarchy; fire-and-forget, at-least-once, idempotent |
| AD-17 | Observability: Prometheus metrics, Grafana dashboards, structured JSON logs |
| NFR-P1 | Position state accuracy vs Polymarket API: within 1% |
| NFR-P2 | PnL update latency: within 1s of price change |
| NFR-P3 | State mismatch detection: within 60s |
| INF-11 | Decimal precision: all monetary values use Decimal (never float64); prices 4dp, quantities 8dp, PnL 8dp |
| INF-12 | All timestamps UTC as TIMESTAMPTZ; display timezone configurable |
| INF-14 | Structured JSON logs with timestamp, level, service, request_id, message, context |
| INF-15 | Prometheus metric naming: pqap_{service}_{metric_name}_{unit} |
| INF-16 | Event naming: past tense verb + noun (PositionOpened, PositionClosed) |
| INF-17 | All events include: event_id (UUID), event_type, timestamp (ISO 8601 UTC), source, payload |
| INF-18 | Include account_id as nullable column in all relevant tables from day one |

## Architecture Decisions Impacting This Story

| Decision | Impact |
|----------|--------|
| AD-5 (State Reconciliation) | Position reconciliation every 60s. Persistent mismatches (>3 consecutive) trigger emergency stop. All reconciliation events logged. |
| AD-6 (Data Ownership) | positions table — sole writer: position-manager. position_history for archived positions. position_reconciliation_log for audit trail. |
| AD-9 (Event Bus) | Consumes: `pqap.order.filled`, `pqap.market.*.price`, `pqap.market.resolved`. Produces: position lifecycle events + risk alerts. Fire-and-forget, at-least-once, idempotent. |
| AD-17 (Observability) | Prometheus metrics for position counts, PnL, reconciliation state, exit latency, limit breaches. |
| INF-11 (Decimal) | All prices, quantities, PnL use `decimal.Decimal` — never `float64`. |
| INF-12 (Time) | All timestamps UTC as `TIMESTAMPTZ`. |
| INF-18 (Multi-Account) | `account_id` nullable column in all tables from day one. |

## Directory Structure

```
services/position-manager/
├── cmd/
│   └── main.go                           # Entry point — starts tracker, reconciler, subscriber
├── internal/
│   ├── tracker/
│   │   ├── tracker.go                    # Position lifecycle (open, monitor, close, settle)
│   │   ├── pnl.go                        # PnL calculation (unrealized, realized)
│   │   ├── reconciler.go                 # API state reconciliation (60s loop)
│   │   ├── resolution.go                 # Market resolution detection and settlement
│   │   ├── exit.go                       # Manual position exit handler
│   │   └── limit_alert.go               # Position limit breach alerts
│   └── ports/
│       ├── position.go                   # PositionPort interface (Polymarket API)
│       └── event.go                      # EventPort interface (NATS)
├── adapters/
│   ├── polymarket_account.go             # Polymarket API adapter (positions, resolution)
│   ├── postgres_repo.go                  # PostgreSQL adapter (positions, history, recon log)
│   ├── nats_subscriber.go                # NATS subscriber (OrderFilled, MarketPriceUpdated, MarketResolved)
│   └── nats_publisher.go                 # NATS publisher (position/risk/notification events)
├── config/
│   └── config.go                         # Configuration (reconciliation interval, limits)
├── metrics/
│   └── metrics.go                        # Prometheus metrics (12 metrics)
└── go.mod
```

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `POSITION_RECONCILIATION_INTERVAL` | `60s` | How often to reconcile with Polymarket API (FR-26) |
| `POSITION_RECONCILIATION_MISMATCH_THRESHOLD` | `3` | Consecutive mismatches before emergency stop (AD-5) |
| `POSITION_MARKET_LIMIT_PCT` | `0.10` | Max position per market as % of capital (FR-34) |
| `POSITION_STRATEGY_LIMIT_PCT` | `0.20` | Max position per strategy as % of capital (FR-33) |
| `POSITION_PNL_UPDATE_TIMEOUT_MS` | `1000` | Max time to recalculate PnL (NFR-P2) |
| `POSITION_EXIT_ORDER_TIMEOUT_MS` | `1000` | Max time to place exit order (FR-30) |
| `POSITION_ALERT_DELIVERY_TIMEOUT_MS` | `5000` | Max time to deliver limit breach alert (FR-29) |
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |
| `POSTGRES_URL` | `postgres://localhost:5432/pqap` | PostgreSQL connection string |
| `POLYMARKET_API_URL` | `https://gamma-api.polymarket.com` | Polymarket API base URL |
