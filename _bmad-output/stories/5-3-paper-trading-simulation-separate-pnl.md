# Story 5.3: Paper Trading — Simulation & Separate PnL

Status: ready-for-dev

## Story

As a quant trader,
I want to run my strategies in paper trading mode using real market data with simulated execution,
So that I can test strategies without risking real capital.

## Acceptance Criteria

1. **Given** the system is in PAPER execution mode (stored in Redis)
   **When** the arb engine detects an opportunity and the execution engine processes it
   **Then** no real orders are placed on Polymarket
   **And** fills are simulated based on real orderbook depth
   **And** fill simulation is realistic — within 10% of actual fill probability
   **And** simulated positions are tracked in a separate `paper_positions` table

2. **Given** paper trades are executed
   **When** PnL is calculated
   **Then** simulated PnL is tracked independently from live PnL
   **And** paper PnL is clearly labeled and separated in the dashboard
   **And** all simulated trades are logged with the same detail as live trades

3. **Given** paper trading is active
   **When** any operation is performed
   **Then** paper trading never affects live positions or capital — complete isolation

## Tasks / Subtasks

- [ ] Task 1: Execution Mode Infrastructure (AC: #1, #3)
  - [ ] Add `execution_mode` enum (LIVE/PAPER) to Redis
  - [ ] Add mode check to execution engine
  - [ ] Skip real order placement when mode=PAPER
- [ ] Task 2: Fill Simulation (AC: #1)
  - [ ] Simulate fills based on orderbook depth
  - [ ] Use existing fill probability estimator
  - [ ] Configurable simulation parameters
- [ ] Task 3: Paper Positions & PnL (AC: #2, #3)
  - [ ] Create `paper_positions` table
  - [ ] Track simulated positions separately
  - [ ] Calculate paper PnL independently
- [ ] Task 4: Paper Trade Logging (AC: #2)
  - [ ] Log paper trades to `paper_trades` table
  - [ ] Same format as live trades
  - [ ] Queryable via API
- [ ] Task 5: Dashboard Integration (AC: #2)
  - [ ] Show paper mode indicator in dashboard
  - [ ] Separate paper PnL display

## Dev Notes

### Architecture Context

- **Service:** execution-engine (Go) — extends existing
- **Database:** PostgreSQL `paper_positions`, `paper_trades` tables
- **Redis:** `execution_mode` key (LIVE/PAPER)
- **Pattern:** Mode check before order placement; skip CLOB API when PAPER

### Key Architecture Rules

- **AD-12:** Global execution_mode enum in Redis; PAPER mode uses simulated fills from real orderbook
- **FR-91:** Paper trading uses real market data but simulated execution
- **FR-92:** Simulated fills based on real orderbook depth
- **FR-93:** Simulated PnL tracked independently from live PnL
- **FR-95:** All simulated trades logged with same detail as live trades
- **NFR-PT1:** Paper trading never affects live positions or capital
- **NFR-PT2:** Fill simulation realism within 10%

### Database Schema

```sql
CREATE TABLE paper_positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    market_id VARCHAR(128) NOT NULL,
    side VARCHAR(4) NOT NULL,
    entry_price DECIMAL(12,4) NOT NULL,
    quantity DECIMAL(20,8) NOT NULL,
    strategy_id VARCHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    pnl DECIMAL(20,8) DEFAULT 0,
    account_id UUID
);

CREATE TABLE paper_trades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    market_id VARCHAR(128) NOT NULL,
    side VARCHAR(4) NOT NULL,
    price DECIMAL(12,4) NOT NULL,
    quantity DECIMAL(20,8) NOT NULL,
    strategy_id VARCHAR(64) NOT NULL,
    fill_status VARCHAR(16) NOT NULL,
    simulated_latency_ms INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Execution Mode Check

```go
// In execution engine before placing order:
mode, _ := redis.Get(ctx, "pqap:execution_mode")
if mode == "PAPER" {
    // Simulate fill, log to paper_trades, update paper_positions
    return
}
// Normal live execution...
```

### References

| Reference | Description |
|-----------|-------------|
| FR-91 | Paper trading uses real market data but simulated execution |
| FR-92 | Simulated fills based on real orderbook depth |
| FR-93 | Simulated PnL tracked independently |
| FR-95 | All simulated trades logged with same detail |
| AD-12 | Global execution_mode in Redis |
| NFR-PT1 | Complete isolation from live |
| NFR-PT2 | Fill simulation within 10% |

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
