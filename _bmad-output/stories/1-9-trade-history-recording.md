# Story 1.9: Trade History Recording

## Story

As a quant trader,
I want every trade attempt recorded immutably with full details,
So that I have a complete audit trail for analysis, debugging, and tax reporting.

## Status

ready-for-dev

## Acceptance Criteria

### Immutable Trade Recording

**Given** the execution engine processes an order (success or failure)
**When** the trade result is finalized
**Then** a record is written to the PostgreSQL `trades` table with: timestamp (UTC TIMESTAMPTZ), market, side, price, quantity, fill status, PnL, strategy ID, latency
**And** all required fields are populated — no NULL values for required fields
**And** the `trades` table is append-only (immutable) — no UPDATE or DELETE operations are permitted
**And** all monetary values use Decimal precision (prices: 4dp, quantities: 8dp, PnL: 8dp)
**And** the `account_id` column is included (nullable, default null) for future multi-account support

### Filtering

**Given** trade history records exist in the database
**When** a user queries trade history with filters
**Then** filtering is supported by: date range, market, strategy, side, PnL sign (positive/negative)
**And** filters are combinable (e.g., date range + strategy + positive PnL)
**And** query response time is < 1 second for 10,000 trades (NFR-TH1)

### Export

**Given** trade history records exist and match a query
**When** a user requests an export
**Then** export to CSV is supported with all fields
**And** export to JSON is supported with all fields
**And** export completes within 10 seconds for 10,000 trades

### Data Retention

**Given** trades are recorded
**When** data ages beyond the retention window
**Then** data is retained for a minimum of 3 years (NFR-TH2)
**And** no automated deletion occurs without explicit archival policy

## Technical Requirements

### Architecture Context

- **Write service:** `execution-engine` (Go) — sole writer to `trades` table per AD-6
- **Read service:** `api-gateway` (Python/FastAPI) — serves queries and exports to dashboard
- **Database:** PostgreSQL 17.10 (OLTP) — `trades` table with proper indexing (NFR-TH3)
- **Paradigm:** Event-driven hexagonal architecture (ports & adapters)
- **Event bus:** NATS — `OrderFilled` and `OrderFailed` events trigger trade recording
- **Immutability:** DB-level enforcement via REVOKE on UPDATE/DELETE + application-level append-only pattern
- **Decimal precision:** All monetary values use `decimal.Decimal` (Go) — never `float64` (INF-11)
- **Timestamps:** All UTC as `TIMESTAMPTZ` (INF-12)
- **Multi-account readiness:** `account_id` nullable column included from day one (INF-18)

### Database Schema

