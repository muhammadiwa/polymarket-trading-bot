# Story 5.1: Backtesting Engine — Replay & Simulation

Status: ready-for-dev

## Story

As a quant trader,
I want to replay historical market data through my strategy engine with realistic execution simulation,
So that I can evaluate how my strategy would have performed in the past.

## Acceptance Criteria

1. **Given** historical market data exists in TimescaleDB (`opportunities` table from Epic 1)
   **When** the user runs a backtest with a date range and strategy configuration
   **Then** the strategy engine processes the historical data in chronological sequence
   **And** results are deterministic — same input always produces the same output (fixed RNG seed)

2. **Given** a backtest is running
   **When** execution is simulated
   **Then** slippage is configurable (default: 1%)
   **And** partial fills are simulated based on configurable probability
   **And** latency is configurable (default: 100ms)
   **And** the simulation uses the same scorer as live arb-engine

3. **Given** historical data is being replayed
   **When** the engine detects potential lookahead bias (strategy accessing data timestamped after current replay point)
   **Then** warnings are logged with the offending data point
   **And** the biased data point is flagged in results

## Tasks / Subtasks

- [ ] Task 1: Backtesting Service Setup
  - [ ] Create `services/backtesting/` Python/FastAPI service
  - [ ] Connect to TimescaleDB (opportunities table)
  - [ ] Connect to PostgreSQL (results storage)
  - [ ] JWT authentication
- [ ] Task 2: Market Data Replay Engine
  - [ ] Fetch historical opportunities from TimescaleDB
  - [ ] Replay data in chronological sequence (ORDER BY timestamp)
  - [ ] Support configurable date range
  - [ ] Use fixed RNG seed for determinism
- [ ] Task 3: Execution Simulation
  - [ ] Configurable slippage (default 1%) — applied to fill price
  - [ ] Partial fill simulation — random based on `partial_fill_probability`
  - [ ] Configurable latency (default 100ms) — simulated delay
  - [ ] Use existing scorer logic for fill probability estimation
- [ ] Task 4: Lookahead Bias Detection
  - [ ] Track data timestamps during replay
  - [ ] Detect when strategy accesses data with timestamp > current replay point
  - [ ] Log warning and flag in results
- [ ] Task 5: API Endpoints
  - [ ] POST /api/backtesting/run — start backtest (returns run_id)
  - [ ] GET /api/backtesting/{id}/status — check progress (pending/running/completed/failed)
  - [ ] GET /api/backtesting/{id}/results — get results

## Dev Notes

### Architecture Context

- **Service:** New `backtesting` (Python/FastAPI)
- **Database:** TimescaleDB (opportunities table from Epic 1), PostgreSQL (results)
- **Pattern:** Replay historical data through strategy logic (same as live arb-engine)
- **Determinism:** Fixed RNG seed per backtest run (NFR-BT2)

### Key Architecture Rules

- **FR-85:** Replay historical market data through strategy engine
- **FR-86:** Simulate realistic execution (slippage, partial fills, latency)
- **FR-90:** Detect and prevent lookahead bias
- **NFR-BT1:** 1-year data backtest <10 minutes
- **NFR-BT2:** Deterministic — same input → same output
- **NFR-BT3:** Simulation matches live behavior within 10%

### Files to REFERENCE (existing code to replicate logic from)

**`services/arb-engine/internal/detector/simple_arb.go`**
- Simple YES+NO arbitrage detection logic
- Backtesting engine replicates this logic on historical data

**`services/arb-engine/internal/detector/cross_market.go`**
- Cross-market arbitrage detection
- Backtesting engine replicates this logic on historical data

**`services/arb-engine/internal/scorer/scorer.go`**
- Scoring logic (spread × liquidity × fill_probability)
- Backtesting uses same scorer for consistency

**`services/analytics/app/repos/analytics_repo.py`**
- `get_trades_in_range()` — query pattern for TimescaleDB
- `calculate_performance_metrics()` — metric calculations to reuse

### Database Schema

