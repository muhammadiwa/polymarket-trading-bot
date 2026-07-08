# Story 7.3: Admin Panel — System Configuration & Health Dashboard

Status: review

baseline_commit: current

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a quant trader,
I want an admin panel to manage system configuration and monitor system health,
so that I can adjust settings and troubleshoot issues without editing config files.

## Acceptance Criteria

**Given** the admin panel is accessible (requires authentication)
**When** the user opens the system configuration page
**Then** the following can be configured via the UI: API keys, risk defaults, notification settings
**And** all changes are validated before save
**And** changes are persisted and logged with timestamp and previous value

**Given** the user opens the system health page
**When** health metrics render
**Then** the following are displayed: CPU, memory, disk, network, connection status
**And** metrics are accurate and update every 5 seconds
**And** alerts are displayed when thresholds are breached

## Tasks / Subtasks

- [x] Task 1: System Configuration API (AC: 1, 2, 3)
  - [x] Subtask 1.1: Create `system_config` table in PostgreSQL
  - [x] Subtask 1.2: Implement config CRUD endpoints in api-gateway
  - [x] Subtask 1.3: Add config validation logic
  - [x] Subtask 1.4: Implement config change audit logging
- [x] Task 2: System Health API (AC: 4, 5, 6)
  - [x] Subtask 2.1: Create health metrics aggregation service
  - [x] Subtask 2.2: Implement health endpoint in api-gateway
  - [x] Subtask 2.3: Add threshold-based alerting logic
- [x] Task 3: Admin Panel Frontend (AC: 1-6)
  - [x] Subtask 3.1: Create admin layout with auth guard
  - [x] Subtask 3.2: Build system configuration page
  - [x] Subtask 3.3: Build system health dashboard page
  - [x] Subtask 3.4: Add real-time health metrics updates via WebSocket

## Dev Notes

### Architecture Context

- **Frontend:** Next.js 16.2.10 (LTS) — INF-4
- **API Gateway:** FastAPI 0.139.0 — INF-3
- **Database:** PostgreSQL 17.10 — INF-6
- **Cache:** Redis 8.8.0 — INF-5
- **Metrics:** Prometheus 3.12.0 — INF-9
- **Auth:** JWT-based authentication from Story 2.6 — AD-14
- **Pattern:** Admin panel is part of dashboard at `dashboard/src/app/admin/` — Structural Seed

### Data Models

**SystemConfig (TypeScript):**
```typescript
interface SystemConfig {
  id: string;
  configKey: string;
  configValue: any;  // JSONB
  category: 'api_keys' | 'risk_defaults' | 'notification_settings';
  description: string | null;
  isSensitive: boolean;
  createdAt: string;
  updatedAt: string;
  updatedBy: string | null;
}

interface SystemConfigUpdate {
  configValue: any;
  reason?: string;
}
```

**ConfigAuditLog (TypeScript):**
```typescript
interface ConfigAuditLog {
  id: string;
  configKey: string;
  oldValue: any;
  newValue: any;
  changedBy: string;
  changedAt: string;
  reason: string | null;
}
```

**HealthStatus (TypeScript):**
```typescript
interface HealthStatus {
  services: ServiceHealth[];
  overall: 'healthy' | 'degraded' | 'unhealthy';
  alerts: HealthAlert[];
  lastUpdated: string;
}

interface ServiceHealth {
  name: string;
  status: 'up' | 'down' | 'degraded';
  cpuPercent: number;
  memoryMB: number;
  memoryLimitMB: number;
  diskPercentFree: number;
  networkBytesIn: number;
  networkBytesOut: number;
  connected: boolean;  // WebSocket status for scanner
  lastHeartbeat: string;
}

interface HealthAlert {
  id: string;
  service: string;
  metric: string;
  threshold: number;
  currentValue: number;
  severity: 'warning' | 'critical';
  triggeredAt: string;
  message: string;
}
```

**WebSocket Messages:**
```typescript
interface HealthUpdateMessage {
  type: 'health_update';
  payload: HealthStatus;
  timestamp: string;
}

interface ConfigChangeMessage {
  type: 'config_changed';
  payload: {
    configKey: string;
    newValue: any;
    changedBy: string;
  };
  timestamp: string;
}
```

