# Story 2.3: Dashboard — Risk Status & Quick Actions

## Story

As a quant trader,
I want to see real-time risk status and have quick action controls on the dashboard,
So that I can monitor risk exposure and take immediate action when needed.

## Status

not-started

## Acceptance Criteria

**Given** the dashboard is loaded
**When** the risk status section renders
**Then** it displays: daily budget remaining, current drawdown, win streak count, circuit breaker status (open/closed)
**And** risk data matches the backend risk manager state
**And** updates are pushed within 2 seconds of any change

**Given** the dashboard quick actions are available
**When** the user triggers an action (emergency stop, pause/resume trading, adjust risk parameters)
**Then** the action executes within 1 second
**And** confirmation is required for critical actions (emergency stop, resume after circuit breaker)
**And** risk parameter adjustments are persisted and logged

## Technical Requirements

### Architecture Context

- **Frontend:** Next.js 16.2.10 (LTS) — INF-4
- **API Gateway:** FastAPI 0.139.0 (Python) — INF-3
- **Backend service:** risk-manager (Go) — Pit Boss state in Redis
- **Event bus:** NATS (for real-time risk state updates)
- **Communication:** API Gateway → risk-manager (internal); Dashboard ↔ API Gateway (REST + WebSocket)
- **Pattern:** Risk state is read from Redis (Pit Boss keys, 60s TTL). Quick actions are REST calls that modify risk state and trigger backend actions.

### Key Components to Implement

1. **Risk Status Component** (`dashboard/src/components/risk/RiskStatus.tsx`)
   - Daily budget remaining (amount + percentage bar) — FR-51
   - Current drawdown (percentage, color-coded)
   - Win streak count (current / threshold)
   - Circuit breaker status (open = green, closed = red with alert)
   - Real-time updates via WebSocket (within 2s per NFR-D1)

2. **Quick Actions Component** (`dashboard/src/components/risk/QuickActions.tsx`)
   - Emergency Stop button — FR-53
   - Pause/Resume Trading toggle
   - Risk parameter adjustment form (daily limit, position limits)
   - Confirmation modal for critical actions
   - Action status feedback (success/failure)

3. **Risk Parameter Adjustment** (`dashboard/src/components/risk/RiskParamForm.tsx`)
   - Editable fields: daily loss limit (%), max position per market (%), max position per strategy (%)
   - Validation against safe ranges
   - Persist changes via API
   - Log all changes with previous/new values

4. **API Gateway Endpoints** (`services/api-gateway/`)
   - `GET /api/risk/status` — current risk state from Pit Boss
   - `POST /api/risk/emergency-stop` — trigger emergency stop
   - `POST /api/risk/pause` — pause trading
   - `POST /api/risk/resume` — resume trading
   - `PUT /api/risk/parameters` — update risk parameters
   - `WS /ws/dashboard` — real-time risk state push (existing from 2.2)

5. **Risk Manager Integration** (`services/risk-manager/`)
   - Expose risk state via internal API (reads from Redis Pit Boss)
   - Handle emergency stop (halt all trading, cancel open orders)
   - Handle pause/resume (update Pit Boss state)
   - Handle parameter updates (validate, persist, log)

### Data Models

**RiskStatus:**
```typescript
interface RiskStatus {
  dailyBudgetRemaining: string;    // Decimal string (8dp)
  dailyBudgetTotal: string;        // Decimal string (8dp)
  dailyBudgetUsedPercent: string;  // Decimal string (4dp)
  currentDrawdown: string;         // Decimal string (4dp, percentage)
  drawdownThreshold: string;       // Decimal string (4dp)
  winStreakCurrent: number;
  winStreakThreshold: number;
  circuitBreakerStatus: 'open' | 'closed';
  circuitBreakerTrippedAt: string | null;  // ISO 8601 UTC
  isPaused: boolean;
  pausedReason: string | null;
  lastUpdated: string;             // ISO 8601 UTC
}
```

**QuickActionRequest:**
```typescript
interface EmergencyStopRequest {
  reason: string;
  confirmationToken: string;       // Required for critical actions
}

interface RiskParameterUpdate {
  dailyLossLimit?: string;         // Decimal percentage
  maxPositionPerMarket?: string;   // Decimal percentage
  maxPositionPerStrategy?: string; // Decimal percentage
}
```

**WebSocket Messages:**
```typescript
interface RiskUpdateMessage {
  type: 'risk_update';
  payload: RiskStatus;
  timestamp: string;
}
```

### API Endpoints

| API | Method | URL | Purpose |
|-----|--------|-----|---------|
| Risk Status | GET | `/api/risk/status` | Current risk state |
| Emergency Stop | POST | `/api/risk/emergency-stop` | Trigger emergency stop |
| Pause Trading | POST | `/api/risk/pause` | Pause all trading |
| Resume Trading | POST | `/api/risk/resume` | Resume trading |
| Update Parameters | PUT | `/api/risk/parameters` | Update risk parameters |

### NATS Events

```
pqap.risk.state.updated           # Risk state change (pushed to dashboard)
pqap.risk.emergency_stop          # Emergency stop triggered
pqap.risk.parameter.changed       # Risk parameter modified
```

### Prometheus Metrics

```
pqap_dashboard_risk_actions_total         # Counter — quick actions executed (by type)
pqap_dashboard_risk_action_latency_ms     # Histogram — action execution latency
pqap_dashboard_risk_param_changes_total   # Counter — parameter adjustments
```

## Implementation Guide

### Step 1: Risk Status Component