```sql
-- migrations/postgres/002_create_trades.up.sql

CREATE TABLE trades (
    -- Primary key
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Trade identification
    client_order_id UUID        NOT NULL UNIQUE,      -- Idempotency key from execution engine (FR-22)
    strategy_id     VARCHAR(64) NOT NULL,              -- Strategy that generated this trade

    -- Market details
    market_id       VARCHAR(128) NOT NULL,             -- Polymarket market ID
    market_slug     VARCHAR(256) NOT NULL,             -- Human-readable market slug
    side            VARCHAR(4)  NOT NULL CHECK (side IN ('YES', 'NO')),

    -- Order details
    order_type      VARCHAR(8)  NOT NULL DEFAULT 'GTC' CHECK (order_type IN ('GTC', 'FOK', 'GTD', 'FAK')),
    price           NUMERIC(12, 4) NOT NULL,           -- 4 decimal places (INF-11)
    quantity         NUMERIC(20, 8) NOT NULL,           -- 8 decimal places (INF-11)
    filled_quantity NUMERIC(20, 8) NOT NULL DEFAULT 0,  -- Actual filled amount

    -- Execution details
    fill_status     VARCHAR(16) NOT NULL CHECK (fill_status IN (
                        'PENDING', 'PLACED', 'FILLED', 'PARTIAL_FILL',
                        'CANCELLED', 'FAILED', 'EXPIRED'
                    )),
    latency_ms      INTEGER     NOT NULL,              -- Order placement latency in ms (FR-24)

    -- Financial
    pnl             NUMERIC(20, 8) NOT NULL DEFAULT 0, -- 8 decimal places (INF-11)
    fee             NUMERIC(12, 4) NOT NULL DEFAULT 0, -- Fee paid (4dp)
    slippage_pct    NUMERIC(8, 4) NOT NULL DEFAULT 0,  -- Slippage as percentage

    -- Timestamps (all UTC TIMESTAMPTZ, INF-12)
    signal_timestamp   TIMESTAMPTZ NOT NULL,            -- When opportunity was detected
    order_timestamp    TIMESTAMPTZ NOT NULL,            -- When order was placed
    fill_timestamp     TIMESTAMPTZ,                     -- When order was filled (NULL if not filled)

    -- Context
    opportunity_id  UUID,                              -- Link to opportunity that triggered this trade
    risk_decision   VARCHAR(16) NOT NULL DEFAULT 'APPROVED', -- Pit Boss decision (FR-47)
    failure_reason  TEXT,                               -- Reason if fill_status = 'FAILED'

    -- Multi-account support (INF-18)
    account_id      UUID,                              -- Nullable; default null for single-account

    -- Immutability guard
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- Record creation time
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()  -- MUST equal created_at (immutability check)
);

-- Immutability constraint: updated_at must equal created_at
-- This prevents silent updates at the application level
-- DB-level protection via REVOKE is applied separately

-- Indexes for query performance (NFR-TH1: <1s for 10k trades)
CREATE INDEX idx_trades_created_at       ON trades (created_at DESC);
CREATE INDEX idx_trades_strategy_id      ON trades (strategy_id, created_at DESC);
CREATE INDEX idx_trades_market_id        ON trades (market_id, created_at DESC);
CREATE INDEX idx_trades_side             ON trades (side, created_at DESC);
CREATE INDEX idx_trades_fill_status      ON trades (fill_status, created_at DESC);
CREATE INDEX idx_trades_pnl              ON trades (pnl, created_at DESC);
CREATE INDEX idx_trades_signal_timestamp ON trades (signal_timestamp DESC);
CREATE INDEX idx_trades_client_order_id  ON trades (client_order_id); -- Unique constraint already creates this

-- Composite indexes for common filter combinations
CREATE INDEX idx_trades_strategy_market  ON trades (strategy_id, market_id, created_at DESC);
CREATE INDEX idx_trades_date_pnl        ON trades (created_at DESC, pnl);
CREATE INDEX idx_trades_account         ON trades (account_id, created_at DESC) WHERE account_id IS NOT NULL;

-- Immutability enforcement: revoke UPDATE and DELETE from application role
-- The application connects as 'pqap_app' role
REVOKE UPDATE, DELETE ON trades FROM pqap_app;

-- Function to enforce immutability at row level
CREATE OR REPLACE FUNCTION enforce_trade_immutability()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.created_at != NEW.created_at THEN
        RAISE EXCEPTION 'Trade records are immutable: created_at cannot be modified';
    END IF;
    IF OLD.updated_at != NEW.updated_at THEN
        RAISE EXCEPTION 'Trade records are immutable: updated_at cannot be modified';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_trade_immutability
    BEFORE UPDATE ON trades
    FOR EACH ROW
    EXECUTE FUNCTION enforce_trade_immutability();

-- Comment for documentation
COMMENT ON TABLE trades IS 'Immutable trade history. Append-only. No UPDATE/DELETE permitted (FR-65).';
COMMENT ON COLUMN trades.client_order_id IS 'UUID for idempotency — prevents duplicate records (FR-22).';
COMMENT ON COLUMN trades.pnl IS 'Realized PnL in USDC. 8 decimal places (INF-11). Zero for unfilled/cancelled orders.';
COMMENT ON COLUMN trades.latency_ms IS 'Time from opportunity signal to order placement in milliseconds (FR-24).';
COMMENT ON COLUMN trades.account_id IS 'Nullable for future multi-account support (INF-8). Default null.';
```

### Key Components to Implement

