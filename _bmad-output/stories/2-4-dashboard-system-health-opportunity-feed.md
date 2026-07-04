# Story 2.4: Dashboard — System Health & Opportunity Feed

## Story

As a quant trader,
I want to see system health metrics and a live feed of detected opportunities,
So that I can verify the bot is operating correctly and see what it's finding.

## Status

not-started

## Acceptance Criteria

**Given** the dashboard is loaded
**When** the system health section renders
**Then** it displays: WebSocket connection status, CPU usage, memory usage, error rate
**And** health metrics update every 5 seconds
**And** metrics are accurate within 10% of actual values

**Given** the arb engine is detecting opportunities
**When** the opportunity feed renders
**Then** it displays a live stream of detected and executed opportunities
**And** the feed updates within 1 second of opportunity detection
**And** historical opportunities are scrollable
**And** each entry shows: market, score, spread, timestamp, status (detected/executed/filtered)

## Technical Requirements

### Architecture Context

- **Frontend:** Next.js 16.2.10 (LTS) — INF-4
- **API Gateway:** FastAPI 0.139.0 (Python) — INF-3
- **Backend services:** scanner (Go), arb-engine (Go), risk-manager (Go)
- **Event bus:** NATS (JetStream for durable subscriptions)
- **Database:** TimescaleDB for opportunity history
- **Communication:** API Gateway aggregates health from all services; opportunity feed via NATS → WebSocket

### Key Components to Implement

1. **System Health Component** (`dashboard/src/components/health/SystemHealth.tsx`)
   - WebSocket connection status (connected/disconnected with last-seen timestamp) — FR-52
   - CPU usage (percentage gauge)
   - Memory usage (percentage gauge or absolute)
   - Error rate (errors per minute)
   - Auto-refresh every 5 seconds
   - Color-coded thresholds (green/yellow/red)

2. **Opportunity Feed Component** (`dashboard/src/components/opportunities/OpportunityFeed.tsx`)
   - Live stream of opportunities — FR-50
   - Each entry: market name, score, spread, timestamp, status badge
   - Status badges: detected (blue), executed (green), filtered (gray)
   - Auto-scroll for new entries
   - Scrollable history (virtual scrolling for performance)
   - Update within 1s of detection

3. **Health Aggregation Service** (`services/api-gateway/health/`)
   - Aggregate health from all backend services
   - Expose `GET /api/system/health`
   - Poll services every 5s or subscribe to health events
   - Cache aggregated health in Redis (5s TTL)

4. **Opportunity Stream Service** (`services/api-gateway/opportunities/`)
   - Subscribe to NATS subjects: `pqap.opportunity.detected`, `pqap.opportunity.executed`, `pqap.opportunity.filtered`
   - Forward events to dashboard via WebSocket
   - Query historical opportunities from TimescaleDB

5. **API Gateway Endpoints** (`services/api-gateway/`)
   - `GET /api/system/health` — aggregated system health
   - `GET /api/opportunities` — historical opportunities (paginated)
   - `WS /ws/dashboard` — real-time opportunity and health push

### Data Models

**SystemHealth:**
```typescript
interface SystemHealth {
  scanner: ServiceHealth;
  arbEngine: ServiceHealth;
  executionEngine: ServiceHealth;
  riskManager: ServiceHealth;
  positionManager: ServiceHealth;
  overall: 'healthy' | 'degraded' | 'unhealthy';
  lastUpdated: string;
}

interface ServiceHealth {
  name: string;
  status: 'up' | 'down' | 'degraded';
  wsConnected: boolean;          // Scanner-specific
  cpuPercent: number;
  memoryMB: number;
  errorRate: number;             // Errors per minute
  lastHeartbeat: string;
}
```

**Opportunity:**
```typescript
interface Opportunity {
  id: string;
  market: string;
  marketSlug: string;
  score: string;                 // Decimal string
  spread: string;                // Decimal string
  fillProbability: string;       // Decimal string
  timestamp: string;             // ISO 8601 UTC
  status: 'detected' | 'executed' | 'filtered';
  filterReason: string | null;   // If filtered, why
  executionLatencyMs: number | null;  // If executed
}
```

**WebSocket Messages:**
```typescript
interface HealthUpdateMessage {
  type: 'health_update';
  payload: SystemHealth;
  timestamp: string;
}

interface OpportunityMessage {
  type: 'opportunity';
  payload: Opportunity;
  timestamp: string;
}
```

### API Endpoints

| API | Method | URL | Purpose |
|-----|--------|-----|---------|
| System Health | GET | `/api/system/health` | Aggregated health metrics |
| Opportunities | GET | `/api/opportunities` | Historical opportunities (paginated) |
| Dashboard WebSocket | WS | `/ws/dashboard` | Real-time push (health + opportunities) |

### NATS Subjects

```
pqap.opportunity.detected        # New opportunity detected
pqap.opportunity.executed        # Opportunity executed
pqap.opportunity.filtered        # Opportunity filtered (below threshold)
pqap.system.health.{service}     # Service health heartbeats
```

### Prometheus Metrics

```
pqap_dashboard_health_polls_total        # Counter — health polling requests
pqap_dashboard_health_stale_total        # Counter — stale health data detected
pqap_dashboard_opportunities_streamed_total  # Counter — opportunities streamed via WS
pqap_dashboard_opportunity_feed_latency_ms   # Histogram — feed update latency
```

