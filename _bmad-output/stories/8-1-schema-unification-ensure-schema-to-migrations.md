# Story 8.1: Schema Unification — Move ensureSchema() to Migrations

Status: ready-for-dev

## Story

As a developer,
I want all table definitions consolidated into migrations instead of ensureSchema(),
so that all services start successfully regardless of startup order and there are no schema conflicts.

## Acceptance Criteria

- [ ] All table definitions consolidated into migrations
- [ ] `ensureSchema()` calls removed from Go services
- [ ] All services use migration-managed tables
- [ ] Existing data preserved during migration
- [ ] All services start successfully regardless of startup order

## Tasks / Subtasks

- [ ] Task 1: Audit ensureSchema() implementations
  - [ ] Subtask 1.1: Find all ensureSchema() calls in Go services
  - [ ] Subtask 1.2: Document conflicting schemas
- [ ] Task 2: Create unified migrations
  - [ ] Subtask 2.1: Create migration for risk_events table
  - [ ] Subtask 2.2: Create migration for positions table
  - [ ] Subtask 2.3: Fix opportunities table migration
  - [ ] Subtask 2.4: Fix trades table INSERT columns
- [ ] Task 3: Remove ensureSchema() from Go services
  - [ ] Subtask 3.1: Remove from risk-manager
  - [ ] Subtask 3.2: Remove from execution-engine
  - [ ] Subtask 3.3: Remove from position-manager
  - [ ] Subtask 3.4: Remove from arb-engine
- [ ] Task 4: Update Go services to use migration-managed tables
  - [ ] Subtask 4.1: Update risk-manager queries
  - [ ] Subtask 4.2: Update execution-engine queries
  - [ ] Subtask 4.3: Update position-manager queries

## Affected Tables

| Table | Conflicting Definitions | Services |
|-------|------------------------|----------|
| `risk_events` | 3 different schemas | risk-manager, execution-engine |
| `positions` | 2 different schemas | risk-manager, position-manager |
| `opportunities` | 2 different schemas | migration 007, arb-engine |
| `trades` | INSERT uses non-existent columns | execution-engine |

## Dev Notes

### Architecture Context

- **Database:** PostgreSQL 17.10 — INF-6
- **Migrations:** golang-migrate for Go services — AD-18
- **Pattern:** Single-writer per table — AD-6

### Schema Conflicts Detail

#### risk_events Table

**Definition A (risk-manager/schema.sql):**
```sql
CREATE TABLE IF NOT EXISTS risk_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision TEXT NOT NULL,
    reason TEXT NOT NULL,
    market_id TEXT DEFAULT NULL,
    strategy_id TEXT DEFAULT NULL,
    trade_size NUMERIC(18,8) NOT NULL DEFAULT 0,
    current_exposure NUMERIC(18,8) NOT NULL DEFAULT 0,
    limit_value NUMERIC(18,8) NOT NULL DEFAULT 0,
    daily_budget_remaining NUMERIC(18,8) NOT NULL,
    capital NUMERIC(18,8) NOT NULL,
    context JSONB DEFAULT '{}',
    account_id UUID DEFAULT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Definition B (execution-engine/adapters/postgres_repo.go):**
```sql
CREATE TABLE IF NOT EXISTS risk_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    market_id TEXT NOT NULL,
    strategy_id TEXT NOT NULL,
    order_size NUMERIC(10,8) NOT NULL,
    allowed BOOLEAN NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    latency_ms INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Unified Schema (Migration 027):**
```sql
CREATE TABLE IF NOT EXISTS risk_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision TEXT NOT NULL,
    reason TEXT NOT NULL,
    market_id TEXT DEFAULT NULL,
    strategy_id TEXT DEFAULT NULL,
    trade_size NUMERIC(18,8) NOT NULL DEFAULT 0,
    order_size NUMERIC(18,8) NOT NULL DEFAULT 0,
    current_exposure NUMERIC(18,8) NOT NULL DEFAULT 0,
    limit_value NUMERIC(18,8) NOT NULL DEFAULT 0,
    daily_budget_remaining NUMERIC(18,8) NOT NULL DEFAULT 0,
    capital NUMERIC(18,8) NOT NULL DEFAULT 0,
    allowed BOOLEAN NOT NULL DEFAULT true,
    latency_ms INTEGER NOT NULL DEFAULT 0,
    context JSONB DEFAULT '{}',
    account_id UUID DEFAULT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

#### positions Table

**Definition A (risk-manager):**
- PK: `position_id TEXT`
- Precision: `NUMERIC(18,8)`

**Definition B (position-manager):**
- PK: `id UUID DEFAULT gen_random_uuid()`
- Precision: `NUMERIC(10,4)`

**Unified Schema (Migration 028):**
```sql
CREATE TABLE IF NOT EXISTS positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    market_id TEXT NOT NULL,
    market_slug TEXT NOT NULL DEFAULT '',
    side TEXT NOT NULL,
    entry_price NUMERIC(18,8) NOT NULL,
    current_price NUMERIC(18,8) NOT NULL,
    quantity NUMERIC(18,8) NOT NULL,
    unrealized_pnl NUMERIC(18,8) NOT NULL DEFAULT 0,
    realized_pnl NUMERIC(18,8) NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'open',
    strategy_id TEXT NOT NULL DEFAULT '',
    entry_order_id UUID DEFAULT NULL,
    exit_order_id UUID DEFAULT NULL,
    opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ DEFAULT NULL,
    settled_at TIMESTAMPTZ DEFAULT NULL,
    account_id UUID DEFAULT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## Implementation Guide

### Step 1: Create Migration 027 - risk_events

- Create unified schema combining both definitions
- Add all columns from both services
- Use sensible defaults for NOT NULL columns

### Step 2: Create Migration 028 - positions

- Create unified schema with UUID primary key
- Use higher precision (18,8) for all price/quantity columns
- Add all columns from both services

### Step 3: Fix Migration 007 - opportunities

- Update to use consistent column names
- Ensure hypertable partitioning works

### Step 4: Fix Execution Engine INSERT

- Update InsertTrade() to use correct column names
- Update InsertAtomicPair() to use correct column names

### Step 5: Remove ensureSchema()

- Remove schema creation code from all Go services
- Services should rely on migrations only

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