**Pydantic Models (Python):**
```python
from pydantic import BaseModel, Field
from typing import Any, Optional
from datetime import datetime
from uuid import UUID

class SystemConfigBase(BaseModel):
    config_key: str = Field(..., max_length=100)
    config_value: Any
    category: str = Field(..., pattern=r'^(api_keys|risk_defaults|notification_settings)$')
    description: Optional[str] = None
    is_sensitive: bool = False

class SystemConfigCreate(SystemConfigBase):
    pass

class SystemConfigUpdate(BaseModel):
    config_value: Any
    reason: Optional[str] = None

class SystemConfigResponse(SystemConfigBase):
    id: UUID
    created_at: datetime
    updated_at: datetime
    updated_by: Optional[UUID] = None

    class Config:
        from_attributes = True

class ConfigAuditLogResponse(BaseModel):
    id: UUID
    config_key: str
    old_value: Optional[Any] = None
    new_value: Any
    changed_by: UUID
    changed_at: datetime
    reason: Optional[str] = None
```

### Key Components to Implement

#### 1. System Configuration Table (`system_config`)

```sql
CREATE TABLE system_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_key VARCHAR(100) UNIQUE NOT NULL,
    config_value JSONB NOT NULL,
    category VARCHAR(50) NOT NULL CHECK (category IN ('api_keys', 'risk_defaults', 'notification_settings')),
    description TEXT,
    is_sensitive BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(id)
);

CREATE INDEX idx_system_config_category ON system_config(category);
CREATE INDEX idx_system_config_key ON system_config(config_key);

CREATE TABLE config_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_key VARCHAR(100) NOT NULL,
    old_value JSONB,
    new_value JSONB NOT NULL,
    changed_by UUID REFERENCES users(id),
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason TEXT
);

CREATE INDEX idx_config_audit_log_key ON config_audit_log(config_key);
CREATE INDEX idx_config_audit_log_changed_at ON config_audit_log(changed_at DESC);
```

#### 2. Configuration Validation Rules

| Category | Key Examples | Validation Rules |
|----------|--------------|------------------|
| `api_keys` | `polymarket_api_key`, `telegram_bot_token`, `polymarket_secret` | Non-empty string, min length 10, max length 500 |
| `risk_defaults` | `daily_loss_limit_pct`, `max_position_per_market_pct`, `max_position_per_strategy_pct`, `drawdown_circuit_breaker_pct`, `win_streak_threshold` | Positive number, 0-100 for percentages |
| `notification_settings` | `throttle_rate_per_min`, `critical_bypass_throttle`, `enable_telegram`, `enable_email` | Integer >= 0 for rate, boolean for flags |

**Sensitive Value Masking:**
- GET responses mask sensitive values: `polymarket_api_key` returns `"sk-...a3b2"` (show last 4 chars)
- Full value only returned when explicitly requested with `?unmask=true` (requires admin role)
- Masking function: `mask_sensitive(value) => value[:3] + "..." + value[-4:]`

#### 3. Configuration API Endpoints

| Method | URL | Purpose | Auth |
|--------|-----|---------|------|
| GET | `/api/admin/config` | List all config (non-sensitive values masked) | JWT |
| GET | `/api/admin/config/{key}` | Get specific config value | JWT |
| GET | `/api/admin/config/{key}?unmask=true` | Get full value (sensitive only) | JWT + Admin |
| PUT | `/api/admin/config/{key}` | Update config value | JWT + Admin + CSRF |
| GET | `/api/admin/config/audit` | Get config change history | JWT |

**Concurrent Update Strategy:**
- Use optimistic locking with `updated_at` timestamp
- PUT request includes `expected_updated_at` in body
- If `updated_at` doesn't match, return 409 Conflict
- Client must re-fetch and retry

#### 4. System Health Aggregation

Health metrics collected from Prometheus endpoints of all services:

```python
# Health metric categories
HEALTH_CATEGORIES = {
    "cpu": {
        "metric": "process_cpu_seconds_total",
        "threshold_warning": 0.7,  # 70%
        "threshold_critical": 0.9  # 90%
    },
    "memory": {
        "metric": "process_resident_memory_bytes",
        "threshold_warning": 0.8,  # 80% of limit
        "threshold_critical": 0.95  # 95% of limit
    },
    "disk": {
        "metric": "node_filesystem_avail_bytes",
        "threshold_warning": 0.2,  # 20% free
        "threshold_critical": 0.1  # 10% free
    },
    "network": {
        "metric": "node_network_receive_bytes_total",
        "threshold_warning": None,  # informational
        "threshold_critical": None
    },
    "connections": {
        "metric": "pqap_scanner_ws_connection_status",
        "threshold_warning": 0,  # 0 = disconnected
        "threshold_critical": 0
    }
}
```

#### 5. Health API Endpoints

