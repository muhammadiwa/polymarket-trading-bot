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
   **And** the orderbook matches the Polymarket API state

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
  - [ ] Proxy scanner service orderbook data
  - [ ] Return bids, asks, spread, last update timestamp
- [ ] Task 2: WebSocket Orderbook Stream (AC: #1)
  - [ ] Extend WS endpoint to stream orderbook updates
  - [ ] Subscribe to scanner orderbook events via NATS
  - [ ] Forward to dashboard WebSocket clients
- [ ] Task 3: Orderbook Component (AC: #1, #4)
  - [ ] Create `services/dashboard/src/components/orderbook/OrderbookView.tsx`
  - [ ] Bids table (green), asks table (red), spread display
  - [ ] Market selector dropdown
  - [ ] Tab system (up to 5 markets)
- [ ] Task 4: Depth Chart (AC: #2)
  - [ ] Create `DepthChart.tsx` using recharts
  - [ ] Cumulative bid/ask at each price level
  - [ ] Real-time updates via WebSocket
- [ ] Task 5: Recent Trades (AC: #3)
  - [ ] Create `RecentTrades.tsx`
  - [ ] Last 100 trades: price, size, timestamp
  - [ ] Real-time updates via WebSocket

## Dev Notes

### Architecture Context

- **Frontend:** Next.js (dashboard) with recharts
- **Backend:** API Gateway proxies scanner service
- **Data source:** Scanner service (Go) — already has orderbook data from Polymarket WebSocket
- **Event bus:** NATS for orderbook updates
- **Pattern:** Dashboard subscribes to orderbook events via WebSocket

### Key Architecture Rules

- **FR-66:** Display real-time orderbook for selected market
- **FR-67:** Display depth chart (cumulative bid/ask)
- **FR-68:** Display recent trades (last 100)
- **FR-69:** Support multiple market tabs (up to 5)
- **NFR-OV1:** Orderbook update latency within 100ms
- **NFR-OV2:** Memory per tab <100MB

### Data Flow

```
Polymarket WS → Scanner (Go) → NATS → API Gateway → WebSocket → Dashboard
```

### API Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/orderbook/{market_id}` | Snapshot of current orderbook |
| WS | `/ws/orderbook/{market_id}` | Real-time orderbook stream |

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

### Debug Log References

### Completion Notes List

### File List
