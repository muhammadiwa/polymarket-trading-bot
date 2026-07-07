# Story 4.4: Trade History — Filtering & Export

Status: ready-for-dev

## Story

As a quant trader,
I want to filter and export my trade history,
So that I can analyze specific subsets of trades and share data externally.

## Acceptance Criteria

1. **Given** trade history records exist
   **When** the user applies filters (date range, market, strategy, side, PnL sign)
   **Then** results are returned within 1 second for 10,000 trades
   **And** filters are combinable (e.g., date range + strategy + winning trades only)
   **And** results are accurate and match the filter criteria

2. **Given** the user requests a CSV or JSON export
   **When** the export is generated
   **Then** all fields are included in the export
   **And** the file is well-formed
   **And** export completes within 10 seconds for 10,000 trades

## Tasks / Subtasks

- [ ] Task 1: Extend Trade Filtering (AC: #1)
  - [ ] Add `side` filter (YES/NO) to `get_trades_in_range()` SQL query
  - [ ] Add `pnl_sign` filter (positive/negative/zero) to SQL query
  - [ ] Ensure all filters are combinable (AND logic)
- [ ] Task 2: Add JSON Export (AC: #2)
  - [ ] Add GET /api/analytics/export/json endpoint
  - [ ] Return JSON array of trade objects
  - [ ] Support same filters as CSV export
- [ ] Task 3: Extend Existing CSV Export (AC: #2)
  - [ ] Add `side` and `pnl_sign` filter params to existing GET /api/analytics/export
- [ ] Task 4: Frontend Filter UI
  - [ ] Add filter controls to existing analytics page
  - [ ] Side filter (YES/NO/all)
  - [ ] PnL sign filter (positive/negative/all)
  - [ ] JSON download button

## Dev Notes

### Architecture Context

- **Service:** `analytics` (Python/FastAPI) — extends existing service
- **Database:** PostgreSQL `trades` table — extend existing queries
- **Frontend:** Dashboard analytics page — extends existing
- **Pattern:** Extend existing `get_trades_in_range()` and export endpoints

### Files to MODIFY

**`services/analytics/app/repos/analytics_repo.py`**
- Current: `get_trades_in_range()` has `strategy_id`, `market_id` filters, hardcoded `fill_status IN ('FILLED', 'PARTIAL_FILL')`
- Change: Add `side` and `pnl_sign` SQL filters
- Preserve: Existing filter logic and fill_status filter

**`services/analytics/app/routes/analytics.py`**
- Current: `/export` endpoint with CSV streaming
- Change: Add `/export/json` endpoint, add `side` and `pnl_sign` params to export endpoints
- Preserve: Existing CSV export logic

### Filter Implementation

```python
# In get_trades_in_range():
if side:
    conditions.append(f"side = ${idx}")
    params.append(side)
    idx += 1

# pnl_sign: SQL filter (more efficient than Python post-filter)
if pnl_sign == "positive":
    conditions.append("pnl > 0")
elif pnl_sign == "negative":
    conditions.append("pnl < 0")
elif pnl_sign == "zero":
    conditions.append("pnl = 0")
# If pnl_sign is None, no filter applied (all trades)
```

### Filter Parameters

| Parameter | Type | Values | Default | SQL |
|-----------|------|--------|---------|-----|
| start_date | str | ISO 8601 | required | `fill_timestamp BETWEEN` |
| end_date | str | ISO 8601 | required | `fill_timestamp BETWEEN` |
| strategy_id | str | UUID | optional | `strategy_id = $N` |
| market_id | str | string | optional | `market_id = $N` |
| side | str | "YES", "NO" | optional | `side = $N` |
| pnl_sign | str | "positive", "negative", "zero" | optional | `pnl > 0` / `< 0` / `= 0` |

### Note: fill_status

`fill_status` is already hardcoded in `get_trades_in_range()` as `IN ('FILLED', 'PARTIAL_FILL')`. This is correct — we only want filled trades for analytics. No change needed.

### References

| Reference | Description |
|-----------|-------------|
| FR-63 | Trade history filtering by date, market, strategy, side, PnL sign |
| FR-64 | CSV and JSON export |
| NFR-TH1 | Query response <1s for 10k trades |

## Dev Agent Record

### Agent Model Used

mimo-v2.5-pro

### Completion Notes List

- Task 1: Added side and pnl_sign SQL filters to get_trades_in_range()
- Task 2: Added GET /api/analytics/export/json endpoint
- Task 3: Extended CSV export with side and pnl_sign filters
- Task 4: Added filter UI (side, pnl_sign dropdowns) + JSON download button

### File List

**Modified files:**
- `services/analytics/app/repos/analytics_repo.py` — added side and pnl_sign filters
- `services/analytics/app/routes/analytics.py` — added /export/json endpoint, extended /export with filters
- `services/dashboard/src/app/analytics/page.tsx` — added filter controls + JSON button
- `services/dashboard/src/lib/api.ts` — added downloadJSON, updated downloadCSV with filters
