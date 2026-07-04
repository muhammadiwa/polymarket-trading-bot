# Story 2.2: Dashboard — Portfolio Overview & Position Display

## Story

As a quant trader,
I want a real-time dashboard showing my portfolio overview and all active positions,
So that I can monitor my trading performance at a glance.

## Status

not-started

## Acceptance Criteria

**Given** the dashboard is loaded and the user is authenticated
**When** the portfolio overview page renders
**Then** it displays: total capital, daily PnL, total PnL, capital utilization rate
**And** data is accurate within 1% of backend values
**And** updates are pushed via WebSocket within 2 seconds of any change

**Given** there are active positions
**When** the positions view renders
**Then** all active positions are displayed with: market, side, entry price, current price, quantity, unrealized PnL
**And** PnL updates in real-time (within 2 seconds of price change)
**And** the position list matches the backend position manager state

**And** the dashboard is responsive — usable on 1024px+ screens with no horizontal scrolling
**And** page loads within 3 seconds on a 3G connection

## Technical Requirements

### Architecture Context

- **Frontend:** Next.js 16.2.10 (LTS) — INF-4
- **API Gateway:** FastAPI 0.139.0 (Python) — INF-3
- **Backend services:** portfolio-manager (Python), position-manager (Go)
- **Event bus:** NATS (for real-time updates)
- **Communication:** API Gateway serves REST endpoints; WebSocket for real-time push to dashboard
- **Pattern:** Dashboard connects to API Gateway via WebSocket for real-time updates; API Gateway aggregates data from portfolio-manager and position-manager

### Key Components to Implement

1. **Portfolio Overview Component** (`dashboard/src/components/portfolio/PortfolioOverview.tsx`)
   - Total capital display with Decimal precision — FR-48
   - Daily PnL (color-coded: green/red)
   - Total PnL (color-coded)
   - Capital utilization rate (percentage bar)
   - Real-time updates via WebSocket (within 2s per NFR-D1)

2. **Position List Component** (`dashboard/src/components/positions/PositionList.tsx`)
   - Table of all active positions — FR-49
   - Columns: market, side, entry price, current price, quantity, unrealized PnL
   - PnL color-coded (green for profit, red for loss)
   - Real-time price and PnL updates (within 2s)
   - Sortable columns
   - Matches backend position manager state

3. **WebSocket Client** (`dashboard/src/lib/websocket.ts`)
   - Connect to API Gateway WebSocket endpoint
   - Handle reconnection with backoff
   - Parse and dispatch real-time updates
   - Support for portfolio and position update channels

4. **API Gateway Endpoints** (`services/api-gateway/`)
   - `GET /api/portfolio/overview` — aggregated portfolio metrics
   - `GET /api/positions` — active positions list
   - `WS /ws/dashboard` — WebSocket for real-time push
   - Authentication middleware (JWT per AD-14)

5. **Portfolio Manager Integration** (`services/portfolio-manager/`)
   - Expose portfolio metrics via internal API
   - Calculate: total capital, daily PnL, total PnL, utilization rate
   - Publish `PortfolioUpdated` events to NATS

6. **Position Manager Integration** (`services/position-manager/`)
   - Expose positions via internal API
   - Publish `PositionUpdated` events to NATS on price/PnL changes

### Data Models

**PortfolioOverview:**
```typescript
interface PortfolioOverview {
  totalCapital: string;        // Decimal string (8dp)
  dailyPnL: string;            // Decimal string (8dp)
  totalPnL: string;            // Decimal string (8dp)
  utilizationRate: string;     // Decimal string (4dp, percentage)
  lastUpdated: string;         // ISO 8601 UTC
}
```

**Position:**
```typescript
interface Position {
  id: string;
  market: string;
  side: 'YES' | 'NO';
  entryPrice: string;          // Decimal string (4dp)
  currentPrice: string;        // Decimal string (4dp)
  quantity: string;            // Decimal string (8dp)
  unrealizedPnL: string;       // Decimal string (8dp)
  updatedAt: string;           // ISO 8601 UTC
}
```

**WebSocket Messages:**
```typescript
interface WSMessage {
  type: 'portfolio_update' | 'position_update';
  payload: PortfolioOverview | Position[];
  timestamp: string;
}
```

### API Endpoints

| API | Method | URL | Purpose |
|-----|--------|-----|---------|
| Portfolio Overview | GET | `/api/portfolio/overview` | Aggregated portfolio metrics |
| Positions | GET | `/api/positions` | Active positions list |
| Dashboard WebSocket | WS | `/ws/dashboard` | Real-time push updates |

### Prometheus Metrics

```
pqap_dashboard_ws_connections_total      # Gauge — active WebSocket connections
pqap_dashboard_ws_messages_sent_total    # Counter — WebSocket messages sent
pqap_dashboard_page_load_ms              # Histogram — page load time
pqap_dashboard_api_latency_ms            # Histogram — API call latency
```

## Implementation Guide

### Step 1: Dashboard Setup

- Initialize Next.js 16 project in `services/dashboard/`
- Add dependencies:
  - `next` 16.2.10
  - `react` 19.x
  - `tailwindcss` — styling
  - `recharts` or `@tanstack/react-table` — data display
  - `ws` or native WebSocket API — real-time connection
- Configure TypeScript with strict mode
- Set up project structure per Next.js 16 conventions

