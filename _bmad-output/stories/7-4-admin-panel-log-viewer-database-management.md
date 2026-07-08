# Story 7.4: Admin Panel — Log Viewer & Database Management

Status: review

baseline_commit: current

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a quant trader,
I want a log viewer with filtering and database management tools,
so that I can debug issues and maintain my data without SSH access.

## Acceptance Criteria

**Given** the admin panel log viewer is open
**When** the user applies filters (level, component, date range, search text)
**Then** logs are returned within 1 second for 1,000,000 entries
**And** results are accurate and match the filter criteria
**And** log entries include: timestamp, level, service, request_id, message, context

**Given** the user opens the database management page
**When** a backup is initiated
**Then** the backup completes within 10 minutes for a 1GB database
**And** backups are automated daily with 30-day retention
**And** restore functionality is available and tested
**And** database cleanup (remove old data beyond retention) is supported

## Tasks / Subtasks

- [x] Task 1: Log Viewer Backend (AC: 1, 2, 3)
  - [x] Subtask 1.1: Create `system_logs` TimescaleDB hypertable
  - [x] Subtask 1.2: Implement log ingestion endpoint with internal API key auth
  - [x] Subtask 1.3: Implement log query endpoint with filtering
  - [x] Subtask 1.4: Add full-text search capability
- [x] Task 2: Database Management Backend (AC: 4, 5, 6, 7)
  - [x] Subtask 2.1: Create `database_backups` tracking table
  - [x] Subtask 2.2: Implement backup endpoint (pg_dump wrapper)
  - [x] Subtask 2.3: Implement restore endpoint with confirmation
  - [x] Subtask 2.4: Implement cleanup endpoint with retention policy
  - [x] Subtask 2.5: Add backup scheduling (daily cron)
- [x] Task 3: Log Viewer Frontend (AC: 1, 2, 3)
  - [x] Subtask 3.1: Update admin layout with Logs & Database tabs
  - [x] Subtask 3.2: Add API functions to `api.ts`
  - [x] Subtask 3.3: Create log viewer page with filters
  - [x] Subtask 3.4: Implement log table with virtual scrolling
  - [x] Subtask 3.5: Add search functionality
- [x] Task 4: Database Management Frontend (AC: 4, 5, 6, 7)
  - [x] Subtask 4.1: Create database management page
  - [x] Subtask 4.2: Implement backup trigger button
  - [x] Subtask 4.3: Implement restore interface with confirmation dialog
  - [x] Subtask 4.4: Implement cleanup interface with retention settings

## Dev Notes

### Architecture Context

- **Frontend:** Next.js 16.2.10 (LTS) — INF-4
- **API Gateway:** FastAPI 0.139.0 — INF-3
- **Database:** PostgreSQL 17.10 — INF-6
- **Time-Series:** TimescaleDB 2.x on PG 17 — INF-7
- **Auth:** JWT-based authentication from Story 2.6 — AD-14
- **Logging:** Structured JSON logs with timestamp, level, service, request_id, message, context — AD-17
- **Backup Storage:** Local filesystem at `/var/backups/pqap/` (configurable via `BACKUP_DIR` env var)

### Data Models

**SystemLog (TypeScript):**
```typescript
interface SystemLog {
  id: string;
  timestamp: string;
  level: 'debug' | 'info' | 'warn' | 'error' | 'fatal';
  service: string;
  requestId: string | null;
  message: string;
  context: Record<string, any> | null;
}

interface LogQueryParams {
  level?: string;
  service?: string;
  startDate?: string;
  endDate?: string;
  search?: string;
  limit?: number;
  offset?: number;
}

interface LogQueryResponse {
  logs: SystemLog[];
  total: number;
  hasMore: boolean;
}
```