1. **Trade Record Model** (`shared/models/trade.go`)
   ```go
   type TradeRecord struct {
       ID              string          `json:"id"`                // UUID
       ClientOrderID   string          `json:"client_order_id"`   // Idempotency key
       StrategyID      string          `json:"strategy_id"`
       MarketID        string          `json:"market_id"`
       MarketSlug      string          `json:"market_slug"`
       Side            string          `json:"side"`              // "YES" or "NO"
       OrderType       string          `json:"order_type"`        // "GTC", "FOK", "GTD", "FAK"
       Price           decimal.Decimal `json:"price"`             // 4dp
       Quantity        decimal.Decimal `json:"quantity"`          // 8dp
       FilledQuantity  decimal.Decimal `json:"filled_quantity"`   // 8dp
       FillStatus      string          `json:"fill_status"`       // PENDING|PLACED|FILLED|PARTIAL_FILL|CANCELLED|FAILED|EXPIRED
       LatencyMs       int             `json:"latency_ms"`
       PnL             decimal.Decimal `json:"pnl"`               // 8dp
       Fee             decimal.Decimal `json:"fee"`               // 4dp
       SlippagePct     decimal.Decimal `json:"slippage_pct"`
       SignalTimestamp time.Time       `json:"signal_timestamp"`  // UTC TIMESTAMPTZ
       OrderTimestamp  time.Time       `json:"order_timestamp"`   // UTC TIMESTAMPTZ
       FillTimestamp   *time.Time      `json:"fill_timestamp"`    // NULL if not filled
       OpportunityID   *string         `json:"opportunity_id"`
       RiskDecision    string          `json:"risk_decision"`     // "APPROVED" or "DENIED"
       FailureReason   *string         `json:"failure_reason"`
       AccountID       *string         `json:"account_id"`        // Nullable (INF-18)
       CreatedAt       time.Time       `json:"created_at"`
   }
   ```

2. **Trade Repository — Write Side** (`services/execution-engine/internal/history/repository.go`)
   - Implements `TradeRepository` port interface
   - `Insert(ctx context.Context, trade *TradeRecord) error` — append-only insert
   - Uses `pgx/v5` for PostgreSQL access
   - Idempotent: if `client_order_id` already exists, skip insert (no error)
   - All inserts wrapped in transactions for atomicity
   - Never exposes UPDATE or DELETE methods

3. **Trade Event Handler** (`services/execution-engine/internal/history/handler.go`)
   - Subscribes to internal execution events (OrderFilled, OrderFailed, OrderCancelled, OrderPartialFill)
   - Constructs `TradeRecord` from event data
   - Calculates latency: `order_timestamp - signal_timestamp`
   - Calculates PnL: `filled_quantity * (resolution_price - entry_price)` for filled orders
   - Calculates slippage: `(actual_price - signal_price) / signal_price * 100`
   - Calls `repository.Insert()` for each finalized trade
   - Publishes `TradeRecorded` event to NATS for downstream consumers (analytics, dashboard)

4. **Trade Repository — Read Side** (`services/api-gateway/app/repos/trade_repo.py`)
   - Implements read operations for trade history queries
   - `list_trades(filters, pagination)` — filtered, paginated trade list
   - `export_trades(filters, format)` — CSV/JSON export
   - `get_trade(trade_id)` — single trade lookup
   - `get_stats(filters)` — aggregate stats (total PnL, win rate, trade count)
   - Uses `asyncpg` for async PostgreSQL access

5. **Trade History API Routes** (`services/api-gateway/app/routes/trades.py`)
   - `GET /api/v1/trades` — List trades with filters
     - Query params: `start_date`, `end_date`, `market_id`, `strategy_id`, `side`, `pnl_sign`, `status`, `page`, `page_size`
     - Default sort: `created_at DESC`
     - Pagination: cursor-based (for stable pagination on append-only table)
   - `GET /api/v1/trades/{trade_id}` — Get single trade
   - `GET /api/v1/trades/export` — Export trades
     - Query params: same filters as list + `format` (csv|json)
     - Returns streaming response for large exports
   - `GET /api/v1/trades/stats` — Aggregate statistics
     - Returns: total trades, total PnL, win rate, avg latency, trades by strategy
   - All endpoints require JWT authentication

6. **CSV Exporter** (`services/api-gateway/app/export/csv_exporter.py`)
   - Streams CSV rows to avoid memory issues with large datasets
   - Header row: all trade fields
   - Proper escaping of special characters (commas, quotes in market slugs)
   - RFC 4180 compliant
   - Content-Disposition: attachment; filename="trades_export_YYYYMMDD_HHMMSS.csv"

7. **JSON Exporter** (`services/api-gateway/app/export/json_exporter.py`)
   - Returns JSON array of trade objects
   - All decimal values as strings to preserve precision
   - All timestamps as ISO 8601 UTC
   - Streaming response for large datasets

