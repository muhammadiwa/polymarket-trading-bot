# Story 4.1: Analytics — PnL & Performance Metrics

Status: ready-for-dev

## Story

As a quant trader,
I want comprehensive PnL and performance metrics calculated from my trade history,
So that I can evaluate my trading performance across time periods and strategies.

## Acceptance Criteria

1. **Given** trade history exists in PostgreSQL (`trades` table)
   **When** the user queries PnL analytics
   **Then** PnL is calculated and displayed by: day, week, month, strategy, market
   **And** all aggregations are accurate within $0.01
   **And** queries support arbitrary date ranges
   **And** zero trades in range returns empty result (not error)

2. **Given** performance metrics are requested
   **When** the analytics service calculates them
   **Then** the following metrics are returned: win rate, average win, average loss, profit factor, Sharpe ratio
   **And** calculations match manual verification within 1%
   **And** metrics are available for any date range combination
   **And** profit factor handles all-wins (loss=0) gracefully

3. **Given** risk metrics are requested
   **When** the analytics service calculates them
   **Then** max drawdown, current drawdown, and VaR (95%, parametric method) are returned
   **And** drawdown is accurate within 1%
   **And** all financial calculations use Decimal precision (never float64)

## Tasks / Subtasks

- [ ] Task 1: Analytics Service Setup (AC: #1, #2, #3)
  - [ ] Create `services/analytics/` Python/FastAPI service
  - [ ] Connect to PostgreSQL (trades table)
  - [ ] JWT authentication on all endpoints
- [ ] Task 2: PnL Aggregation (AC: #1)
  - [ ] Calculate PnL by day, week, month (using `fill_timestamp`)
  - [ ] Calculate PnL by strategy and market
  - [ ] Support arbitrary date range queries
  - [ ] Decimal precision for all calculations
  - [ ] Handle empty trades (return empty result)
- [ ] Task 3: Performance Metrics (AC: #2)
  - [ ] Win rate (wins / total trades, Decimal)
  - [ ] Average win / average loss (Decimal)
  - [ ] Profit factor (gross profit / gross loss, handle loss=0 → return "infinity" or large number)
  - [ ] Sharpe ratio (annualized, see formula below)
- [ ] Task 4: Risk Metrics (AC: #3)
  - [ ] Max drawdown from peak equity (Decimal)
  - [ ] Current drawdown from peak (Decimal)
  - [ ] VaR 95% parametric (see formula below)
  - [ ] All calculations use Decimal
- [ ] Task 5: API Endpoints
  - [ ] GET /api/analytics/pnl (by period, strategy, market)
  - [ ] GET /api/analytics/metrics (win rate, Sharpe, etc.)
  - [ ] GET /api/analytics/risk (drawdown, VaR)
  - [ ] GET /api/analytics/summary (combined overview)

## Dev Notes

### Architecture Context

- **Service:** New `analytics` (Python/FastAPI)
- **Database:** PostgreSQL (`trades` table from Epic 1)
- **Pattern:** Read-only analytics service, no state mutation
- **Precision:** All financial calculations use Decimal (INF-11)
- **Timestamp:** Use `fill_timestamp` for PnL aggregation (when trade was actually filled)

### Key Architecture Rules

- **FR-56:** PnL by day, week, month, strategy, market
- **FR-57:** Win rate, average win/loss, profit factor, Sharpe ratio
- **FR-58:** Max drawdown, current drawdown, VaR (95%)
- **INF-11:** Decimal precision for all monetary values
- **NFR-AN1:** Financial calculation accuracy $0.01

### Database Schema (trades table)

```sql
-- Key columns for analytics:
-- pnl NUMERIC(20,8) — Realized PnL (NOT realized_pnl)
-- strategy_id VARCHAR(64) — Strategy identifier
-- market_id VARCHAR(128) — Market identifier
-- market_slug VARCHAR(256) — Market display name
-- side VARCHAR(4) — 'YES' or 'NO'
-- fill_status VARCHAR(16) — 'FILLED', 'PARTIAL_FILL', etc.
-- fill_timestamp TIMESTAMPTZ — When trade was filled (use for PnL aggregation)
-- created_at TIMESTAMPTZ — Record creation time
-- quantity NUMERIC(20,8) — Trade quantity
-- price NUMERIC(12,4) — Trade price
```

### Database Queries (Corrected)

```sql
-- PnL by day (use fill_timestamp, filter only filled trades)
SELECT DATE(fill_timestamp) as date, SUM(pnl) as daily_pnl, COUNT(*) as trade_count
FROM trades
WHERE fill_timestamp BETWEEN $1 AND $2
  AND fill_status IN ('FILLED', 'PARTIAL_FILL')
GROUP BY DATE(fill_timestamp)
ORDER BY date

-- PnL by strategy
SELECT strategy_id, SUM(pnl) as total_pnl, COUNT(*) as trade_count
FROM trades
WHERE fill_timestamp BETWEEN $1 AND $2
  AND fill_status IN ('FILLED', 'PARTIAL_FILL')
GROUP BY strategy_id

-- PnL by market
SELECT market_id, market_slug, SUM(pnl) as total_pnl, COUNT(*) as trade_count
FROM trades
WHERE fill_timestamp BETWEEN $1 AND $2
  AND fill_status IN ('FILLED', 'PARTIAL_FILL')
GROUP BY market_id, market_slug

-- Win rate (calculated in Python with Decimal, NOT SQL float)
SELECT pnl FROM trades
WHERE fill_timestamp BETWEEN $1 AND $2
  AND fill_status IN ('FILLED', 'PARTIAL_FILL')
```

### Metric Formulas

**Win Rate:**
```python
from decimal import Decimal
wins = sum(1 for pnl in pnls if pnl > 0)
total = len(pnls)
win_rate = Decimal(wins) / Decimal(total) if total > 0 else Decimal(0)
```

**Average Win / Average Loss:**
```python
wins = [pnl for pnl in pnls if pnl > 0]
losses = [pnl for pnl in pnls if pnl < 0]
avg_win = sum(wins) / len(wins) if wins else Decimal(0)
avg_loss = sum(losses) / len(losses) if losses else Decimal(0)
```

**Profit Factor:**
```python
gross_profit = sum(pnl for pnl in pnls if pnl > 0)
gross_loss = abs(sum(pnl for pnl in pnls if pnl < 0))
profit_factor = gross_profit / gross_loss if gross_loss > 0 else Decimal("999999")  # Handle all-wins
```

**Sharpe Ratio (Annualized):**
```python
# Risk-free rate: 0% (configurable via env var)
# Formula: (mean_return - risk_free) / std_return * sqrt(365)
from decimal import Decimal
import math
returns = [pnl for pnl in pnls]  # Daily returns
mean_return = sum(returns) / len(returns)
std_return = (sum((r - mean_return)**2 for r in returns) / len(returns)).sqrt()
risk_free = Decimal(0)  # Configurable
sharpe = (mean_return - risk_free) / std_return * Decimal(math.sqrt(365)) if std_return > 0 else Decimal(0)
```

**Max Drawdown:**
```python
# Calculate from cumulative PnL series
peak = Decimal(0)
max_dd = Decimal(0)
cumulative = Decimal(0)
for pnl in daily_pnls:
    cumulative += pnl
    if cumulative > peak:
        peak = cumulative
    dd = (peak - cumulative) / peak if peak > 0 else Decimal(0)
    if dd > max_dd:
        max_dd = dd
```

**VaR 95% (Parametric):**
```python
# VaR = mean - z * std_dev (z = 1.645 for 95%)
from decimal import Decimal
z_score = Decimal("1.645")
mean_return = sum(returns) / len(returns)
std_return = (sum((r - mean_return)**2 for r in returns) / len(returns)).sqrt()
var_95 = mean_return - z_score * std_return  # Negative means loss
```

### Prometheus Metrics

```
pqap_analytics_queries_total          # Counter — analytics queries
pqap_analytics_query_latency_ms       # Histogram — query latency
pqap_analytics_calculations_total     # Counter — calculations performed
```

### Testing Standards

- Unit tests for each metric calculation (win rate, Sharpe, drawdown, VaR)
- Unit tests for PnL aggregation by period/strategy/market
- Unit tests for edge cases: empty trades, all wins, all losses, single trade
- Integration tests with PostgreSQL
- Accuracy tests: verify within $0.01 tolerance

### References

| Reference | Description |
|-----------|-------------|
| FR-56 | PnL by day, week, month, strategy, market |
| FR-57 | Win rate, average win/loss, profit factor, Sharpe ratio |
| FR-58 | Max drawdown, current drawdown, VaR (95%) |
| INF-11 | Decimal precision for all monetary values |
| NFR-AN1 | Financial calculation accuracy $0.01 |

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