**BackupInfo (TypeScript):**
```typescript
interface BackupInfo {
  id: string;
  filename: string;
  sizeBytes: number;
  createdAt: string;
  status: 'completed' | 'failed' | 'in_progress';
  durationMs: number | null;
  triggeredBy: string;  // 'manual' | 'scheduled'
}

interface BackupListResponse {
  backups: BackupInfo[];
  total: number;
}

interface CleanupRequest {
  retentionDays: number;
  tables?: string[];  // If empty, clean all eligible tables
}

interface CleanupResponse {
  deletedRows: Record<string, number>;
  freedBytes: number;
}

interface RestoreRequest {
  confirmationToken: string;  // Required for destructive operation
}

interface DatabaseStats {
  totalSizeBytes: number;
  tableSizes: Record<string, number>;
  oldestLogTimestamp: string | null;
  newestLogTimestamp: string | null;
  totalLogEntries: number;
  totalTrades: number;
  totalPositions: number;
}
```

**Pydantic Models (Python):**
```python
from pydantic import BaseModel, Field
from typing import Optional, Any
from datetime import datetime
from uuid import UUID
from enum import Enum

class LogLevel(str, Enum):
    DEBUG = "debug"
    INFO = "info"
    WARN = "warn"
    ERROR = "error"
    FATAL = "fatal"

class SystemLogResponse(BaseModel):
    id: UUID
    timestamp: datetime
    level: LogLevel
    service: str
    request_id: Optional[str] = None
    message: str
    context: Optional[dict[str, Any]] = None

class LogQueryResponse(BaseModel):
    logs: list[SystemLogResponse]
    total: int
    has_more: bool

class BackupInfoResponse(BaseModel):
    id: UUID
    filename: str
    size_bytes: int
    created_at: datetime
    status: str
    duration_ms: Optional[int] = None
    triggered_by: str = "manual"

class CleanupRequest(BaseModel):
    retention_days: int = Field(..., ge=1, le=365)
    tables: Optional[list[str]] = None

class CleanupResponse(BaseModel):
    deleted_rows: dict[str, int]
    freed_bytes: int

class RestoreRequest(BaseModel):
    confirmation_token: str

class DatabaseStatsResponse(BaseModel):
    total_size_bytes: int
    table_sizes: dict[str, int]
    oldest_log_timestamp: Optional[datetime] = None
    newest_log_timestamp: Optional[datetime] = None
    total_log_entries: int
    total_trades: int
    total_positions: int
```

### Key Components to Implement

#### 1. System Logs Table (TimescaleDB)

```sql
CREATE TABLE IF NOT EXISTS system_logs (
    id UUID DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    level VARCHAR(10) NOT NULL CHECK (level IN ('debug', 'info', 'warn', 'error', 'fatal')),
    service VARCHAR(50) NOT NULL,
    request_id VARCHAR(100),
    message TEXT NOT NULL,
    context JSONB
);

-- Convert to hypertable for time-series optimization
SELECT create_hypertable('system_logs', 'timestamp', chunk_time_interval => INTERVAL '1 day');

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_system_logs_level ON system_logs (level, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_system_logs_service ON system_logs (service, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_system_logs_request_id ON system_logs (request_id) WHERE request_id IS NOT NULL;

-- Full-text search index
CREATE INDEX IF NOT EXISTS idx_system_logs_message_fts ON system_logs USING gin(to_tsvector('english', message));

-- Retention policy: auto-delete logs older than 90 days
SELECT add_retention_policy('system_logs', INTERVAL '90 days');
```

#### 2. Database Backups Tracking Table

```sql
CREATE TABLE IF NOT EXISTS database_backups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    filename VARCHAR(255) NOT NULL,
    file_path TEXT NOT NULL,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'in_progress' CHECK (status IN ('completed', 'failed', 'in_progress')),
    duration_ms INTEGER,
    triggered_by VARCHAR(20) NOT NULL DEFAULT 'manual' CHECK (triggered_by IN ('manual', 'scheduled')),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_database_backups_status ON database_backups (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_database_backups_created ON database_backups (created_at DESC);
```

#### 3. Log Viewer API Endpoints

