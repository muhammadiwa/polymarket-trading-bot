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
   **And** charts are interactive (hover tooltips, scroll-to-zoom, click-drag selection)
   **And** charts render within 2 seconds for 1 year of data (~10,000 trades)

2. **Given** the user requests a CSV export
   **When** the export is generated
   **Then** all raw trade data is included with all fields
   **And** the CSV is well-formed and downloadable
   **And** export completes within 10 seconds for 10,000 trades

## Tasks / Subtasks

- [ ] Task 1: Backend — CSV Export Endpoint (AC: #2)
  - [ ] Add GET /api/analytics/export to analytics service
  - [ ] Streaming response (FastAPI StreamingResponse)
  - [ ] Include all trades table fields
  - [ ] Proper CSV escaping (quotes, commas, newlines)
- [ ] Task 2: Backend — Histogram Data Endpoint (AC: #1)
  - [ ] Add GET /api/analytics/histogram to analytics service
  - [ ] Return raw PnL values for frontend binning
  - [ ] Support date range filtering
- [ ] Task 3: Frontend — Analytics Page (AC: #1)
  - [ ] Create `services/dashboard/src/app/analytics/page.tsx`
  - [ ] Install recharts dependency
  - [ ] Responsive layout (desktop-first, 1024px+)
- [ ] Task 4: Frontend — Chart Components (AC: #1)
  - [ ] `PnLLineChart` — daily PnL line chart from /api/analytics/pnl?group_by=day
  - [ ] `PnLHistogram` — PnL distribution from /api/analytics/histogram
  - [ ] `StrategyPieChart` — PnL by strategy from /api/analytics/pnl?group_by=strategy
  - [ ] Interactive: hover tooltips, scroll-to-zoom, click-drag selection
- [ ] Task 5: Frontend — CSV Export (AC: #2)
  - [ ] Download button triggers GET /api/analytics/export
  - [ ] Progress indicator during download
  - [ ] File download via browser blob

## Dev Notes

### Architecture Context

- **Frontend:** Next.js (dashboard) with recharts
- **Backend:** Analytics service (Python/FastAPI)
- **Data:** PostgreSQL trades table
- **Pattern:** Charts use pre-aggregated data from /api/analytics/pnl; histogram uses raw values; CSV streams raw trades

### Key Architecture Rules

- **FR-59:** PnL over time line chart, PnL distribution histogram, PnL by strategy pie chart
- **FR-60:** CSV export with all raw trade data
- **NFR-AN2:** Charts render within 2s for 1 year data

### Chart Data Sources

| Chart | API Endpoint | Data |
|-------|-------------|------|
| PnL Line | GET /api/analytics/pnl?group_by=day | Daily PnL series |
| PnL Histogram | GET /api/analytics/histogram | Raw PnL values for binning |
| Strategy Pie | GET /api/analytics/pnl?group_by=strategy | PnL per strategy |

### Histogram Binning Logic

Frontend bins raw PnL values into buckets:
```typescript
// Default: 20 buckets from min to max PnL
const binCount = 20;
const min = Math.min(...pnls);
const max = Math.max(...pnls);
const binWidth = (max - min) / binCount;
// Count values in each bin
```

### CSV Export — Backend Implementation

```python
# FastAPI StreamingResponse for large datasets
from fastapi.responses import StreamingResponse
import csv
import io

@router.get("/export")
async def export_trades(
    start_date: str = Query(...),
    end_date: str = Query(...),
    _user: dict = Depends(verify_jwt),
):
    async def generate():
        yield "timestamp,market_id,market_slug,strategy_id,side,price,quantity,filled_quantity,pnl,fee,slippage_pct,fill_status,latency_ms\n"
        # Stream rows in batches of 1000
        async with pool.acquire() as conn:
            async with conn.transaction():
                async for row in conn.cursor(query, *params):
                    yield f"{row['fill_timestamp'].isoformat()},{row['market_id']},...\n"

    return StreamingResponse(generate(), media_type="text/csv",
                             headers={"Content-Disposition": "attachment; filename=trades.csv"})
```

### CSV Fields (from trades table)

| CSV Column | DB Column | Type |
|-----------|-----------|------|
| timestamp | fill_timestamp | TIMESTAMPTZ (ISO 8601) |
| market_id | market_id | VARCHAR |
| market_slug | market_slug | VARCHAR |
| strategy_id | strategy_id | VARCHAR |
| side | side | VARCHAR (YES/NO) |
| price | price | NUMERIC(12,4) |
| quantity | quantity | NUMERIC(20,8) |
| filled_quantity | filled_quantity | NUMERIC(20,8) |
| pnl | pnl | NUMERIC(20,8) |
| fee | fee | NUMERIC(12,4) |
| slippage_pct | slippage_pct | NUMERIC(8,4) |
| fill_status | fill_status | VARCHAR |
| latency_ms | latency_ms | INTEGER |

### CSV Escaping Rules

- Fields containing commas, quotes, or newlines wrapped in double quotes
- Double quotes within fields escaped as `""`
- NULL values exported as empty string

### Interactive Chart Behavior

| Feature | Implementation |
|---------|---------------|
| Hover tooltips | recharts `<Tooltip>` component |
| Scroll-to-zoom | recharts `<CartesianGrid>` + custom zoom handler |
| Click-drag selection | recharts `<Brush>` component for range selection |
| Responsive | recharts `<ResponsiveContainer>` |

### Frontend Dependencies to Install

```json
{
  "recharts": "^2.15.0"
}
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
- Unit tests for CSV generation (correct fields, escaping, special chars)
- E2E tests: page loads, charts render, CSV downloads
- Performance tests: 10k trades charts <2s, CSV <10s

### References

| Reference | Description |
|-----------|-------------|
| FR-59 | PnL charts (line, histogram, pie) |
| FR-60 | CSV export with all fields |
| NFR-AN2 | Chart render <2s for 1 year data |

## Dev Agent Record

### Agent Model Used

mimo-v2.5-pro

### Completion Notes List

- Task 1: Backend — /api/analytics/export (StreamingResponse CSV) + /api/analytics/histogram
- Task 2: Frontend — PnLLineChart, PnLHistogram, StrategyPieChart (recharts)
- Task 3: Frontend — Analytics page with date picker + CSV download
- Task 4: Frontend — useAnalytics hook + API functions + types

### File List

**New files:**
- `services/dashboard/src/app/analytics/page.tsx`
- `services/dashboard/src/components/charts/PnLLineChart.tsx`
- `services/dashboard/src/components/charts/PnLHistogram.tsx`
- `services/dashboard/src/components/charts/StrategyPieChart.tsx`
- `services/dashboard/src/hooks/useAnalytics.ts`

**Modified files:**
- `services/analytics/app/routes/analytics.py` — added histogram + export endpoints
- `services/dashboard/src/lib/api.ts` — added analytics API functions
- `services/dashboard/src/types/index.ts` — added analytics types