### Step 2: API Gateway WebSocket Endpoint

- Implement WebSocket handler in FastAPI (`services/api-gateway/`)
- On connect, authenticate via JWT (AD-14)
- Subscribe to NATS subjects: `pqap.portfolio.updated`, `pqap.position.updated`
- Forward events to connected dashboard clients
- Handle client disconnect gracefully
- Rate limit connections per user

### Step 3: Portfolio Overview Component

- Fetch initial data from `GET /api/portfolio/overview`
- Subscribe to WebSocket `portfolio_update` messages
- Display metrics with proper formatting:
  - Capital: `$X,XXX.XX` (8dp internally, 2dp display)
  - PnL: `+$X.XX` / `-$X.XX` with color coding
  - Utilization: `XX.X%` with progress bar
- Update display on WebSocket message (within 2s per NFR-D1)
- Handle loading and error states

### Step 4: Position List Component

- Fetch initial data from `GET /api/positions`
- Subscribe to WebSocket `position_update` messages
- Render table with columns: Market, Side, Entry, Current, Qty, PnL
- Color-code PnL (green > 0, red < 0, gray = 0)
- Sort by any column (click header)
- Update in real-time (within 2s)
- Handle empty state (no positions)

### Step 5: Responsive Layout

- Desktop-first layout (1024px+ minimum)
- Portfolio overview at top (card layout)
- Position list below (table layout)
- No horizontal scrolling at 1024px+
- Tablet-compatible (768px+ with adjusted layout)
- Use Tailwind responsive utilities

### Step 6: Performance Optimization

- Page load < 3s on 3G (NFR-D2):
  - Code splitting via Next.js dynamic imports
  - Lazy load position list
  - Optimize bundle size
- WebSocket reconnection with exponential backoff
- Stale-while-revalidate for initial data fetch

## Testing

### Unit Tests

- **PortfolioOverview component:** Renders all metrics, handles updates, color coding
- **PositionList component:** Renders positions, handles empty state, sorting
- **WebSocket client:** Connection, reconnection, message parsing
- **API Gateway endpoints:** Authentication, data aggregation, error handling

### Integration Tests

- **Dashboard → API Gateway → Portfolio Manager:** End-to-end data flow
- **Dashboard → WebSocket → NATS:** Real-time update propagation
- **Authentication flow:** JWT validation on API and WebSocket

### E2E Tests

- **Full dashboard load:** Page renders within 3s, all metrics visible
- **Real-time updates:** Position PnL updates within 2s of price change
- **Responsive layout:** No horizontal scroll at 1024px+

### Test Files

```
tests/unit/dashboard/
├── portfolio_overview_test.tsx
├── position_list_test.tsx
├── websocket_client_test.ts
└── api_gateway_test.py

tests/integration/
├── dashboard_portfolio_test.py
├── dashboard_websocket_test.py
└── dashboard_auth_test.py

tests/e2e/
├── dashboard_load_test.ts
└── dashboard_responsive_test.ts
```

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| `next` | 16.2.10 | Dashboard framework |
| `react` | 19.x | UI library |
| `tailwindcss` | latest | Styling |
| `@tanstack/react-table` | latest | Data table |
| `recharts` | latest | Charts (if needed) |
| `fastapi` | 0.139.0 | API Gateway |
| `websockets` | latest | FastAPI WebSocket support |
| `python-jose` | latest | JWT authentication |

## External Dependencies (Infrastructure)

| Service | Required | Purpose |
|---------|----------|---------|
| API Gateway | Yes | REST and WebSocket endpoints |
| Portfolio Manager | Yes | Portfolio metrics calculation |
| Position Manager | Yes | Position data |
| NATS | Yes | Real-time event propagation |
| Redis | Yes | Session state (optional) |

## Definition of Done

- [ ] Portfolio overview displays: total capital, daily PnL, total PnL, utilization rate
- [ ] Position list displays all active positions with real-time PnL
- [ ] Updates pushed via WebSocket within 2 seconds (NFR-D1)
- [ ] Data accurate within 1% of backend values
- [ ] Responsive layout — usable on 1024px+ screens, no horizontal scroll
- [ ] Page loads within 3 seconds on 3G connection (NFR-D2)
- [ ] JWT authentication enforced on all endpoints (AD-14)
- [ ] WebSocket reconnection with exponential backoff
- [ ] Unit tests pass with ≥80% coverage
- [ ] Integration tests pass
- [ ] E2E tests pass
- [ ] Accessibility: WCAG 2.1 AA compliance (NFR-D3)

## References

| Reference | Description |
|-----------|-------------|
| FR-48 | Dashboard SHALL display portfolio overview: total capital, daily PnL, total PnL, utilization rate |
| FR-49 | Dashboard SHALL display all active positions with real-time PnL |
| FR-54 | Dashboard SHALL be responsive (desktop-first, tablet-compatible) |
| AD-14 | JWT auth on Dashboard/Admin Panel |
| NFR-D1 | Real-time update latency within 2s |
| NFR-D2 | Page load performance <3s on 3G |
| NFR-D3 | Accessibility WCAG 2.1 AA |
| INF-4 | Next.js 16.2.10 (LTS) for Dashboard |
| INF-3 | FastAPI 0.139.0 for API Gateway |
| INF-11 | Decimal precision: all monetary values use Decimal |
| INF-12 | All timestamps UTC as TIMESTAMPTZ |