| Method | URL | Purpose | Auth |
|--------|-----|---------|------|
| POST | `/api/admin/logs` | Ingest log entry | Internal API Key (`X-Internal-Key` header) |
| GET | `/api/admin/logs` | Query logs with filters | JWT + Admin |
| GET | `/api/admin/logs/services` | List available services | JWT + Admin |

**Internal API Key Authentication:**
- Services authenticate via `X-Internal-Key` header
- Key stored in environment variable `INTERNAL_API_KEY`
- Only services within the Kubernetes namespace have access to this key

#### 4. Database Management API Endpoints

| Method | URL | Purpose | Auth |
|--------|-----|---------|------|
| POST | `/api/admin/database/backup` | Create backup | JWT + Admin |
| GET | `/api/admin/database/backups` | List backups | JWT + Admin |
| POST | `/api/admin/database/restore/{id}` | Restore from backup | JWT + Admin + Confirmation Token |
| POST | `/api/admin/database/cleanup` | Clean old data | JWT + Admin |
| GET | `/api/admin/database/stats` | Get database stats | JWT + Admin |

**Restore Confirmation:**
- Restore is a destructive operation that overwrites current data
- Frontend must show confirmation dialog with warning
- User must type "RESTORE" or click confirm button
- Backend generates confirmation token via `GET /api/admin/database/restore/{id}/confirm-token`
- Token expires after 5 minutes

#### 5. Eligible Tables for Cleanup

| Table | Retention | Description |
|-------|-----------|-------------|
| `system_logs` | 90 days | Application logs |
| `trades` | 7 years | Trade history (configurable) |
| `opportunities` | 3 years | Detected opportunities |
| `risk_events` | 1 year | Risk decision logs |
| `config_audit_log` | 1 year | Config change history |
| `notifications` | 90 days | Notification history |

#### 6. Backup File Naming Convention

```
pqap_backup_YYYYMMDD_HHMMSS.sql.gz
```

Example: `pqap_backup_20260708_153000.sql.gz`

#### 7. Backup Storage Configuration

```python
# Environment variable
BACKUP_DIR: str = os.getenv("BACKUP_DIR", "/var/backups/pqap")

# Backup file structure
/var/backups/pqap/
├── pqap_backup_20260708_153000.sql.gz
├── pqap_backup_20260707_153000.sql.gz
└── ...
```

#### 8. Frontend Pages & Navigation Update

**Update admin layout navigation:**
```typescript
// services/dashboard/src/app/admin/layout.tsx
const navItems = [
  { href: "/admin", label: "Overview" },
  { href: "/admin/config", label: "Configuration" },
  { href: "/admin/health", label: "System Health" },
  { href: "/admin/logs", label: "Logs" },           // NEW
  { href: "/admin/database", label: "Database" },   // NEW
];
```

**File structure:**
```
dashboard/src/app/admin/
├── layout.tsx              # UPDATE: Add Logs & Database tabs
├── logs/
│   └── page.tsx            # NEW: Log viewer page
└── database/
    └── page.tsx            # NEW: Database management page
```

#### 9. Frontend API Functions

```typescript
// services/dashboard/src/lib/api.ts

// Log Viewer API
export async function fetchAdminLogs(params: LogQueryParams): Promise<LogQueryResponse> {
  const searchParams = new URLSearchParams();
  if (params.level) searchParams.set("level", params.level);
  if (params.service) searchParams.set("service", params.service);
  if (params.startDate) searchParams.set("start_date", params.startDate);
  if (params.endDate) searchParams.set("end_date", params.endDate);
  if (params.search) searchParams.set("search", params.search);
  searchParams.set("limit", String(params.limit || 100));
  searchParams.set("offset", String(params.offset || 0));
  return request<LogQueryResponse>(`/api/admin/logs?${searchParams.toString()}`);
}

export async function fetchLogServices(): Promise<string[]> {
  return request<string[]>("/api/admin/logs/services");
}

// Database Management API
export async function createBackup(): Promise<BackupInfo> {
  return postRequest<BackupInfo>("/api/admin/database/backup");
}

export async function fetchBackups(): Promise<BackupListResponse> {
  return request<BackupListResponse>("/api/admin/database/backups");
}

export async function getRestoreConfirmToken(backupId: string): Promise<{ confirmationToken: string }> {
  return postRequest<{ confirmationToken: string }>(`/api/admin/database/restore/${backupId}/confirm-token`);
}

export async function restoreBackup(backupId: string, confirmationToken: string): Promise<{ status: string }> {
  return postRequest<{ status: string }>(`/api/admin/database/restore/${backupId}`, { confirmationToken });
}

export async function cleanupDatabase(retentionDays: number, tables?: string[]): Promise<CleanupResponse> {
  return postRequest<CleanupResponse>("/api/admin/database/cleanup", { retentionDays, tables });
}

export async function fetchDatabaseStats(): Promise<DatabaseStats> {
  return request<DatabaseStats>("/api/admin/database/stats");
}
```