8. **Trade Recording NATS Event** (`shared/proto/events.go`)
   ```go
   type TradeRecorded struct {
       EventID   string              `json:"event_id"`     // UUID
       EventType string              `json:"event_type"`   // "TradeRecorded"
       Timestamp time.Time           `json:"timestamp"`    // ISO 8601 UTC
       Source    string              `json:"source"`       // "execution-engine"
       Payload   TradeRecordedPayload `json:"payload"`
   }

   type TradeRecordedPayload struct {
       TradeID        string          `json:"trade_id"`        // UUID of the trade record
       ClientOrderID  string          `json:"client_order_id"`
       StrategyID     string          `json:"strategy_id"`
       MarketID       string          `json:"market_id"`
       Side           string          `json:"side"`
       Price          decimal.Decimal `json:"price"`
       FilledQuantity decimal.Decimal `json:"filled_quantity"`
       FillStatus     string          `json:"fill_status"`
       PnL            decimal.Decimal `json:"pnl"`
       LatencyMs      int             `json:"latency_ms"`
   }
   ```

9. **Immutability Guard — Application Level** (`services/execution-engine/internal/history/immutable.go`)
   - Repository only exposes `Insert()` — no `Update()` or `Delete()` methods
   - `Insert()` uses `INSERT ... ON CONFLICT (client_order_id) DO NOTHING` for idempotency
   - No ORM — raw SQL prevents accidental bulk updates
   - Code review checklist: verify no UPDATE/DELETE SQL in history package

### Data Models

**TradeRecord (Go — write side):**
```go
// See TradeRecord struct above in Key Components
```

**TradeResponse (Python — read side API response):**
```python
class TradeResponse(BaseModel):
    id: str                           # UUID
    client_order_id: str              # Idempotency key
    strategy_id: str
    market_id: str
    market_slug: str
    side: Literal["YES", "NO"]
    order_type: Literal["GTC", "FOK", "GTD", "FAK"]
    price: Decimal                    # Serialized as string
    quantity: Decimal                 # Serialized as string
    filled_quantity: Decimal          # Serialized as string
    fill_status: FillStatusEnum
    latency_ms: int
    pnl: Decimal                     # Serialized as string
    fee: Decimal                     # Serialized as string
    slippage_pct: Decimal
    signal_timestamp: datetime        # ISO 8601 UTC
    order_timestamp: datetime         # ISO 8601 UTC
    fill_timestamp: Optional[datetime]
    opportunity_id: Optional[str]
    risk_decision: str
    failure_reason: Optional[str]
    account_id: Optional[str]         # Nullable (INF-18)
    created_at: datetime

class TradeFilterParams(BaseModel):
    start_date: Optional[datetime]
    end_date: Optional[datetime]
    market_id: Optional[str]
    strategy_id: Optional[str]
    side: Optional[Literal["YES", "NO"]]
    pnl_sign: Optional[Literal["positive", "negative"]]
    fill_status: Optional[FillStatusEnum]
    page: int = 1
    page_size: int = 50
```

### Events

**NATS Subjects:**
```
pqap.trade.recorded              # Produced: TradeRecorded (after successful DB insert)
pqap.order.filled                # Consumed: OrderFilled (triggers trade recording)
pqap.order.failed                # Consumed: OrderFailed (triggers trade recording)
pqap.order.cancelled             # Consumed: OrderCancelled (triggers trade recording)
pqap.order.partial_fill          # Consumed: OrderPartialFill (triggers trade recording)
```

### Prometheus Metrics (AD-17)

```
pqap_execution_trade_records_total                 # Counter — total trade records written
pqap_execution_trade_record_latency_ms             # Histogram — DB insert latency
pqap_execution_trade_record_errors_total           # Counter — failed trade record inserts
pqap_api_trade_query_latency_ms                    # Histogram — trade query response time
pqap_api_trade_query_total                         # Counter — total trade queries served
pqap_api_trade_export_total                        # Counter — total exports (by format)
pqap_api_trade_export_duration_ms                  # Histogram — export duration
pqap_api_trade_export_rows_total                   # Counter — total rows exported
```

## Implementation Guide

### Step 1: Database Migration

Create the `trades` table migration:
```sql
-- migrations/postgres/002_create_trades.up.sql
-- (Full schema as defined in Database Schema section above)
```

Create rollback migration:
```sql
-- migrations/postgres/002_create_trades.down.sql
DROP TRIGGER IF EXISTS trg_trade_immutability ON trades;
DROP FUNCTION IF EXISTS enforce_trade_immutability();
DROP TABLE IF EXISTS trades;
```

### Step 2: Trade Record Model (Go)

