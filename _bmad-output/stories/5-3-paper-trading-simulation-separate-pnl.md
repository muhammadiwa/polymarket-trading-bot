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
  - [ ] Add `execution_mode` key to Redis (LIVE/PAPER)
  - [ ] Add GET/PUT /api/execution-mode endpoint to api-gateway
  - [ ] Add mode check to execution engine before order placement
  - [ ] Skip real CLOB API calls when mode=PAPER
- [ ] Task 2: Fill Simulation (AC: #1)
  - [ ] Simulate fills using existing `FillProbabilityEstimator`
  - [ ] Use real orderbook depth from Redis market price cache
  - [ ] Configurable simulation: slippage, partial fill probability
  - [ ] Simulation result logged with `simulated_latency_ms`
- [ ] Task 3: Paper Positions & PnL (AC: #2, #3)
  - [ ] Create `paper_positions` table migration
  - [ ] Track simulated positions in execution engine
  - [ ] Calculate paper PnL = (exit_price - entry_price) * quantity
  - [ ] Paper PnL never mixes with live PnL
- [ ] Task 4: Paper Trade Logging (AC: #2)
  - [ ] Create `paper_trades` table migration
  - [ ] Log paper trades with same format as live trades
  - [ ] Include `pnl` per trade
  - [ ] Queryable via GET /api/paper-trades
- [ ] Task 5: Dashboard Integration (AC: #2)
  - [ ] Show paper mode indicator in dashboard header
  - [ ] Add paper PnL display to portfolio overview
  - [ ] Add paper trades to trade history (labeled)

## Dev Notes

### Architecture Context

- **Service:** execution-engine (Go) — extends existing, api-gateway — extends existing
- **Database:** PostgreSQL `paper_positions`, `paper_trades` tables
- **Redis:** `pqap:execution_mode` key (LIVE/PAPER)
- **Pattern:** Mode check before order placement; skip CLOB API when PAPER

### Key Architecture Rules

- **AD-12:** Global execution_mode enum in Redis; PAPER mode uses simulated fills from real orderbook
- **FR-91:** Paper trading uses real market data but simulated execution
- **FR-92:** Simulated fills based on real orderbook depth
- **FR-93:** Simulated PnL tracked independently from live PnL
- **FR-95:** All simulated trades logged with same detail as live trades
- **NFR-PT1:** Paper trading never affects live positions or capital
- **NFR-PT2:** Fill simulation realism within 10%

### Files to MODIFY

**`services/execution-engine/internal/executor/executor.go`**
- Current: `Execute()` places real orders via `orderPort`
- Change: Add mode check before `orderPort.PlaceOrder()` — if PAPER, simulate fill instead
- Preserve: All existing live execution logic

**`services/execution-engine/internal/executor/risk_check.go`**
- Current: Checks Pit Boss before order
- Change: Same check applies to paper mode (risk limits still enforced)
- Preserve: All existing risk check logic

**`services/api-gateway/app/routes/`**
- New: `execution_mode.py` — GET/PUT /api/execution-mode endpoint
- Mode change requires admin role

### Files to CREATE

**`migrations/postgres/016_create_paper_positions.up/down.sql`**
- Paper positions table

**`migrations/postgres/017_create_paper_trades.up/down.sql`**
- Paper trades table

**`services/api-gateway/app/routes/execution_mode.py`**
- Mode management endpoint

### Database Schema

```sql
-- Migration 016
CREATE TABLE paper_positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    market_id VARCHAR(128) NOT NULL,
    side VARCHAR(4) NOT NULL CHECK (side IN ('YES', 'NO')),
    entry_price NUMERIC(12,4) NOT NULL,
    quantity NUMERIC(20,8) NOT NULL,
    strategy_id VARCHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    pnl NUMERIC(20,8) DEFAULT 0,
    account_id UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_paper_positions_strategy ON paper_positions(strategy_id, status);
CREATE INDEX idx_paper_positions_market ON paper_positions(market_id, status);

-- Migration 017
CREATE TABLE paper_trades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    market_id VARCHAR(128) NOT NULL,
    side VARCHAR(4) NOT NULL,
    price NUMERIC(12,4) NOT NULL,
    quantity NUMERIC(20,8) NOT NULL,
    pnl NUMERIC(20,8) NOT NULL DEFAULT 0,
    strategy_id VARCHAR(64) NOT NULL,
    fill_status VARCHAR(16) NOT NULL,
    simulated_latency_ms INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_paper_trades_strategy ON paper_trades(strategy_id, created_at DESC);
CREATE INDEX idx_paper_trades_market ON paper_trades(market_id, created_at DESC);
```

### Execution Mode Check

```go
// In executor.go Execute() method, before placing order:
mode, err := r.riskPort.GetExecutionMode(ctx)
if err != nil {
    r.logger.Error("failed to get execution mode", zap.Error(err))
    // Default to LIVE for safety
    mode = "LIVE"
}

if mode == "PAPER" {
    // Simulate fill
    fill := r.simulateFill(ctx, order)
    // Log to paper_trades
    r.logPaperTrade(ctx, order, fill)
    // Update paper_positions
    r.updatePaperPosition(ctx, order, fill)
    return fill, nil
}

// Normal live execution...
```

### Fill Simulation Logic

```go
func (r *Executor) simulateFill(ctx context.Context, order *ports.Order) *ports.SimulatedFill {
    // Get real orderbook depth from Redis
    depth := r.marketPricePort.GetLiquidityDepth(ctx, order.MarketID)
    
    // Use existing fill probability estimator
    fillProb := r.fillProbEstimator.Estimate(ctx, depth, order.MarketID)
    
    // Simulate with configurable parameters
    rng := rand.New(rand.NewSource(time.Now().UnixNano()))
    filled := rng.Float64() < fillProb.InexactFloat64()
    
    if !filled {
        return &ports.SimulatedFill{Filled: false, Status: "SIMULATED_NO_FILL"}
    }
    
    // Apply slippage
    slippage := order.Price.Mul(decimal.NewFromFloat(0.01)) // 1% default
    fillPrice := order.Price.Sub(slippage)
    
    return &ports.SimulatedFill{
        Filled:    true,
        Status:    "SIMULATED_FILL",
        FillPrice: fillPrice,
        Quantity:  order.Quantity,
        LatencyMs: int64(rng.Intn(100)), // Simulated latency
    }
}
```

### Redis Key

```
Key: pqap:execution_mode
Values: "LIVE" | "PAPER"
TTL: None (persistent)
Set by: API gateway PUT /api/execution-mode (admin only)
Read by: execution engine before every order
```

### PnL Calculation

Paper PnL uses same formula as live:
```
PnL = (exit_price - entry_price) * quantity  // for YES buy
PnL = (entry_price - exit_price) * quantity  // for NO buy
```

Tracked in `paper_positions.pnl` and `paper_trades.pnl`.

### Prometheus Metrics

```
pqap_paper_trades_total          # Counter — paper trades executed
pqap_paper_positions_open        # Gauge — open paper positions
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