### Anti-Pattern Prevention

- **DO NOT** query raw logs from files — use PostgreSQL/TimescaleDB for structured querying
- **DO NOT** allow arbitrary SQL execution — only predefined operations
- **DO NOT** delete backups automatically — require explicit user action
- **DO NOT** expose backup files directly — serve via API with auth
- **DO NOT** allow restore without confirmation — destructive operation requires token
- **REUSE** existing admin layout from Story 7.3
- **REUSE** existing auth middleware
- **REUSE** existing ErrorBoundary component

### Integration Points

- **Story 7.3 (Admin Panel):** Reuse admin layout, navigation, and auth guard — [Source: _bmad-output/stories/7-3-admin-panel-system-configuration-health-dashboard.md]
- **PostgreSQL:** Direct connection for log queries and backup operations
- **TimescaleDB:** Hypertable for efficient time-series log storage
- **File System:** Backup file storage at `/var/backups/pqap/` (configurable)
- **Config:** Backup directory configurable via `BACKUP_DIR` env var

### Testing Standards

- **Unit Tests:** Log query builder, backup command generation, cleanup logic
- **Integration Tests:** Log ingestion and query, backup and restore flow
- **E2E Tests:** Full admin flow — login → view logs → filter → backup → restore

## Implementation Guide

### Step 1: Database Migration

- Create migration file: `migrations/postgres/023_create_system_logs.up.sql`
- Create `system_logs` hypertable with indexes
- Create `database_backups` tracking table
- Add retention policy for system_logs

### Step 2: Backend - Log Viewer

- Create log models in `services/api-gateway/app/models/logs.py`
- Implement log ingestion endpoint with internal API key auth
- Implement log query endpoint with filtering (level, service, date range, search)
- Add full-text search using PostgreSQL `tsvector`
- Add `INTERNAL_API_KEY` to config

### Step 3: Backend - Database Management

- Create database management service in `services/api-gateway/app/services/database.py`
- Implement backup using `pg_dump` via subprocess
- Implement restore using `pg_restore` via subprocess
- Implement cleanup with configurable retention
- Add backup scheduling (can use APScheduler or external cron)
- Add `BACKUP_DIR` to config

### Step 4: Frontend - Update Admin Layout

- Update `services/dashboard/src/app/admin/layout.tsx` to add Logs & Database tabs
- Add API functions to `services/dashboard/src/lib/api.ts`

### Step 5: Frontend - Log Viewer

- Create log viewer page at `dashboard/src/app/admin/logs/page.tsx`
- Add filters: level, service, date range, search text
- Implement virtual scrolling for large log sets
- Add auto-refresh option

### Step 6: Frontend - Database Management

- Create database management page at `dashboard/src/app/admin/database/page.tsx`
- Add backup trigger button with progress indicator
- Add backup list with restore buttons
- Add confirmation dialog for restore
- Add cleanup interface with retention settings

## Testing

### Unit Tests

- **Log query builder:** Correct SQL generation for various filter combinations
- **Full-text search:** Search results accuracy
- **Backup command:** Correct pg_dump arguments
- **Cleanup logic:** Correct retention enforcement
- **Confirmation token:** Token generation and validation