- Create `shared/models/trade.go` with `TradeRecord` struct
- All monetary fields use `decimal.Decimal` (INF-11)
- All timestamps use `time.Time` with UTC timezone (INF-12)
- JSON tags use snake_case
- Validate fill_status enum values

### Step 3: Trade Repository — Write Side (Go)

- Create `services/execution-engine/internal/history/repository.go`
- Interface: `TradeRepository` with `Insert(ctx, record) error` and `GetByClientOrderID(ctx, id) (*TradeRecord, error)`
- Implementation uses `pgx/v5` directly (no ORM)
- Insert SQL: `INSERT INTO trades (...) VALUES (...) ON CONFLICT (client_order_id) DO NOTHING RETURNING id`
- Idempotent: if client_order_id exists, returns existing ID without error
- Connection pool configured for write-heavy workload

### Step 4: Trade Event Handler (Go)

- Create `services/execution-engine/internal/history/handler.go`
- Subscribes to internal execution events (not NATS — these are internal to execution-engine)
- Flow for each finalized order:
  1. Receive order result (filled, failed, cancelled, partial)
  2. Construct `TradeRecord` from order data
  3. Calculate latency: `order_timestamp - signal_timestamp` in ms
  4. Calculate PnL for filled orders: `filled_quantity * (market_resolution_price - entry_price)` (or 0 for open orders)
  5. Calculate slippage: `(|actual_price - signal_price| / signal_price) * 100`
  6. Call `repository.Insert()`
  7. Publish `TradeRecorded` event to NATS (`pqap.trade.recorded`)
  8. Update Prometheus metrics

### Step 5: Wire Handler into Execution Engine

- Update `services/execution-engine/internal/executor/executor.go`
- After order result is finalized (any terminal state), call trade history handler
- Trade recording is synchronous (must complete before event acknowledgment)
- If trade recording fails, log error but do not retry (trade execution already succeeded)
- Failed recordings are logged to `pqap_execution_trade_record_errors_total` metric

### Step 6: Trade History Read Repository (Python)

- Create `services/api-gateway/app/repos/trade_repo.py`
- Uses `asyncpg` for async PostgreSQL access
- Query builder for dynamic filters:
  ```python
  def build_query(filters: TradeFilterParams) -> tuple[str, list]:
      conditions = []
      params = []
      param_idx = 1

      if filters.start_date:
          conditions.append(f"created_at >= ${param_idx}")
          params.append(filters.start_date)
          param_idx += 1
      if filters.end_date:
          conditions.append(f"created_at <= ${param_idx}")
          params.append(filters.end_date)
          param_idx += 1
      if filters.market_id:
          conditions.append(f"market_id = ${param_idx}")
          params.append(filters.market_id)
          param_idx += 1
      if filters.strategy_id:
          conditions.append(f"strategy_id = ${param_idx}")
          params.append(filters.strategy_id)
          param_idx += 1
      if filters.side:
          conditions.append(f"side = ${param_idx}")
          params.append(filters.side)
          param_idx += 1
      if filters.pnl_sign == "positive":
          conditions.append("pnl > 0")
      elif filters.pnl_sign == "negative":
          conditions.append("pnl < 0")
      if filters.fill_status:
          conditions.append(f"fill_status = ${param_idx}")
          params.append(filters.fill_status)
          param_idx += 1

      where = " AND ".join(conditions) if conditions else "1=1"
      return f"SELECT * FROM trades WHERE {where} ORDER BY created_at DESC", params
  ```
- Pagination: cursor-based using `created_at` and `id` for stable ordering
- Indexes ensure query plans use index scans (NFR-TH1)

### Step 7: Trade History API Routes (Python)

- Create `services/api-gateway/app/routes/trades.py`
- `GET /api/v1/trades`:
  - Parse filter params from query string
  - Call `trade_repo.list_trades(filters, pagination)`
  - Return paginated response with metadata (total_count, page, page_size)
- `GET /api/v1/trades/{trade_id}`:
  - Lookup by ID
  - Return 404 if not found
- `GET /api/v1/trades/export`:
  - Parse filters and format (csv|json)
  - Stream response to avoid memory issues
  - Set appropriate Content-Type and Content-Disposition headers
- `GET /api/v1/trades/stats`:
  - Aggregate query: COUNT, SUM(pnl), AVG(latency_ms), COUNT WHERE pnl > 0 / COUNT as win_rate
  - Group by strategy_id
  - Return stats object

