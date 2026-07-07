# Story 4.5: Orderbook Viewer — Real-time Display & Depth Chart

Status: ready-for-dev

## Story

As a quant trader,
I want to view the real-time orderbook with depth chart and recent trades for any market,
So that I can analyze market microstructure and make informed decisions.

## Acceptance Criteria

1. **Given** the user selects a market in the orderbook viewer
   **When** the orderbook renders
   **Then** bids, asks, and spread are displayed in real-time
   **And** updates arrive within 100ms of Polymarket data
   **And** the orderbook matches the Polymarket CLOB API state

2. **Given** the orderbook is displayed
   **When** the depth chart renders
   **Then** cumulative bid/ask at each price level is visualized
   **And** the chart updates in real-time, accurate within 1%

3. **Given** recent trades are requested
   **When** the trade list renders
   **Then** the last 100 trades are displayed with price, size, timestamp
   **And** the list updates in real-time

4. **Given** multiple markets are viewed
   **When** up to 5 market tabs are open simultaneously
   **Then** each tab is independent (no cross-contamination)
   **And** memory per tab stays under 100MB

## Tasks / Subtasks

- [ ] Task 1: Orderbook API Endpoint (AC: #1)
  - [ ] Add GET /api/orderbook/{market_id} to api-gateway
  - [ ] Fetch orderbook data from Polymarket CLOB API
  - [ ] Return bids, asks, spread, last update timestamp
  - [ ] Cache response for 100ms to avoid API hammering
- [ ] Task 2: Recent Trades API Endpoint (AC: #3)
  - [ ] Add GET /api/orderbook/{market_id}/trades to api-gateway
  - [ ] Fetch recent trades from Polymarket CLOB API
  - [ ] Return last 100 trades: price, size, side, timestamp
- [ ] Task 3: Orderbook Component (AC: #1, #4)
  - [ ] Create `services/dashboard/src/components/orderbook/OrderbookView.tsx`
  - [ ] Bids table (green), asks table (red), spread display
  - [ ] Market selector dropdown
  - [ ] Tab system (up to 5 markets, local state per tab)
- [ ] Task 4: Depth Chart (AC: #2)
  - [ ] Create `DepthChart.tsx` using recharts
  - [ ] Cumulative bid/ask at each price level
  - [ ] Polling every 2s (or WebSocket if available)
- [ ] Task 5: Recent Trades (AC: #3)
  - [ ] Create `RecentTrades.tsx`
  - [ ] Last 100 trades: price, size, timestamp
  - [ ] Polling every 2s

## Dev Notes

### Architecture Context

- **Frontend:** Next.js (dashboard) with recharts
- **Backend:** API Gateway fetches from Polymarket CLOB API
- **Data source:** Polymarket CLOB API (NOT scanner — scanner only has price snapshots, not full orderbook)
- **Pattern:** REST API for snapshots, polling for updates (WebSocket orderbook stream is future enhancement)

### Key Architecture Rules

- **FR-66:** Display real-time orderbook for selected market
- **FR-67:** Display depth chart (cumulative bid/ask)
- **FR-68:** Display recent trades (last 100)
- **FR-69:** Support multiple market tabs (up to 5)
- **NFR-OV1:** Orderbook update latency within 100ms
- **NFR-OV2:** Memory per tab <100MB

### Data Source Clarification

The scanner service (`services/scanner/`) has a `Market` struct with:
- `YESPrice`, `NOPrice`, `Spread`, `Volume24h`, `LiquidityDepth`

This is a **price snapshot**, NOT a full orderbook. For orderbook data (bids/asks at each price level), we need to fetch from **Polymarket CLOB API** directly.

```
Polymarket CLOB API → API Gateway (fetch + cache) → Dashboard (polling)
```

### API Endpoints

| Method | Path | Source | Purpose |
|--------|------|--------|---------|
| GET | `/api/orderbook/{market_id}` | Polymarket CLOB API | Orderbook snapshot (bids, asks, spread) |
| GET | `/api/orderbook/{market_id}/trades` | Polymarket CLOB API | Recent trades (last 100) |

### Orderbook Data Structure

```typescript
interface OrderbookSnapshot {
  market_id: string;
  bids: OrderbookLevel[];  // sorted by price descending
  asks: OrderbookLevel[];  // sorted by price ascending
  spread: string;
  last_update: string;
}

interface OrderbookLevel {
  price: string;
  size: string;
  cumulative: string;
}

interface RecentTrade {
  price: string;
  size: string;
  side: "BUY" | "SELL";
  timestamp: string;
}
```

### Tab System

- Tabs managed via React local state (useState per tab)
- Each tab fetches its own orderbook data independently
- Max 5 tabs enforced at UI level
- Closing a tab cleans up its state and polling interval

### Testing Standards

- Unit tests for orderbook component rendering
- Unit tests for depth chart calculation
- E2E tests: market selection, orderbook display, tab switching
- Performance tests: memory usage per tab <100MB

### References

| Reference | Description |
|-----------|-------------|
| FR-66 | Real-time orderbook display |
| FR-67 | Depth chart visualization |
| FR-68 | Recent trades display |
| FR-69 | Multiple market tabs (up to 5) |
| NFR-OV1 | Update latency within 100ms |
| NFR-OV2 | Memory per tab <100MB |

## Dev Agent Record

### Agent Model Used

mimo-v2.5-pro

### Completion Notes List

- Task 1: GET /api/orderbook/{market_id} — proxies Polymarket CLOB API
- Task 2: GET /api/orderbook/{market_id}/trades — recent trades from CLOB API
- Task 3: OrderbookView with tab system (up to 5 markets)
- Task 4: DepthChart using recharts (bid/ask cumulative)
- Task 5: RecentTrades component with real-time polling

### File List

**New files:**
- `services/api-gateway/app/routes/orderbook.py`
- `services/dashboard/src/app/orderbook/page.tsx`
- `services/dashboard/src/components/orderbook/OrderbookView.tsx`
- `services/dashboard/src/components/orderbook/OrderbookTable.tsx`
- `services/dashboard/src/components/orderbook/DepthChart.tsx`
- `services/dashboard/src/components/orderbook/RecentTrades.tsx`
- `services/dashboard/src/hooks/useOrderbook.ts`

**Modified files:**
- `services/dashboard/src/lib/api.ts` — added orderbook API functions
- `services/dashboard/src/types/index.ts` — added orderbook types
