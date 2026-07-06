# Story 4.2: Analytics — Charts & CSV Export

Status: ready-for-dev

## Story

As a quant trader,
I want interactive performance charts and CSV export capability,
So that I can visualize trends and analyze data in external tools.

## Acceptance Criteria

1. **Given** the user opens the analytics page
   **When** charts render
   **Then** a PnL over time line chart, PnL distribution histogram, and PnL by strategy pie chart are displayed
   **And** charts are interactive (hover for details, zoom)
   **And** charts render within 2 seconds for 1 year of data

2. **Given** the user requests a CSV export
   **When** the export is generated
   **Then** all raw trade data is included with all fields
   **And** the CSV is well-formed and downloadable
   **And** export completes within 10 seconds for 10,000 trades

## Tasks / Subtasks

- [ ] Task 1: Dashboard Analytics Page (AC: #1)
  - [ ] Create `services/dashboard/src/app/analytics/page.tsx`
  - [ ] PnL over time line chart (recharts)
  - [ ] PnL distribution histogram
  - [ ] PnL by strategy pie chart
  - [ ] Interactive: hover tooltips, zoom
- [ ] Task 2: Chart Components (AC: #1)
  - [ ] `PnLLineChart` — daily/weekly/monthly PnL
  - [ ] `PnLHistogram` — PnL distribution buckets
  - [ ] `StrategyPieChart` — PnL by strategy
  - [ ] Responsive layout (desktop-first)
- [ ] Task 3: CSV Export (AC: #2)
  - [ ] Backend: GET /api/analytics/export endpoint
  - [ ] Frontend: download button with progress
  - [ ] Include all trade fields
  - [ ] Streaming response for large datasets
- [ ] Task 4: Performance Optimization (AC: #1, #2)
  - [ ] Charts render <2s for 1 year data
  - [ ] CSV export <10s for 10k trades
  - [ ] Pagination or streaming for large datasets

## Dev Notes

### Architecture Context

- **Frontend:** Next.js (dashboard) with recharts
- **Backend:** Analytics service (Python/FastAPI)
- **Data:** PostgreSQL trades table
- **Pattern:** Charts use pre-aggregated data from /api/analytics/pnl; CSV uses raw trade data

### Key Architecture Rules

- **FR-59:** PnL over time line chart, PnL distribution histogram, PnL by strategy pie chart
- **FR-60:** CSV export with all raw trade data
- **NFR-AN2:** Charts render within 2s for 1 year data

### Chart Data Sources

| Chart | API Endpoint | Data |
|-------|-------------|------|
| PnL Line | GET /api/analytics/pnl?group_by=day | Daily PnL series |
| PnL Histogram | GET /api/analytics/pnl?group_by=day | Bucket PnL values |
| Strategy Pie | GET /api/analytics/pnl?group_by=strategy | PnL per strategy |

### CSV Export Schema

```csv
timestamp,market_id,market_slug,strategy_id,side,price,quantity,filled_quantity,pnl,fee,slippage_pct,fill_status,latency_ms
2025-01-15T10:30:00Z,market-123,will-btc-hit-100k,strat-1,YES,0.65,100,100,12.50,0.10,0.05,FILLED,45
```

### Prometheus Metrics

```
pqap_analytics_charts_rendered_total    # Counter — chart renders
pqap_analytics_chart_latency_ms         # Histogram — chart render time
pqap_analytics_csv_exports_total        # Counter — CSV exports
pqap_analytics_csv_export_latency_ms    # Histogram — CSV export time
```

### Testing Standards

- Unit tests for chart data transformation
- Unit tests for CSV generation (correct fields, escaping)
- E2E tests: page loads, charts render, CSV downloads
- Performance tests: 1 year data <2s, 10k trades <10s

### References

| Reference | Description |
|-----------|-------------|
| FR-59 | PnL charts (line, histogram, pie) |
| FR-60 | CSV export with all fields |
| NFR-AN2 | Chart render <2s for 1 year data |

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