### Step 8: CSV Exporter (Python)

- Create `services/api-gateway/app/export/csv_exporter.py`
- Uses `csv.writer` with `io.StringIO` buffer
- Streams rows in batches (1000 rows per batch)
- Header: id, client_order_id, strategy_id, market_id, market_slug, side, order_type, price, quantity, filled_quantity, fill_status, latency_ms, pnl, fee, slippage_pct, signal_timestamp, order_timestamp, fill_timestamp, opportunity_id, risk_decision, failure_reason, account_id, created_at
- Decimal values as strings to preserve precision
- Timestamps as ISO 8601

### Step 9: JSON Exporter (Python)

- Create `services/api-gateway/app/export/json_exporter.py`
- Streaming JSON array response
- Decimal values as strings (JSON does not guarantee decimal precision)
- Timestamps as ISO 8601 UTC
- Uses `orjson` for fast serialization

### Step 10: TradeRecorded NATS Event

- Update `shared/proto/events.go` with `TradeRecorded` event type
- Publish after successful DB insert
- Downstream consumers: analytics (for real-time metrics), dashboard (for live trade feed)
- Event includes enough data to avoid DB lookups for basic display

### Step 11: Prometheus Metrics

- Update `services/execution-engine/internal/metrics/metrics.go`:
  - `pqap_execution_trade_records_total` (counter by fill_status)
  - `pqap_execution_trade_record_latency_ms` (histogram)
  - `pqap_execution_trade_record_errors_total` (counter)
- Update `services/api-gateway/app/metrics.py`:
  - `pqap_api_trade_query_latency_ms` (histogram)
  - `pqap_api_trade_query_total` (counter)
  - `pqap_api_trade_export_total` (counter by format)
  - `pqap_api_trade_export_duration_ms` (histogram)
  - `pqap_api_trade_export_rows_total` (counter)

## Testing

### Unit Tests

- **Trade repository (`repository_test.go`):**
  - Insert creates record with all fields populated
  - Insert with duplicate client_order_id is idempotent (no error, no duplicate)
  - Insert rejects NULL required fields at DB level
  - Insert uses correct Decimal precision (4dp price, 8dp quantity/pnl)
  - No UPDATE or DELETE methods exposed
  - Latency calculation correct

- **Trade event handler (`handler_test.go`):**
  - OrderFilled event creates trade record with status "FILLED"
  - OrderFailed event creates trade record with status "FAILED" and failure_reason
  - OrderCancelled event creates trade record with status "CANCELLED"
  - OrderPartialFill event creates trade record with status "PARTIAL_FILL"
  - PnL calculated correctly for filled orders
  - Slippage calculated correctly
  - TradeRecorded event published to NATS
  - Prometheus metrics updated

- **Trade query builder (`trade_repo_test.py`):**
  - Filter by date range returns correct subset
  - Filter by market_id returns correct subset
  - Filter by strategy_id returns correct subset
  - Filter by side returns correct subset
  - Filter by pnl_sign positive returns only positive PnL trades
  - Filter by pnl_sign negative returns only negative PnL trades
  - Combined filters work correctly
  - Pagination returns correct page
  - Empty result set handled gracefully

- **CSV exporter (`csv_exporter_test.py`):**
  - All fields exported
  - Special characters escaped correctly
  - Decimal precision preserved (as strings)
  - Timestamps in ISO 8601 format
  - RFC 4180 compliant

- **JSON exporter (`json_exporter_test.py`):**
  - All fields exported
  - Decimal values as strings
  - Timestamps as ISO 8601 UTC
  - Valid JSON output

- **Immutability (`immutability_test.go`):**
  - UPDATE statement on trades raises exception
  - DELETE statement on trades raises exception
  - Trigger enforces created_at cannot change
  - REVOKE prevents direct SQL updates from app role

### Integration Tests

- **End-to-end trade recording:**
  - Simulate order fill → verify trade record in database
  - Verify all fields populated correctly
  - Verify latency calculated correctly
  - Verify TradeRecorded event published

- **Query performance:**
  - Seed 10,000 trades
  - Run filtered query → verify < 1 second response time (NFR-TH1)
  - Run combined filter query → verify < 1 second
  - Run export of 10,000 trades → verify < 10 seconds

- **Immutability enforcement:**
  - Attempt UPDATE via SQL → verify rejection
  - Attempt DELETE via SQL → verify rejection
  - Verify application cannot modify existing records

