# Story 5.2: Backtesting — Metrics, Reports & Parameter Sweeps

Status: ready-for-dev

## Story

As a quant trader,
I want comprehensive backtest metrics, detailed reports, and the ability to test multiple parameter configurations in batch,
So that I can find the optimal strategy parameters before going live.

## Acceptance Criteria

1. **Given** a backtest completes
   **When** the results are calculated
   **Then** all performance metrics are computed: PnL (total, daily), win rate, Sharpe ratio, drawdown, profit factor
   **And** metrics match the Analytics service calculations

2. **Given** a backtest report is requested
   **When** the report is generated
   **Then** it includes all metrics, charts (PnL curve, drawdown chart), and trade-by-trade breakdown
   **And** the report is exportable

3. **Given** the user wants to test multiple parameter configurations
   **When** a parameter sweep is initiated (e.g., test 100 different threshold values)
   **Then** all configurations are tested in batch
   **And** the sweep completes within 1 hour for 100 configurations on 1 year of data
   **And** results are ranked by the selected metric (e.g., Sharpe ratio)

## Tasks / Subtasks

- [ ] Task 1: Enhanced Metrics (AC: #1)
  - [ ] Add daily PnL breakdown to backtest results
  - [ ] Add VaR 95% calculation
  - [ ] Ensure metrics match analytics service (reuse Decimal precision)
- [ ] Task 2: Report Generation (AC: #2)
  - [ ] Add GET /api/backtesting/{id}/report endpoint
  - [ ] Include PnL curve data, drawdown chart data, trade breakdown
  - [ ] Export as JSON (frontend renders charts)
- [ ] Task 3: Parameter Sweeps (AC: #3)
  - [ ] Add POST /api/backtesting/sweep endpoint
  - [ ] Accept parameter ranges (min, max, step)
  - [ ] Run batch backtests in parallel (asyncio.gather)
  - [ ] Rank results by selected metric
  - [ ] Return ranked results

## Dev Notes

### Architecture Context

- **Service:** `backtesting` (Python/FastAPI) — extends Story 5.1
- **Database:** PostgreSQL backtest_runs table — extends existing
- **Pattern:** Reuse existing backtest engine for parameter sweeps

### Files to MODIFY

**`services/backtesting/app/engine/backtest_engine.py`**
- Current: Returns BacktestResults with summary + trades + warnings
- Change: Add daily PnL breakdown, VaR calculation
- Preserve: Existing PnL and Sharpe calculations

**`services/backtesting/app/routes/backtest.py`**
- Current: POST /run, GET /status, GET /results
- Change: Add GET /report, POST /sweep
- Preserve: Existing endpoints

**`services/backtesting/app/models/backtest.py`**
- Current: BacktestRequest, BacktestStatus, BacktestResults
- Change: Add SweepRequest, SweepResults, BacktestReport
- Preserve: Existing models

### Parameter Sweep Schema

```python
class SweepParameter(BaseModel):
    name: str  # e.g., "min_profit_threshold"
    min_value: float
    max_value: float
    step: float

class SweepRequest(BaseModel):
    strategy_id: str
    start_date: str
    end_date: str
    parameters: list[SweepParameter]
    rank_by: str = "sharpe_ratio"  # sharpe_ratio, total_pnl, win_rate, max_drawdown
    simulation: SimulationConfig = SimulationConfig()

class SweepResult(BaseModel):
    parameters: dict
    summary: BacktestSummary

class SweepResults(BaseModel):
    results: list[SweepResult]
    best: SweepResult
    total_configs: int
```

### Report Schema

```python
class BacktestReport(BaseModel):
    run_id: str
    summary: BacktestSummary
    pnl_curve: list[dict]  # [{date, cumulative_pnl}]
    drawdown_curve: list[dict]  # [{date, drawdown}]
    trades: list[BacktestTrade]
    warnings: list[dict]
```

### References

| Reference | Description |
|-----------|-------------|
| FR-87 | All performance metrics computed |
| FR-88 | Parameter sweeps in batch |
| FR-89 | Detailed reports with charts |

## Dev Agent Record

### Agent Model Used

mimo-v2.5-pro

### Completion Notes List

- Task 1: Added daily PnL breakdown, VaR 95% to backtest results
- Task 2: Added GET /{id}/report endpoint with PnL and drawdown curves
- Task 3: Added POST /sweep endpoint for parameter sweeps (max 1000 combos)

### File List

**Modified files:**
- `services/backtesting/app/engine/backtest_engine.py` — daily PnL, VaR, report generation
- `services/backtesting/app/routes/backtest.py` — added /report and /sweep endpoints
- `services/backtesting/app/models/backtest.py` — added SweepRequest, SweepResults, BacktestReport
