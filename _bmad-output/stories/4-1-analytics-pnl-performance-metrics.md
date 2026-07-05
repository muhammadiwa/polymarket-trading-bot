# Story 4.1: Analytics — PnL & Performance Metrics

Status: ready-for-dev

## Story

As a quant trader,
I want comprehensive PnL and performance metrics calculated from my trade history,
So that I can evaluate my trading performance across time periods and strategies.

## Acceptance Criteria

1. **Given** trade history exists in PostgreSQL and TimescaleDB
   **When** the user queries PnL analytics
   **Then** PnL is calculated and displayed by: day, week, month, strategy, market
   **And** all aggregations are accurate within $0.01
   **And** queries support arbitrary date ranges

2. **Given** performance metrics are requested
   **When** the analytics service calculates them
   **Then** the following metrics are returned: win rate, average win, average loss, profit factor, Sharpe ratio
   **And** calculations match manual verification within 1%
   **And** metrics are available for any date range combination

3. **Given** risk metrics are requested
   **When** the analytics service calculates them
   **Then** max drawdown, current drawdown, and VaR (95%, parametric method) are returned
   **And** drawdown is accurate within 1%
   **And** all financial calculations use Decimal precision (never float64)

## Tasks / Subtasks

- [ ] Task 1: Analytics Service Setup (AC: #1, #2, #3)
  - [ ] Create `services/analytics/` Python/FastAPI service
  - [ ] Connect to PostgreSQL (trades) and TimescaleDB (opportunities)
  - [ ] JWT authentication on all endpoints
- [ ] Task 2: PnL Aggregation (AC: #1)
  - [ ] Calculate PnL by day, week, month
  - [ ] Calculate PnL by strategy and market
  - [ ] Support arbitrary date range queries
  - [ ] Decimal precision for all calculations
- [ ] Task 3: Performance Metrics (AC: #2)
  - [ ] Win rate (wins / total trades)
  - [ ] Average win / average loss
  - [ ] Profit factor (gross profit / gross loss)
  - [ ] Sharpe ratio (annualized)
- [ ] Task 4: Risk Metrics (AC: #3)
  - [ ] Max drawdown
  - [ ] Current drawdown
  - [ ] VaR (95%, parametric method)
  - [ ] All calculations use Decimal
- [ ] Task 5: API Endpoints
  - [ ] GET /api/analytics/pnl (by period, strategy, market)
  - [ ] GET /api/analytics/metrics (win rate, Sharpe, etc.)
  - [ ] GET /api/analytics/risk (drawdown, VaR)
  - [ ] GET /api/analytics/summary (combined overview)

## Dev Notes

### Architecture Context

- **Service:** New `analytics` (Python/FastAPI)
- **Database:** PostgreSQL (trades table from Epic 1), TimescaleDB (opportunities)
- **Pattern:** Read-only analytics service, no state mutation
- **Precision:** All financial calculations use Decimal (INF-11)

### Key Architecture Rules

- **FR-56:** PnL by day, week, month, strategy, market
- **FR-57:** Win rate, average win/loss, profit factor, Sharpe ratio
- **FR-58:** Max drawdown, current drawdown, VaR (95%)
- **AD-7:** TimescaleDB for time-series queries
- **INF-11:** Decimal precision for all monetary values
- **NFR-AN1:** Financial calculation accuracy $0.01

### Database Queries

```sql
-- PnL by day
SELECT DATE(created_at) as date, SUM(realized_pnl) as daily_pnl
FROM trades WHERE created_at BETWEEN $1 AND $2
GROUP BY DATE(created_at) ORDER BY date

-- PnL by strategy
SELECT strategy_id, SUM(realized_pnl) as total_pnl, COUNT(*) as trade_count
FROM trades WHERE created_at BETWEEN $1 AND $2
GROUP BY strategy_id

-- Win rate
SELECT
  COUNT(*) FILTER (WHERE realized_pnl > 0) as wins,
  COUNT(*) as total,
  COUNT(*) FILTER (WHERE realized_pnl > 0)::float / COUNT(*) as win_rate
FROM trades WHERE created_at BETWEEN $1 AND $2
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
- Integration tests with PostgreSQL
- Accuracy tests: verify within $0.01 tolerance

### References

| Reference | Description |
|-----------|-------------|
| FR-56 | PnL by day, week, month, strategy, market |
| FR-57 | Win rate, average win/loss, profit factor, Sharpe ratio |
| FR-58 | Max drawdown, current drawdown, VaR (95%) |
| AD-7 | TimescaleDB for time-series |
| INF-11 | Decimal precision for all monetary values |
| NFR-AN1 | Financial calculation accuracy $0.01 |

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