| Method | URL | Purpose | Auth |
|--------|-----|---------|------|
| GET | `/api/admin/health` | Get aggregated health status | JWT |
| GET | `/api/admin/health/services` | Get per-service health | JWT |
| GET | `/api/admin/health/alerts` | Get active alerts | JWT |
| WS | `/ws/admin/health` | Real-time health updates | JWT |

#### 6. Frontend Pages

```
dashboard/src/app/admin/
├── layout.tsx              # Admin layout with auth guard
├── page.tsx                # Admin dashboard (redirects to config or health)
├── config/
│   ├── page.tsx            # System configuration page
│   └── ConfigForm.tsx      # Configuration form component
└── health/
    ├── page.tsx            # System health dashboard
    ├── ServiceStatus.tsx   # Per-service status card
    └── AlertList.tsx       # Active alerts list
```

### Anti-Pattern Prevention

- **DO NOT** store sensitive config (API keys) in plain text — use `is_sensitive` flag and mask values in GET responses
- **DO NOT** poll health metrics — use WebSocket push from api-gateway
- **DO NOT** hardcode thresholds — make them configurable in `system_config`
- **DO NOT** skip audit logging — every config change must be logged
- **DO NOT** allow concurrent updates without optimistic locking — use `updated_at` check
- **REUSE** existing JWT auth middleware from Story 2.6
- **REUSE** existing Prometheus client libraries already in use by other services

### Integration Points