- Fetch initial state from `GET /api/risk/status`
- Subscribe to WebSocket `risk_update` messages
- Display metrics:
  - Daily Budget: `$XXX.XX / $X,XXX.XX (XX.X%)` with progress bar
  - Drawdown: `X.XX%` with color (green < 5%, yellow 5-8%, red > 8%)
  - Win Streak: `3 / 5` (current / threshold)
  - Circuit Breaker: Status badge (green "Open" / red "Closed — Tripped at HH:MM")
  - Paused: Banner if trading is paused with reason
- Update on WebSocket message (within 2s)

### Step 2: Quick Actions Component

- **Emergency Stop:**
  - Red prominent button
  - Confirmation modal: "Are you sure? This will halt all trading and cancel open orders."
  - Requires typing "STOP" to confirm (confirmation token)
  - On confirm: `POST /api/risk/emergency-stop`
  - Success: "Trading halted" notification
  - Failure: Error message with retry option

- **Pause/Resume Toggle:**
  - Toggle switch with current state label
  - On pause: `POST /api/risk/pause` with reason
  - On resume: `POST /api/risk/resume`
  - Resume after circuit breaker requires confirmation

- **Parameter Adjustment:**
  - Form with current values pre-filled
  - Validation: percentages must be positive, within safe ranges
  - Save button → `PUT /api/risk/parameters`
  - Success: "Parameters updated" notification
  - All changes logged with previous and new values

### Step 3: API Gateway Endpoints

- Implement in FastAPI (`services/api-gateway/`)
- All endpoints require JWT authentication (AD-14)
- Emergency stop: validate confirmation token before executing
- Parameter updates: validate ranges, persist to PostgreSQL, update Redis Pit Boss state
- Return appropriate HTTP status codes (200, 400, 401, 403, 500)

### Step 4: Risk Manager Integration

- Expose internal API for risk state queries
- Emergency stop handler:
  - Set emergency stop flag in Redis
  - Publish `EmergencyStop` event to NATS
  - Trigger order cancellation via execution engine
  - Log to `risk_events` table
- Pause/Resume handler:
  - Update `trading_paused` flag in Redis Pit Boss state
  - Publish state change event
- Parameter update handler:
  - Validate new values
  - Persist to PostgreSQL `risk_parameters` table
  - Update Redis Pit Boss state
  - Log change with old/new values

### Step 5: Real-time Updates

- Risk manager publishes `RiskStateUpdated` events to NATS on any state change
- API Gateway subscribes to `pqap.risk.state.updated`
- Forwards updates to connected dashboard clients via WebSocket
- Updates include full risk state (not deltas) for simplicity

## Testing

### Unit Tests

- **RiskStatus component:** Renders all metrics, handles updates, color coding
- **QuickActions component:** Emergency stop flow, pause/resume, parameter form
- **API Gateway endpoints:** Authentication, validation, error handling
- **Risk Manager:** State assembly, emergency stop execution, parameter persistence

### Integration Tests

- **Dashboard → API Gateway → Risk Manager:** End-to-end risk status flow
- **Emergency stop:** Full flow from button click to trading halt
- **Parameter update:** Form submission → validation → persistence → state update
- **WebSocket updates:** Risk state changes propagate to dashboard

### E2E Tests

- **Risk monitoring:** Dashboard displays accurate risk state
- **Quick actions:** Emergency stop halts trading within 1 second
- **Parameter adjustment:** Changes persist and take effect

### Test Files

```
tests/unit/dashboard/
├── risk_status_test.tsx
├── quick_actions_test.tsx
└── risk_param_form_test.tsx

tests/unit/api_gateway/
└── risk_endpoints_test.py

tests/integration/
├── dashboard_risk_status_test.py
├── emergency_stop_test.py
└── risk_param_update_test.py

tests/e2e/
└── dashboard_risk_actions_test.ts
```

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| `next` | 16.2.10 | Dashboard framework |
| `react` | 19.x | UI library |
| `tailwindcss` | latest | Styling |
| `fastapi` | 0.139.0 | API Gateway |
| `python-jose` | latest | JWT authentication |
| `redis` | 8.8.0 | Pit Boss state |

## External Dependencies (Infrastructure)

| Service | Required | Purpose |
|---------|----------|---------|
| Risk Manager | Yes | Risk state, emergency stop, parameter management |
| API Gateway | Yes | REST endpoints, WebSocket |
| Redis | Yes | Pit Boss state (60s TTL) |
| PostgreSQL | Yes | risk_events, risk_parameters tables |
| NATS | Yes | Real-time risk state events |

## Definition of Done

- [ ] Risk status displays: daily budget, drawdown, win streak, circuit breaker status
- [ ] Risk data matches backend risk manager state within 2s (NFR-D1)
- [ ] Emergency stop executes within 1 second with confirmation
- [ ] Pause/Resume trading works correctly
- [ ] Risk parameter adjustments persisted and logged
- [ ] Confirmation required for critical actions (emergency stop, resume after breaker)
- [ ] JWT authentication enforced on all endpoints (AD-14)
- [ ] Unit tests pass with ≥80% coverage
- [ ] Integration tests pass
- [ ] E2E tests pass

## References

| Reference | Description |
|-----------|-------------|
| FR-51 | Dashboard SHALL display risk status: daily budget, drawdown, win streak, circuit breaker status |
| FR-53 | Dashboard SHALL provide quick actions: emergency stop, pause/resume, risk param adjustment |
| AD-4 | Pit Boss is sole authority; Redis keys with 60s TTL |
| AD-14 | JWT auth on Dashboard/Admin Panel |
| NFR-D1 | Real-time update latency within 2s |
| INF-4 | Next.js 16.2.10 (LTS) for Dashboard |
| INF-3 | FastAPI 0.139.0 for API Gateway |
| INF-8 | Redis 8.8.0 for cache/coordination |