- **Idempotency:**
  - Send same order result twice → verify only one record created
  - Verify duplicate client_order_id handled gracefully

### Test Files

```
tests/unit/execution-engine/
├── history/
│   ├── repository_test.go        # DB insert, idempotency, precision
│   ├── handler_test.go           # Event handling, PnL calc, NATS publish
│   └── immutability_test.go      # UPDATE/DELETE prevention, trigger

tests/unit/api-gateway/
├── routes/
│   └── trades_test.py            # API endpoint tests, filter validation
├── repos/
│   └── trade_repo_test.py        # Query builder, pagination, filters
└── export/
    ├── csv_exporter_test.py      # CSV format, escaping, precision
    └── json_exporter_test.py     # JSON format, precision, streaming

tests/integration/
├── trade_recording_flow_test.go  # Order → trade record → NATS event
├── trade_query_perf_test.py      # 10k trades < 1s query time
├── trade_export_test.py          # CSV/JSON export correctness
└── trade_immutability_test.go    # DB-level immutability enforcement
```

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| Story 1.4 | — | Execution Engine — Order Placement (prerequisite — produces order results to record) |
| Story 1.3 | — | Arbitrage Detection (prerequisite — provides opportunity_id for trade linking) |
| `github.com/jackc/pgx/v5` | latest | PostgreSQL driver for trade writes |
| `github.com/shopspring/decimal` | latest | Decimal precision (INF-11) |
| `github.com/nats-io/nats.go` | latest | NATS event publishing (TradeRecorded) |
| `github.com/prometheus/client_golang` | latest | Metrics export |
| `go.uber.org/zap` | latest | Structured logging |
| `github.com/google/uuid` | latest | Trade ID generation |
| `asyncpg` | latest | Async PostgreSQL for read side |
| `fastapi` | 0.139.0 | API framework for trade history endpoints |
| `orjson` | latest | Fast JSON serialization for export |
| `pydantic` | latest | Request/response validation |

## External Dependencies (Infrastructure)

| Service | Required | Purpose |
|---------|----------|---------|
| PostgreSQL | Yes | trades table (sole source of truth for trade history) |
| NATS | Yes | TradeRecorded event publishing |
| Redis | No | Not directly used (trade history is PostgreSQL-only) |

## Definition of Done

- [ ] `trades` table created with all columns, constraints, and indexes (NFR-TH3)
- [ ] Immutability enforced at DB level (REVOKE UPDATE/DELETE + trigger) (FR-65)
- [ ] Execution engine writes trade record for every order result (FR-62)
- [ ] All required fields populated — no NULLs for required fields (FR-62)
- [ ] Decimal precision: prices 4dp, quantities 8dp, PnL 8dp (INF-11)
- [ ] All timestamps UTC as TIMESTAMPTZ (INF-12)
- [ ] `account_id` nullable column included (INF-18)
- [ ] Idempotent insert via client_order_id UNIQUE constraint (FR-22)
- [ ] Filtering by date range, market, strategy, side, PnL sign works (FR-63)
- [ ] Combined filters work correctly (FR-63)
- [ ] Query response < 1s for 10,000 trades (NFR-TH1)
- [ ] CSV export with all fields, proper escaping, precision preserved (FR-64)
- [ ] JSON export with all fields, decimal as strings (FR-64)
- [ ] Export < 10s for 10,000 trades (FR-64)
- [ ] TradeRecorded event published to NATS (AD-9)
- [ ] Prometheus metrics exported (AD-17)
- [ ] Structured JSON logging (INF-14)
- [ ] JWT auth on all API endpoints
- [ ] Unit tests pass with ≥80% coverage
- [ ] Integration tests pass
- [ ] Query performance test passes (< 1s for 10k trades)

## References