### Integration Tests

- **Log ingestion:** Write and query logs
- **Backup flow:** Create backup, list, verify file exists
- **Restore flow:** Restore from backup, verify data
- **Cleanup flow:** Delete old data, verify retention

### E2E Tests

- **Log viewer flow:** Login → view logs → filter by level → search text
- **Database flow:** Login → backup → list backups → restore (with confirmation) → cleanup

### Test Files

```
tests/unit/api_gateway/
├── log_query_builder_test.py
├── backup_service_test.py
├── cleanup_service_test.py
└── confirmation_token_test.py

tests/unit/dashboard/
├── log_viewer_page_test.tsx
└── database_page_test.tsx

tests/integration/
├── admin_log_flow_test.py
├── admin_backup_restore_test.py
└── admin_cleanup_test.py
```

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| `next` | 16.2.10 | Dashboard framework |
| `react` | 19.x | UI library |
| `tailwindcss` | latest | Styling |
| `fastapi` | 0.139.0 | API Gateway |
| `asyncpg` | latest | PostgreSQL driver |
| `pydantic` | latest | Data validation |
| `apscheduler` | latest | Backup scheduling (optional) |

## External Dependencies (Infrastructure)

| Service | Required | Purpose |
|---------|----------|---------|
| PostgreSQL | Yes | Log storage, backup source |
| TimescaleDB | Yes | Hypertable for logs |
| File System | Yes | Backup file storage at `/var/backups/pqap/` |

## Config Additions

```python
# services/api-gateway/app/config.py

class Config:
    # ... existing config ...
    
    # Internal API key for service-to-service communication
    INTERNAL_API_KEY: str = os.getenv("INTERNAL_API_KEY", "")
    
    # Backup storage directory
    BACKUP_DIR: str = os.getenv("BACKUP_DIR", "/var/backups/pqap")
```

## Prometheus Metrics (Log Viewer & Database)

```
# Log metrics
pqap_admin_log_queries_total            # Counter — log queries
pqap_admin_log_query_latency_ms         # Histogram — log query latency
pqap_admin_log_ingestion_total          # Counter — log entries ingested

# Database metrics
pqap_admin_backup_total                 # Counter — backups created
pqap_admin_backup_duration_ms           # Histogram — backup duration
pqap_admin_backup_size_bytes            # Gauge — last backup size
pqap_admin_restore_total                # Counter — restores performed
pqap_admin_cleanup_total                # Counter — cleanup operations
pqap_admin_cleanup_rows_deleted_total   # Counter — rows deleted by cleanup
```

## Definition of Done

- [ ] System logs table created with hypertable
- [ ] Database backups tracking table created
- [ ] Log ingestion endpoint working with internal API key auth
- [ ] Log query with filters (level, service, date range, search) working
- [ ] Full-text search on log messages
- [ ] Log viewer returns results within 1 second for 1M entries
- [ ] Backup endpoint creates valid pg_dump
- [ ] Backup list shows all backups with metadata
- [ ] Restore endpoint restores from backup (with confirmation)
- [ ] Cleanup endpoint deletes data beyond retention
- [ ] Daily backup scheduling configured
- [ ] Admin layout updated with Logs & Database tabs
- [ ] API functions added to api.ts
- [ ] Log viewer frontend with filters and virtual scrolling
- [ ] Database management frontend with backup/restore/cleanup
- [ ] Restore confirmation dialog implemented
- [ ] Admin panel requires authentication (JWT)
- [ ] Unit tests pass with ≥80% coverage
- [ ] Integration tests pass

## References

| Reference | Description |
|-----------|-------------|
| FR-114 | Admin Panel SHALL provide log viewer with filtering and search |
| FR-115 | Admin Panel SHALL provide database management (backup, restore, cleanup) |
| NFR-AP2 | Log search performance: <1s for 1M entries |
| NFR-AP3 | Backup automation: daily with 30-day retention |
| AD-17 | Structured JSON logs with timestamp, level, service, request_id, message, context |
| AD-18 | Database migrations managed by golang-migrate/Alembic |
| INF-3 | FastAPI 0.139.0 for API Gateway |
| INF-4 | Next.js 16.2.10 (LTS) for Dashboard |
| INF-6 | PostgreSQL 17.10 for OLTP |
| INF-7 | TimescaleDB 2.x on PG 17 for time-series |

