# Story 5.1: Backtesting Engine — Replay & Simulation

Status: ready-for-dev

## Story

As a quant trader,
I want to replay historical market data through my strategy engine with realistic execution simulation,
So that I can evaluate how my strategy would have performed in the past.

## Acceptance Criteria

1. **Given** historical market data exists in TimescaleDB
   **When** the user runs a backtest with a date range and strategy configuration
   **Then** the strategy engine processes the historical data in sequence
   **And** results are deterministic — same input always produces the same output

2. **Given** a backtest is running
   **When** execution is simulated
   **Then** slippage is configurable (default: 1%)
   **And** partial fills are simulated
   **And** latency is configurable (default: 100ms)
   **And** the simulation matches live behavior within 10% on key metrics (win rate, average PnL)

3. **Given** historical data is being replayed
   **When** the engine detects potential lookahead bias
   **Then** warnings are logged
   **And** the biased data point is flagged in results

## Tasks / Subtasks

- [ ] Task 1: Backtesting Service Setup
  - [ ] Create `services/backtesting/` Python/FastAPI service
  - [ ] Connect to TimescaleDB (historical market data)
  - [ ] Connect to PostgreSQL (trade results)
  - [ ] JWT authentication
- [ ] Task 2: Market Data Replay Engine
  - [ ] Fetch historical market data from TimescaleDB
  - [ ] Replay data in chronological sequence
  - [ ] Support configurable date range
  - [ ] Deterministic replay (same input → same output)
- [ ] Task 3: Execution Simulation
  - [ ] Configurable slippage (default 1%)
  - [ ] Partial fill simulation
  - [ ] Configurable latency (default 100ms)
  - [ ] Use existing scorer for fill probability
- [ ] Task 4: Lookahead Bias Detection
  - [ ] Detect when strategy uses future data
  - [ ] Log warnings for biased data points
  - [ ] Flag biased results in output
- [ ] Task 5: API Endpoint
  - [ ] POST /api/backtesting/run — start backtest
  - [ ] GET /api/backtesting/{id}/status — check progress
  - [ ] GET /api/backtesting/{id}/results — get results

## Dev Notes

### Architecture Context

- **Service:** New `backtesting` (Python/FastAPI)
- **Database:** TimescaleDB (historical market data), PostgreSQL (results)
- **Pattern:** Replay historical data through existing strategy logic
- **Determinism:** Same input always produces same output (NFR-BT2)

### Key Architecture Rules

- **FR-85:** Replay historical market data through strategy engine
- **FR-86:** Simulate realistic execution (slippage, partial fills, latency)
- **FR-90:** Detect and prevent lookahead bias
- **NFR-BT1:** 1-year data backtest <10 minutes
- **NFR-BT2:** Deterministic — same input → same output
- **NFR-BT3:** Simulation matches live behavior within 10%

### Database Schema

```sql
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
```

### Simulation Config

```python
class SimulationConfig(BaseModel):
    slippage_pct: float = 0.01  # 1%
    partial_fill_probability: float = 0.1
    latency_ms: int = 100
    min_fill_ratio: float = 0.5  # minimum fill ratio for partial fills
```

### Data Flow

```
TimescaleDB (historical data) → Backtesting Engine → Strategy Logic → Simulated Trades → PostgreSQL (results)
```

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

### Debug Log References

### Completion Notes List

### File List