```sql
-- Migration: migrations/postgres/015_create_backtest_runs.up.sql
CREATE TABLE backtest_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id VARCHAR(64) NOT NULL,
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    config JSONB NOT NULL,
    results JSONB,
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_backtest_runs_strategy ON backtest_runs(strategy_id, created_at DESC);
CREATE INDEX idx_backtest_runs_status ON backtest_runs(status);

-- Migration DOWN: migrations/postgres/015_create_backtest_runs.down.sql
DROP TABLE IF EXISTS backtest_runs;
```

### TimescaleDB Source Table

```sql
-- From Epic 1: opportunities table (logged by arb-engine)
-- Columns: id, market_id, spread, score, fill_probability, liquidity,
--           filter_reason, detected_at, strategy_id, side, etc.
SELECT * FROM opportunities
WHERE detected_at BETWEEN $1 AND $2
ORDER BY detected_at ASC
```

### Simulation Config

```python
class SimulationConfig(BaseModel):
    slippage_pct: float = Field(default=0.01, ge=0, le=0.1)  # 1%
    partial_fill_probability: float = Field(default=0.1, ge=0, le=1)
    latency_ms: int = Field(default=100, ge=0, le=10000)
    min_fill_ratio: float = Field(default=0.5, ge=0, le=1)
    rng_seed: int = Field(default=42)  # Determinism
```

### Results Schema (JSONB)

```json
{
  "summary": {
    "total_pnl": "1234.5678",
    "total_trades": 150,
    "win_rate": "0.65",
    "sharpe_ratio": "1.23",
    "max_drawdown": "0.08",
    "profit_factor": "2.1"
  },
  "trades": [
    {
      "timestamp": "2025-01-15T10:30:00Z",
      "market_id": "...",
      "side": "YES",
      "price": "0.65",
      "quantity": "100",
      "slippage": "0.0065",
      "pnl": "12.50",
      "lookahead_warning": false
    }
  ],
  "warnings": [
    {
      "type": "lookahead_bias",
      "timestamp": "2025-01-15T10:30:00Z",
      "message": "Strategy accessed future data at T+5min"
    }
  ]
}
```

### API Response Formats

**POST /api/backtesting/run:**
```json
{
  "run_id": "uuid",
  "status": "pending",
  "message": "Backtest queued"
}
```

**GET /api/backtesting/{id}/status:**
```json
{
  "run_id": "uuid",
  "status": "running",
  "progress": "45%",
  "started_at": "2025-01-15T10:30:00Z"
}
```

**GET /api/backtesting/{id}/results:**
```json
{
  "run_id": "uuid",
  "status": "completed",
  "results": { ... }  // Results schema above
}
```

### Data Flow

```
TimescaleDB (opportunities) → Backtesting Engine → Strategy Logic (replicated) → Simulated Trades → PostgreSQL (backtest_runs)
```

### Performance Notes

- 1 year of data ≈ 10,000 opportunities (estimated)
- Target: <10 minutes for full replay
- Use batch processing: fetch all data upfront, replay in memory
- No network calls during replay (pure computation)

### References

| Reference | Description |
|-----------|-------------|
| FR-85 | Replay historical market data through strategy engine |
| FR-86 | Simulate realistic execution (slippage, partial fills, latency) |
| FR-90 | Detect and prevent lookahead bias |
| NFR-BT1 | 1-year data backtest <10 minutes |
| NFR-BT2 | Deterministic — same input → same output |
| NFR-BT3 | Simulation matches live within 10% |

## Dev Agent Record

### Agent Model Used

mimo-v2.5-pro

### Completion Notes List

- Task 1: Backtesting service setup (config, db, auth, main)
- Task 2: Market data replay from TimescaleDB opportunities table
- Task 3: Execution simulation (slippage, partial fills, latency, RNG seed)
- Task 4: Lookahead bias detection framework
- Task 5: API endpoints (POST /run, GET /status, GET /results)

### File List

**New files:**
- `services/backtesting/app/main.py`
- `services/backtesting/app/config.py`
- `services/backtesting/app/db.py`
- `services/backtesting/app/middleware/auth.py`
- `services/backtesting/app/models/backtest.py`
- `services/backtesting/app/engine/backtest_engine.py`
- `services/backtesting/app/repos/backtest_repo.py`
- `services/backtesting/app/routes/backtest.py`
- `services/backtesting/requirements.txt`
- `services/backtesting/Dockerfile`
- `migrations/postgres/015_create_backtest_runs.up/down.sql`