- **Story 2.6 (Auth):** Reuse JWT auth middleware and admin role check — [Source: _bmad-output/stories/2-6-admin-authentication-scanner-metrics-export.md]
- **Story 2.4 (System Health):** Health dashboard extends existing system health display — [Source: _bmad-output/stories/2-4-dashboard-system-health-opportunity-feed.md]
- **Prometheus:** Query Prometheus API for current metrics — [Source: architecture/ARCHITECTURE-SPINE.md#AD-17]
- **Redis:** Cache health metrics for fast retrieval (TTL 5s) — [Source: architecture/ARCHITECTURE-SPINE.md#AD-8]
- **NATS:** Subscribe to `pqap.system.health` events for real-time updates — [Source: architecture/ARCHITECTURE-SPINE.md#AD-9]

## Implementation Guide

### Step 1: Database Migration

- Create migration file: `migrations/postgres/007_create_system_config.up.sql`
- Create `system_config` table with indexes
- Create `config_audit_log` table with indexes
- Seed default config values for each category:
  ```sql
  INSERT INTO system_config (config_key, config_value, category, description, is_sensitive) VALUES
  ('daily_loss_limit_pct', '2.0', 'risk_defaults', 'Daily loss limit as percentage of capital', false),
  ('max_position_per_market_pct', '10.0', 'risk_defaults', 'Max position per market as percentage of capital', false),
  ('max_position_per_strategy_pct', '20.0', 'risk_defaults', 'Max position per strategy as percentage of capital', false),
  ('drawdown_circuit_breaker_pct', '10.0', 'risk_defaults', 'Drawdown circuit breaker threshold', false),
  ('win_streak_threshold', '5', 'risk_defaults', 'Batasi Win streak threshold', false),
  ('throttle_rate_per_min', '10', 'notification_settings', 'Max non-critical notifications per minute', false),
  ('critical_bypass_throttle', 'true', 'notification_settings', 'Critical notifications bypass throttle', false),
  ('enable_telegram', 'true', 'notification_settings', 'Enable Telegram notifications', false),
  ('polymarket_api_key', '""', 'api_keys', 'Polymarket API key', true),
  ('telegram_bot_token', '""', 'api_keys', 'Telegram bot token', true);
  ```

### Step 2: Backend - Config API

- Create Pydantic models in `services/api-gateway/app/models/config.py`
- Implement CRUD endpoints in `services/api-gateway/app/routes/admin.py`
- Add validation logic per config category
- Implement sensitive value masking function
- Implement audit logging on every update
- Add optimistic locking check on PUT endpoint

### Step 3: Backend - Health Aggregation

- Create health service in `services/api-gateway/app/services/health.py`
- Query Prometheus API for current metrics from all services
- Implement threshold-based alerting logic
- Cache results in Redis (TTL 5s)
- Implement WebSocket endpoint for real-time health push

### Step 4: Frontend - Admin Layout

- Create admin layout at `dashboard/src/app/admin/layout.tsx`
- Implement auth guard (reuse from Story 2.6)
- Add navigation sidebar with links to Config and Health pages
- Add breadcrumbs for admin section

### Step 5: Frontend - Config Page

- Build config form at `dashboard/src/app/admin/config/page.tsx`
- Create category tabs (API Keys, Risk Defaults, Notification Settings)
- Implement form validation before save
- Show masked values for sensitive fields
- Add "Show Value" toggle for sensitive fields (requires confirmation)
- Display audit log in collapsible sidebar or modal
- Handle 409 Conflict with user-friendly retry prompt

### Step 6: Frontend - Health Dashboard

- Build service status cards at `dashboard/src/app/admin/health/page.tsx`
- Create `ServiceStatus.tsx` component with color-coded status indicators
- Create `AlertList.tsx` component for active alerts
- Implement real-time updates via WebSocket connection
- Add auto-refresh fallback if WebSocket disconnects

## Testing

### Unit Tests

- **Config validation:** Valid values, invalid values, boundary cases per category
- **Sensitive masking:** Mask function returns correct format
- **Health aggregation:** Correct aggregation logic, threshold detection
- **Optimistic locking:** Concurrent update detection

### Integration Tests

- **Config CRUD:** Create, read, update config with database
- **Audit logging:** Verify audit log entries on config changes
- **Health endpoint:** Query Prometheus mock and return aggregated health
- **WebSocket health:** Real-time health updates propagate correctly

### E2E Tests

- **Admin flow:** Login → view config → update config → verify audit log
- **Health monitoring:** Login → view health → verify metrics display → verify alerts
- **Concurrent updates:** Two sessions update same config, second gets 409

### Test Files

```
tests/unit/api_gateway/
├── config_validation_test.py
├── config_crud_test.py
├── config_masking_test.py
├── health_aggregation_test.py
└── threshold_alerting_test.py

tests/unit/dashboard/
├── admin_config_page_test.tsx
├── admin_health_page_test.tsx
├── config_form_validation_test.tsx
└── service_status_card_test.tsx

tests/integration/
├── admin_config_flow_test.py
├── admin_config_audit_test.py
├── admin_health_realtime_test.py
└── admin_concurrent_update_test.py

tests/e2e/
└── admin_panel_full_flow_test.ts
```

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| `next` | 16.2.10 | Dashboard framework |
| `react` | 19.x | UI library |
| `tailwindcss` | latest | Styling |
| `fastapi` | 0.139.0 | API Gateway |
| `sqlalchemy` | latest | ORM for PostgreSQL |
| `pydantic` | latest | Data validation |
| `prometheus-client` | latest (Python) | Query Prometheus API |
| `python-jose[cryptography]` | latest | JWT validation (from Story 2.6) |
| `redis` | latest | Health cache |

## External Dependencies (Infrastructure)

| Service | Required | Purpose |
|---------|----------|---------|
| PostgreSQL | Yes | Config storage, audit log |
| Redis | Yes | Health cache (5s TTL) |
| Prometheus | Yes | Health metrics source |
| NATS | Yes | Health events (pqap.system.health) |

## Prometheus Metrics (Admin Panel)

```
# Config metrics
pqap_admin_config_changes_total            # Counter — config changes made
pqap_admin_config_validation_errors_total  # Counter — validation failures

# Health metrics
pqap_admin_health_checks_total             # Counter — health check requests
pqap_admin_health_check_latency_ms         # Histogram — health check latency
pqap_admin_active_alerts_total             # Gauge — active alerts count
pqap_admin_ws_connections_total            # Gauge — active WebSocket connections
```

## Definition of Done

- [ ] System config table created with migration
- [ ] Default config values seeded
- [ ] Config CRUD API endpoints working
- [ ] Config validation for all categories
- [ ] Sensitive values masked in GET responses
- [ ] Audit logging on every config change
- [ ] Optimistic locking for concurrent updates
- [ ] Health metrics aggregated from all services
- [ ] Health dashboard displays CPU, memory, disk, network, connections
- [ ] Health metrics update every 5 seconds
- [ ] Alerts displayed when thresholds breached
- [ ] Real-time health updates via WebSocket
- [ ] Admin panel requires authentication (JWT)
- [ ] CSRF protection on config update endpoints
- [ ] Prometheus metrics exported for admin operations
- [ ] Unit tests pass with ≥80% coverage
- [ ] Integration tests pass
- [ ] E2E tests pass

## References

| Reference | Description |
|-----------|-------------|
| FR-112 | Admin Panel SHALL provide system configuration interface (API keys, risk defaults, notification settings) |
| FR-113 | Admin Panel SHALL provide system health dashboard (CPU, memory, disk, network, connections) |
| FR-116 | Admin Panel SHALL require authentication (even for single user) |
| NFR-AP1 | Authentication security: session timeout; CSRF protection |
| AD-14 | Secrets in Kubernetes Secrets; JWT auth on Dashboard/Admin Panel |
| AD-17 | Prometheus metrics on /metrics for all services |
| INF-3 | FastAPI 0.139.0 for API Gateway |
| INF-4 | Next.js 16.2.10 (LTS) for Dashboard |
| INF-6 | PostgreSQL 17.10 for OLTP |
| INF-9 | Prometheus 3.12.0 for metrics |

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

**All Tasks Completed:**

**Task 3 Completed (Frontend):**

1. **Admin Layout (layout.tsx):**
   - Navigation with Overview, Configuration, Health tabs
   - AdminGuard for authentication
   - Link back to main dashboard

2. **Admin Overview (page.tsx):**
   - Card links to Configuration and Health pages
   - User Management placeholder

3. **Config Page (config/page.tsx):**
   - Category filter (API Keys, Risk Defaults, Notification Settings)
   - Inline editing with validation
   - Sensitive value masking
   - Audit log modal
   - Optimistic locking conflict handling

4. **Health Page (health/page.tsx):**
   - Service status cards (Scanner, Arb Engine, Execution Engine, Risk Manager, Position Manager)
   - CPU, Memory, Error Rate metrics
   - WebSocket connection status for Scanner
   - Active alerts display
   - Auto-refresh every 5 seconds

**Task 1 & 2 Completed (Backend):**

1. **Database Migration (022):**
   - Created `system_config` table with validation constraints
   - Created `config_audit_log` table for tracking changes
   - Added indexes for performance
   - Seeded 12 default config values across 3 categories

2. **Config API (admin.py):**
   - GET `/api/admin/config` — List all configs (sensitive values masked)
   - GET `/api/admin/config/{key}` — Get specific config (with unmask option for admin)
   - PUT `/api/admin/config/{key}` — Update config with validation and optimistic locking
   - GET `/api/admin/config/audit/logs` — Get audit log history

3. **Config Validation (models/config.py):**
   - API keys: min 10 chars, max 500 chars
   - Risk defaults: 0-100 range for percentages
   - Notification settings: boolean flags or non-negative integers
   - Sensitive value masking function

4. **Health API (health.py):**
   - GET `/api/admin/health` — Aggregated health with alerts
   - GET `/api/admin/health/services` — Per-service health
   - GET `/api/admin/health/alerts` — Active alerts
   - WS `/ws/admin/health` — Real-time WebSocket updates
   - Threshold-based alerting (CPU, memory, error rate)
   - Alert severity: warning (70%/80%/5) and critical (90%/95%/10)

5. **Prometheus Metrics:**
   - `pqap_admin_config_changes_total` — Config changes counter
   - `pqap_admin_config_validation_errors_total` — Validation errors
   - `pqap_admin_health_checks_total` — Health check requests
   - `pqap_admin_health_check_latency_ms` — Health check latency
   - `pqap_admin_active_alerts_total` — Active alerts gauge
   - `pqap_admin_ws_connections_total` — WebSocket connections

### File List

**New Files:**
- `migrations/postgres/022_create_system_config.up.sql` — Database migration for system_config and config_audit_log tables
- `migrations/postgres/022_create_system_config.down.sql` — Rollback migration
- `services/api-gateway/app/models/config.py` — Pydantic models and validation logic for system config
- `services/dashboard/src/app/admin/layout.tsx` — Admin layout with navigation
- `services/dashboard/src/app/admin/config/page.tsx` — System configuration page
- `services/dashboard/src/app/admin/health/page.tsx` — System health dashboard page

**Modified Files:**
- `services/api-gateway/app/routes/admin.py` — Added config CRUD endpoints (GET/PUT/audit)
- `services/api-gateway/app/routes/health.py` — Added admin health endpoints with alerts and WebSocket
- `services/api-gateway/app/metrics.py` — Added admin panel Prometheus metrics
- `services/api-gateway/app/config.py` — Added PROMETHEUS_URL config
- `services/api-gateway/app/main.py` — Added admin_health_router and health_ws_router
- `services/dashboard/src/types/index.ts` — Added admin types (SystemConfig, HealthAlert, etc.)
- `services/dashboard/src/lib/api.ts` — Added admin API functions
- `services/dashboard/src/app/admin/page.tsx` — Updated admin overview page