## Implementation Guide

### Step 1: System Health Component

- Fetch initial health from `GET /api/system/health`
- Subscribe to WebSocket `health_update` messages
- Display per-service health cards:
  - Service name + status badge (green/yellow/red)
  - Scanner: WebSocket connection status (connected since HH:MM / disconnected)
  - CPU: XX.X% with color threshold (< 60% green, 60-80% yellow, > 80% red)
  - Memory: XXX MB / 1024 MB
  - Error rate: X.X errors/min
- Overall system status: healthy (all up), degraded (1-2 degraded), unhealthy (any down)
- Auto-refresh every 5s via WebSocket push

### Step 2: Opportunity Feed Component

- Fetch historical opportunities from `GET /api/opportunities` (initial load, paginated)
- Subscribe to WebSocket `opportunity` messages for live updates
- Render each opportunity as a card/row:
  - Market name (truncated if long)
  - Score: X.XXXX
  - Spread: $X.XX
  - Timestamp: HH:MM:SS
  - Status badge: Detected (blue), Executed (green), Filtered (gray)
- New entries appear at top (reverse chronological)
- Implement virtual scrolling for performance (thousands of entries)
- Filter controls: by status, by market

### Step 3: Health Aggregation

- API Gateway polls backend services every 5s for health status
- Alternatively, services publish heartbeats to NATS `pqap.system.health.{service}`
- Aggregate into `SystemHealth` response
- Cache in Redis with 5s TTL
- Include error rate calculation (errors in last minute)

### Step 4: Opportunity Stream

- API Gateway subscribes to NATS opportunity subjects
- On new opportunity event:
  - Forward to all connected dashboard clients via WebSocket
  - Log for debugging
- For historical data, query TimescaleDB `opportunities` table
- Pagination: cursor-based for efficient scrolling

### Step 5: Responsive Layout

- System health: horizontal card layout (5 cards in a row on desktop, 2-3 on tablet)
- Opportunity feed: full-width scrollable list below health cards
- Maintain desktop-first approach (1024px+)

## Testing

### Unit Tests

- **SystemHealth component:** Renders all services, color coding, status badges
- **OpportunityFeed component:** Renders opportunities, status badges, virtual scrolling
- **Health aggregation:** Correct aggregation logic, caching
- **Opportunity stream:** NATS subscription, WebSocket forwarding

### Integration Tests

- **Health polling:** API Gateway polls services and returns aggregated health
- **Opportunity feed:** End-to-end from NATS event to dashboard display
- **WebSocket updates:** Both health and opportunity updates propagate correctly

### E2E Tests

- **Health monitoring:** Dashboard displays accurate system health
- **Opportunity feed:** Live opportunities appear within 1 second
- **Historical scroll:** Past opportunities load and scroll correctly

### Test Files

```
tests/unit/dashboard/
├── system_health_test.tsx
└── opportunity_feed_test.tsx

tests/unit/api_gateway/
├── health_aggregation_test.py
└── opportunity_stream_test.py

tests/integration/
├── dashboard_health_test.py
└── dashboard_opportunity_feed_test.py

tests/e2e/
└── dashboard_health_opportunities_test.ts
```

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| `next` | 16.2.10 | Dashboard framework |
| `react` | 19.x | UI library |
| `tailwindcss` | latest | Styling |
| `@tanstack/react-virtual` | latest | Virtual scrolling for opportunity feed |
| `fastapi` | 0.139.0 | API Gateway |

## External Dependencies (Infrastructure)

| Service | Required | Purpose |
|---------|----------|---------|
| Scanner | Yes | Health metrics, WebSocket status |
| Arb Engine | Yes | Opportunity detection events |
| Risk Manager | Yes | Health metrics |
| NATS | Yes | Opportunity events, health heartbeats |
| TimescaleDB | Yes | Historical opportunity data |
| Redis | Yes | Health cache (5s TTL) |

## Definition of Done

- [ ] System health displays: WebSocket status, CPU, memory, error rate per service
- [ ] Health metrics update every 5 seconds (accurate within 10%)
- [ ] Opportunity feed shows live stream of detected/executed/filtered opportunities
- [ ] Feed updates within 1 second of opportunity detection
- [ ] Historical opportunities scrollable with pagination
- [ ] Each opportunity entry shows: market, score, spread, timestamp, status
- [ ] Overall system status correctly calculated (healthy/degraded/unhealthy)
- [ ] Virtual scrolling for performance with large opportunity lists
- [ ] Unit tests pass with ≥80% coverage
- [ ] Integration tests pass
- [ ] E2E tests pass

## References

| Reference | Description |
|-----------|-------------|
| FR-50 | Dashboard SHALL display live opportunity feed (detected and executed) |
| FR-52 | Dashboard SHALL display system health: connection status, CPU, memory, error rate |
| AD-7 | TimescaleDB for opportunities hypertable |
| AD-9 | NATS event bus with defined subject hierarchy |
| AD-17 | Prometheus metrics on /metrics for all services |
| NFR-D1 | Real-time update latency within 2s |
| INF-4 | Next.js 16.2.10 (LTS) for Dashboard |
| INF-3 | FastAPI 0.139.0 for API Gateway |
| INF-7 | TimescaleDB 2.x on PG 17 for time-series |
| INF-8 | Redis 8.8.0 for cache/coordination |
