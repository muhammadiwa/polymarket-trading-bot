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
  - [ ] Add `side` filter (YES/NO) to existing `get_trades_in_range()`
  - [ ] Add `pnl_sign` filter (positive/negative/zero) to `get_trades_in_range()`
  - [ ] Add `fill_status` filter to `get_trades_in_range()`
  - [ ] Ensure all filters are combinable
- [ ] Task 2: Add JSON Export (AC: #2)
  - [ ] Add GET /api/analytics/export/json endpoint
  - [ ] Return JSON array of trade objects
  - [ ] Support same filters as CSV export
- [ ] Task 3: Extend Existing CSV Export (AC: #2)
  - [ ] Add new filters to existing GET /api/analytics/export
  - [ ] Ensure all fields are included (already done in Story 4.2)
- [ ] Task 4: Frontend Filter UI
  - [ ] Add filter controls to existing analytics page
  - [ ] Side filter (YES/NO/all)
  - [ ] PnL sign filter (positive/negative/all)
  - [ ] JSON download button

## Dev Notes

### Architecture Context

- **Service:** `analytics` (Python/FastAPI) — extends existing service
- **Database:** PostgreSQL `trades` table — existing queries
- **Frontend:** Dashboard analytics page — extends existing
- **Pattern:** Extend existing `get_trades_in_range()` and export endpoints

### Files to MODIFY

**`services/analytics/app/repos/analytics_repo.py`**
- Current: `get_trades_in_range()` has `strategy_id`, `market_id` filters
- Change: Add `side`, `pnl_sign`, `fill_status` filters
- Preserve: All existing filter logic

**`services/analytics/app/routes/analytics.py`**
- Current: `/export` endpoint with CSV streaming
- Change: Add `/export/json` endpoint, add new filter params to both endpoints
- Preserve: Existing CSV export logic

### Filter Parameters

| Parameter | Type | Values | Default |
|-----------|------|--------|---------|
| start_date | str | ISO 8601 | required |
| end_date | str | ISO 8601 | required |
| strategy_id | str | UUID | optional |
| market_id | str | string | optional |
| side | str | "YES", "NO" | optional |
| pnl_sign | str | "positive", "negative", "zero" | optional |
| fill_status | str | "FILLED", "PARTIAL_FILL" | optional |

### References

| Reference | Description |
|-----------|-------------|
| FR-63 | Trade history filtering by date, market, strategy, side, PnL sign |
| FR-64 | CSV and JSON export |
| NFR-TH1 | Query response <1s for 10k trades |

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