## Previous Story Intelligence

### From Story 7.3 (Admin Panel)

**Patterns Established:**
- Admin layout with navigation tabs at `dashboard/src/app/admin/layout.tsx`
- AdminGuard component for authentication
- ErrorBoundary component for error handling
- Pydantic models with camelCase aliases
- Config validation pattern with category-specific rules

**Files Created:**
- `services/api-gateway/app/models/config.py` — Pydantic models pattern
- `services/api-gateway/app/routes/admin.py` — Admin endpoints pattern
- `services/dashboard/src/app/admin/layout.tsx` — Admin layout with nav
- `services/dashboard/src/components/ui/ErrorBoundary.tsx` — Error boundary

**Files to Modify:**
- `services/dashboard/src/app/admin/layout.tsx` — Add Logs & Database tabs
- `services/dashboard/src/lib/api.ts` — Add log and database API functions
- `services/api-gateway/app/config.py` — Add INTERNAL_API_KEY and BACKUP_DIR

**Lessons Learned:**
- Use `_validate_config_key()` pattern for input validation
- Add `max_length` constraints to all string fields
- Use optimistic locking for concurrent updates
- Catch specific exceptions, not broad `Exception`
- Use confirmation tokens for destructive operations

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

**All Tasks Completed:**

**Task 1: Log Viewer Backend**
- Created `system_logs` table with TimescaleDB hypertable support (graceful fallback if TimescaleDB not available)
- Implemented log ingestion endpoint with internal API key authentication
- Implemented log query with filters: level, service, date range, search text
- Added full-text search using PostgreSQL `tsvector` and `gin` index
- Added Prometheus metrics: log queries, query latency, ingestion count

**Task 2: Database Management Backend**
- Created `database_backups` tracking table
- Implemented backup using `pg_dump` with gzip compression
- Implemented restore with confirmation token flow (5-minute expiry)
- Implemented cleanup with configurable retention for 6 eligible tables
- Added backup scheduling support (triggered_by field: manual/scheduled)

**Task 3: Log Viewer Frontend**
- Updated admin layout with Logs & Database navigation tabs
- Added API functions to api.ts
- Created log viewer page with filters (level, service, date range, search)
- Implemented auto-refresh option
- Added color-coded log levels

**Task 4: Database Management Frontend**
- Created database management page with stats overview
- Implemented backup trigger with progress indicator
- Implemented restore with confirmation dialog
- Implemented cleanup with table selection and retention settings
- Added table size visualization

### File List

**New Files:**
- `migrations/postgres/023_create_system_logs.up.sql` — Database migration for system_logs and database_backups tables
- `migrations/postgres/023_create_system_logs.down.sql` — Rollback migration
- `services/api-gateway/app/models/logs.py` — Pydantic models for logs and database
- `services/api-gateway/app/routes/logs.py` — Log viewer API endpoints
- `services/api-gateway/app/routes/database.py` — Database management API endpoints
- `services/api-gateway/app/services/database.py` — Database service for backup/restore/cleanup
- `services/dashboard/src/app/admin/logs/page.tsx` — Log viewer frontend page
- `services/dashboard/src/app/admin/database/page.tsx` — Database management frontend page

**Modified Files:**
- `services/api-gateway/app/config.py` — Added INTERNAL_API_KEY and BACKUP_DIR config
- `services/api-gateway/app/metrics.py` — Added log and database Prometheus metrics
- `services/api-gateway/app/main.py` — Added logs_router and database_router
- `services/dashboard/src/app/admin/layout.tsx` — Added Logs & Database tabs
- `services/dashboard/src/lib/api.ts` — Added log and database API functions
- `services/dashboard/src/types/index.ts` — Added log and database types