| Reference | Description |
|-----------|-------------|
| FR-62 | History SHALL record every trade with: timestamp, market, side, price, quantity, fill status, PnL, strategy, latency |
| FR-63 | History SHALL support filtering by: date range, market, strategy, side, PnL sign |
| FR-64 | History SHALL support export to CSV and JSON |
| FR-65 | History SHALL be immutable (no edits, no deletes) |
| FR-22 | Engine SHALL implement idempotent order placement (client order ID prevents duplicates) |
| FR-24 | Engine SHALL log every order attempt with: timestamp, market, side, price, size, result, latency |
| FR-47 | System SHALL log all risk decisions (approve/deny) with full context |
| AD-6 | PostgreSQL single-writer per table: trades written by Execution Engine only |
| AD-9 | NATS is primary event bus; fire-and-forget with at-least-once delivery; consumers idempotent |
| AD-17 | Prometheus metrics on /metrics for all services |
| NFR-TH1 | Trade History query response time: <1s for 10k trades |
| NFR-TH2 | Trade History data retention: minimum 3 years |
| NFR-TH3 | Trade History storage: PostgreSQL with proper indexing |
| INF-11 | Decimal precision: all monetary values use Decimal (never float64); prices 4dp, quantities 8dp, PnL 8dp |
| INF-12 | All timestamps UTC as TIMESTAMPTZ; display timezone configurable |
| INF-14 | Structured JSON logs with timestamp, level, service, request_id, message, context |
| INF-15 | Prometheus metric naming: pqap_{service}_{metric_name}_{unit} |
| INF-16 | Event naming: past tense verb + noun (TradeRecorded) |
| INF-17 | All events include: event_id (UUID), event_type, timestamp (ISO 8601 UTC), source, payload |
| INF-18 | Include account_id as nullable column in all relevant tables from day one |

## Architecture Decisions Impacting This Story

| Decision | Impact |
|----------|--------|
| AD-6 (Data Ownership — PostgreSQL) | trades table written by Execution Engine only. Reads unrestricted. Schema managed by migration tool. |
| AD-9 (NATS) | TradeRecorded event published to NATS for downstream consumers. Idempotent by event UUID. |
| AD-17 (Observability) | Prometheus metrics for trade recording latency, query latency, export metrics. |
| INF-11 (Decimal) | All monetary values use `decimal.Decimal` — never `float64`. Prices 4dp, quantities 8dp, PnL 8dp. |
| INF-12 (Time) | All timestamps UTC as `TIMESTAMPTZ`. Display timezone configurable. |
| INF-18 (Multi-Account) | `account_id` nullable column included from day one for future multi-account support. |

## Directory Structure

```
services/execution-engine/
├── internal/
│   ├── history/
│   │   ├── repository.go           # TradeRepository — insert-only DB access
│   │   ├── repository_test.go      # Unit tests
│   │   ├── handler.go              # Order event → TradeRecord → DB + NATS
│   │   ├── handler_test.go         # Unit tests
│   │   ├── immutability.go         # Application-level immutability guards
│   │   ├── immutability_test.go    # Unit tests
│   │   └── models.go               # TradeRecord struct
│   └── ...
├── adapters/
│   └── postgres_trade_repo.go      # PostgreSQL adapter for TradeRepository
└── ...

services/api-gateway/
├── app/
│   ├── routes/
│   │   ├── trades.py               # Trade history API endpoints
│   │   └── trades_test.py          # Route tests
│   ├── repos/
│   │   ├── trade_repo.py           # Trade read repository (filters, pagination)
│   │   └── trade_repo_test.py      # Repository tests
│   ├── export/
│   │   ├── csv_exporter.py         # CSV export logic
│   │   ├── csv_exporter_test.py    # CSV tests
│   │   ├── json_exporter.py        # JSON export logic
│   │   └── json_exporter_test.py   # JSON tests
│   └── models/
│       └── trade.py                # TradeResponse, TradeFilterParams (Pydantic)
└── ...

shared/
├── proto/
│   └── events.go                   # TradeRecorded event definition
└── models/
    └── trade.go                    # TradeRecord (shared Go model)

migrations/
├── postgres/
│   ├── 002_create_trades.up.sql    # trades table, indexes, immutability trigger
│   └── 002_create_trades.down.sql  # Rollback
└── ...
```

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `TRADE_HISTORY_BATCH_SIZE` | `1000` | Rows per batch for streaming exports |
| `TRADE_HISTORY_MAX_PAGE_SIZE` | `100` | Maximum page size for list endpoint |
| `TRADE_HISTORY_DEFAULT_PAGE_SIZE` | `50` | Default page size for list endpoint |
| `TRADE_HISTORY_EXPORT_TIMEOUT` | `30s` | Max time for export generation |
| `TRADE_HISTORY_RETENTION_YEARS` | `3` | Minimum data retention period |
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |
| `POSTGRES_URL` | `postgres://localhost:5432/pqap` | PostgreSQL connection string |
| `JWT_SECRET` | — | JWT secret for API authentication |
