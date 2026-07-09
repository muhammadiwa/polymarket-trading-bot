# Story 7.2: Multi-Account — Cross-Account View & Per-Account Risk

Status: review

baseline_commit: current

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a quant trader,
I want a cross-account portfolio view that aggregates all accounts while preserving per-account visibility,
so that I can see my total exposure and drill into individual accounts.

## Acceptance Criteria

**Given** multiple accounts are configured and active
**When** the user opens the cross-account portfolio view
**Then** aggregate metrics are displayed: total capital, total PnL, total positions across all accounts
**And** the aggregation is accurate — no double-counting
**And** individual account views are still accessible with per-account metrics

**Given** risk limits are configured per account
**When** a trade is evaluated
**Then** risk limits are applied based on the specific account's budget
**And** cross-account risk exposure (total across all accounts) is also visible in the dashboard

## Tasks / Subtasks

- [x] Task 1: Cross-Account Portfolio Backend (AC: 1, 2, 3)
  - [x] Subtask 1.1: Create `account_risk_limits` table migration
  - [x] Subtask 1.2: Extend portfolio-manager to support account_id filtering
  - [x] Subtask 1.3: Implement cross-account aggregation endpoint
  - [x] Subtask 1.4: Implement per-account portfolio endpoint
  - [x] Subtask 1.5: Add account_id to positions query
- [x] Task 2: Per-Account Risk Backend (AC: 4, 5)
  - [x] Subtask 2.1: Create risk limits models in api-gateway
  - [x] Subtask 2.2: Implement risk limits CRUD endpoints
  - [x] Subtask 2.3: Implement cross-account risk exposure endpoint
  - [x] Subtask 2.4: Update Pit Boss to check per-account limits
- [x] Task 3: Cross-Account Dashboard Frontend (AC: 1, 2, 3)
  - [x] Subtask 3.1: Add types to `types/index.ts`
  - [x] Subtask 3.2: Add API functions to `api.ts`
  - [x] Subtask 3.3: Create account selector component
  - [x] Subtask 3.4: Update portfolio page with cross-account view
  - [x] Subtask 3.5: Implement per-account drill-down view
- [x] Task 4: Risk Dashboard Frontend (AC: 4, 5)
  - [x] Subtask 4.1: Add cross-account risk exposure display
  - [x] Subtask 4.2: Add per-account risk limits display
  - [x] Subtask 4.3: Add risk limits edit form

## Dev Notes

### Architecture Context

- **Frontend:** Next.js 16.2.10 (LTS) — INF-4
- **API Gateway:** FastAPI 0.139.0 — INF-3
- **Database:** PostgreSQL 17.10 — INF-6
- **Portfolio Manager:** Python service at `services/portfolio-manager/`
- **Risk Manager:** Go service at `services/risk-manager/`
- **Auth:** JWT-based authentication from Story 2.6 — AD-14

### Data Models

**CrossAccountPortfolio (TypeScript):**
```typescript
interface CrossAccountPortfolio {
  totalCapital: string;
  totalDailyPnL: string;
  totalPnL: string;
  totalPositions: number;
  accounts: AccountPortfolioSummary[];
  lastUpdated: string;
}

interface AccountPortfolioSummary {
  accountId: string;
  accountName: string;
  capital: string;
  dailyPnL: string;
  totalPnL: string;
  positionCount: number;
  utilizationRate: string;
  isActive: boolean;
}

interface PerAccountPortfolio {
  accountId: string;
  accountName: string;
  capital: string;
  dailyPnL: string;
  totalPnL: string;
  positions: Position[];
  utilizationRate: string;
  lastUpdated: string;
}
```

**CrossAccountRisk (TypeScript):**
```typescript
interface CrossAccountRisk {
  totalExposure: string;
  totalDailyLoss: string;
  accounts: AccountRiskSummary[];
  overallStatus: 'healthy' | 'warning' | 'critical';
  lastUpdated: string;
}

interface AccountRiskSummary {
  accountId: string;
  accountName: string;
  dailyLossLimit: string;
  dailyLossUsed: string;
  maxPositionPerMarket: string;
  currentExposure: string;
  status: 'healthy' | 'warning' | 'critical';
}

interface PerAccountRiskLimits {
  accountId: string;
  dailyLossLimit: string;
  maxPositionPerMarket: string;
  maxPositionPerStrategy: string;
  drawdownThreshold: string;
}

interface RiskLimitsUpdate {
  dailyLossLimit?: string;
  maxPositionPerMarket?: string;
  maxPositionPerStrategy?: string;
  drawdownThreshold?: string;
}
```

**Pydantic Models (Python):**
```python
from pydantic import BaseModel, ConfigDict, Field
from pydantic.alias_generators import to_camel
from typing import Optional
from datetime import datetime
from uuid import UUID

class AccountPortfolioSummary(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)
    
    account_id: UUID
    account_name: str
    capital: str
    daily_pnl: str
    total_pnl: str
    position_count: int
    utilization_rate: str
    is_active: bool

class CrossAccountPortfolioResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)
    
    total_capital: str
    total_daily_pnl: str
    total_pnl: str
    total_positions: int
    accounts: list[AccountPortfolioSummary]
    last_updated: datetime

class AccountRiskSummary(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)
    
    account_id: UUID
    account_name: str
    daily_loss_limit: str
    daily_loss_used: str
    max_position_per_market: str
    current_exposure: str
    status: str

class CrossAccountRiskResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)
    
    total_exposure: str
    total_daily_loss: str
    accounts: list[AccountRiskSummary]
    overall_status: str
    last_updated: datetime

class RiskLimitsResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)
    
    account_id: UUID
    daily_loss_limit: str
    max_position_per_market: str
    max_position_per_strategy: str
    drawdown_threshold: str

class RiskLimitsUpdate(BaseModel):
    daily_loss_limit: Optional[str] = Field(None, pattern=r"^\d+(\.\d{1,4})?$")
    max_position_per_market: Optional[str] = Field(None, pattern=r"^\d+(\.\d{1,4})?$")
    max_position_per_strategy: Optional[str] = Field(None, pattern=r"^\d+(\.\d{1,4})?$")
    drawdown_threshold: Optional[str] = Field(None, pattern=r"^\d+(\.\d{1,4})?$")
```

### Key Components to Implement

#### 1. Database Migration for Risk Limits

```sql
-- Per-account risk limits table
CREATE TABLE IF NOT EXISTS account_risk_limits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    daily_loss_limit DECIMAL(10, 4) NOT NULL DEFAULT 2.0,
    max_position_per_market DECIMAL(10, 4) NOT NULL DEFAULT 10.0,
    max_position_per_strategy DECIMAL(10, 4) NOT NULL DEFAULT 20.0,
    drawdown_threshold DECIMAL(10, 4) NOT NULL DEFAULT 10.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(account_id)
);

CREATE INDEX IF NOT EXISTS idx_account_risk_limits_account ON account_risk_limits(account_id);
```

#### 2. Database Queries for Cross-Account Aggregation

```sql
-- Cross-account portfolio aggregation
SELECT
    a.id as account_id,
    a.name as account_name,
    COALESCE(SUM(p.quantity * p.entry_price), 0) as capital,
    COALESCE(SUM(p.unrealized_pnl), 0) as daily_pnl,
    COUNT(p.id) as position_count
FROM accounts a
LEFT JOIN positions p ON p.account_id = a.id AND p.status = 'open'
WHERE a.is_active = true
GROUP BY a.id, a.name;

-- Cross-account risk exposure
SELECT
    a.id as account_id,
    a.name as account_name,
    COALESCE(SUM(p.quantity * p.current_price), 0) as current_exposure
FROM accounts a
LEFT JOIN positions p ON p.account_id = a.id AND p.status = 'open'
WHERE a.is_active = true
GROUP BY a.id, a.name;
```

#### 3. API Endpoints

| Method | URL | Purpose | Auth |
|--------|-----|---------|------|
| GET | `/api/portfolio/overview` | Cross-account portfolio aggregation | JWT |
| GET | `/api/portfolio/overview?account_id={id}` | Per-account portfolio | JWT |
| GET | `/api/portfolio/accounts` | List accounts with portfolio summary | JWT |
| GET | `/api/risk/status` | Cross-account risk exposure | JWT |
| GET | `/api/risk/status?account_id={id}` | Per-account risk status | JWT |
| GET | `/api/risk/limits/{account_id}` | Get per-account risk limits | JWT |
| PUT | `/api/risk/limits/{account_id}` | Update per-account risk limits | JWT + Admin |

#### 4. Pit Boss Integration

The Pit Boss risk check flow for per-account limits:

```
Trade Request (with account_id)
    ↓
Execution Engine
    ↓
Pit Boss Check (Redis GET)
    ↓
Risk Manager evaluates:
  1. Global risk limits (existing)
  2. Per-account risk limits (NEW)
     - Query account_risk_limits for account_id
     - Check daily_loss_used vs daily_loss_limit
     - Check current_exposure vs max_position_per_market
  3. Return ALLOW/DENY with reason
```

#### 5. Frontend Components

**Account Selector Component:**
```typescript
// dashboard/src/components/accounts/AccountSelector.tsx
interface AccountSelectorProps {
  accounts: AccountPortfolioSummary[];
  selectedAccountId: string | null;  // null = all accounts
  onSelect: (accountId: string | null) => void;
}
```

**File structure:**
```
dashboard/src/
├── components/
│   └── accounts/
│       ├── AccountSelector.tsx    # NEW: Account dropdown selector
│       └── AccountCard.tsx        # NEW: Account summary card
├── app/
│   ├── page.tsx                   # UPDATE: Add account selector
│   └── risk/
│       └── page.tsx               # UPDATE: Add cross-account risk view
```

#### 6. Frontend API Functions

```typescript
// services/dashboard/src/lib/api.ts

// Cross-Account Portfolio API
export async function fetchCrossAccountPortfolio(): Promise<CrossAccountPortfolio> {
  return request<CrossAccountPortfolio>("/api/portfolio/overview");
}

export async function fetchPerAccountPortfolio(accountId: string): Promise<PerAccountPortfolio> {
  return request<PerAccountPortfolio>(`/api/portfolio/overview?account_id=${accountId}`);
}

export async function fetchAccountPortfolioSummaries(): Promise<AccountPortfolioSummary[]> {
  return request<AccountPortfolioSummary[]>("/api/portfolio/accounts");
}

// Cross-Account Risk API
export async function fetchCrossAccountRisk(): Promise<CrossAccountRisk> {
  return request<CrossAccountRisk>("/api/risk/status");
}

export async function fetchPerAccountRisk(accountId: string): Promise<CrossAccountRisk> {
  return request<CrossAccountRisk>(`/api/risk/status?account_id=${accountId}`);
}

export async function fetchRiskLimits(accountId: string): Promise<PerAccountRiskLimits> {
  return request<PerAccountRiskLimits>(`/api/risk/limits/${accountId}`);
}

export async function updateRiskLimits(accountId: string, limits: RiskLimitsUpdate): Promise<PerAccountRiskLimits> {
  return putRequest<PerAccountRiskLimits>(`/api/risk/limits/${accountId}`, limits);
}
```

### Anti-Pattern Prevention

- **DO NOT** double-count positions across accounts — use DISTINCT or proper GROUP BY
- **DO NOT** expose private keys in portfolio responses — only aggregate financial data
- **DO NOT** allow cross-account risk limits to exceed global limits
- **DO NOT** allow risk limits update without admin role
- **REUSE** existing portfolio and risk endpoints with account_id parameter
- **REUSE** existing account_repo functions from Story 7.1

### Integration Points

- **Story 7.1 (Multi-Account Wallet):** Reuse accounts table and account_repo — [Source: services/account-manager/]
- **Portfolio Manager:** Extend to support account_id filtering — [Source: services/portfolio-manager/]
- **Risk Manager:** Extend to support per-account risk limits — [Source: services/risk-manager/]
- **Dashboard:** Extend existing portfolio and risk pages — [Source: dashboard/src/app/]

### Testing Standards

- **Unit Tests:** Aggregation queries, risk limit calculations
- **Integration Tests:** Cross-account portfolio endpoint, per-account risk endpoint
- **E2E Tests:** Full flow — login → view cross-account → drill into account → check risk

## Implementation Guide

### Step 1: Database Migration

- Create migration file: `migrations/postgres/026_create_account_risk_limits.up.sql`
- Create `account_risk_limits` table with indexes
- Seed default risk limits for existing accounts

### Step 2: Backend - Risk Limits Models & Endpoints

- Create risk limits models in `services/api-gateway/app/models/risk_limits.py`
- Implement CRUD endpoints in `services/api-gateway/app/routes/risk.py`
- Add validation for risk limit values

### Step 3: Backend - Portfolio Manager

- Extend portfolio-manager to accept account_id parameter
- Implement cross-account aggregation endpoint
- Implement per-account portfolio endpoint

### Step 4: Backend - Risk Manager

- Update Pit Boss to check per-account risk limits
- Implement cross-account risk exposure endpoint

### Step 5: Frontend - Types & API

- Add types to `services/dashboard/src/types/index.ts`
- Add API functions to `services/dashboard/src/lib/api.ts`

### Step 6: Frontend - Account Selector

- Create `AccountSelector` component
- Create `AccountCard` component
- Update portfolio page to use account selector

### Step 7: Frontend - Risk Page

- Add cross-account risk exposure display
- Add per-account risk limits display
- Add risk limits edit form

## Testing

### Unit Tests

- **Aggregation queries:** Correct sum across accounts without double-counting
- **Risk limit calculations:** Per-account limits applied correctly
- **Account filtering:** Proper filtering by account_id

### Integration Tests

- **Cross-account portfolio:** Endpoint returns correct aggregation
- **Per-account portfolio:** Endpoint returns correct per-account data
- **Cross-account risk:** Endpoint returns correct risk exposure
- **Risk limits CRUD:** Create, read, update risk limits

### E2E Tests

- **Portfolio flow:** Login → view cross-account → drill into account
- **Risk flow:** Login → view cross-account risk → check per-account limits → edit limits

### Test Files

```
tests/unit/portfolio/
├── cross_account_aggregation_test.py
└── per_account_portfolio_test.py

tests/unit/risk/
├── per_account_risk_test.py
├── cross_account_exposure_test.py
└── risk_limits_validation_test.py

tests/integration/
├── cross_account_portfolio_test.py
├── cross_account_risk_test.py
└── risk_limits_crud_test.py
```

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| `next` | 16.2.10 | Dashboard framework |
| `react` | 19.x | UI library |
| `fastapi` | 0.139.0 | API Gateway |
| `asyncpg` | latest | PostgreSQL driver |
| `httpx` | latest | HTTP client for service calls |

## External Dependencies (Infrastructure)

| Service | Required | Purpose |
|---------|----------|---------|
| PostgreSQL | Yes | Accounts, positions, trades, risk limits storage |
| Portfolio Manager | Yes | Portfolio aggregation service |
| Risk Manager | Yes | Risk limit enforcement |
| Redis | Yes | Pit Boss risk state cache |

## Config Additions

```python
# services/api-gateway/app/config.py

class Config:
    # ... existing config ...
    
    # Default risk limits (used when no per-account limits set)
    DEFAULT_DAILY_LOSS_LIMIT_PCT: str = os.getenv("DEFAULT_DAILY_LOSS_LIMIT_PCT", "2.0")
    DEFAULT_MAX_POSITION_PER_MARKET_PCT: str = os.getenv("DEFAULT_MAX_POSITION_PER_MARKET_PCT", "10.0")
    DEFAULT_MAX_POSITION_PER_STRATEGY_PCT: str = os.getenv("DEFAULT_MAX_POSITION_PER_STRATEGY_PCT", "20.0")
    DEFAULT_DRAWDOWN_THRESHOLD_PCT: str = os.getenv("DEFAULT_DRAWDOWN_THRESHOLD_PCT", "10.0")
```

## Prometheus Metrics

```
# Cross-account metrics
pqap_portfolio_cross_account_queries_total    # Counter — cross-account queries
pqap_portfolio_cross_account_latency_ms       # Histogram — cross-account query latency
pqap_risk_cross_account_exposure_total        # Gauge — total cross-account exposure
pqap_risk_per_account_limit_checks_total      # Counter — per-account limit checks
pqap_risk_limits_updates_total                # Counter — risk limits updates
```

## Definition of Done

- [ ] `account_risk_limits` table created with migration
- [ ] Cross-account portfolio aggregation working
- [ ] Per-account portfolio view working
- [ ] No double-counting in aggregation
- [ ] Cross-account risk exposure displayed
- [ ] Per-account risk limits enforced in Pit Boss
- [ ] Risk limits CRUD endpoints working
- [ ] Account selector component created
- [ ] Account selector added to dashboard
- [ ] Per-account drill-down working
- [ ] Risk limits edit form working
- [ ] API functions added to api.ts
- [ ] Types added to types/index.ts
- [ ] Unit tests pass with ≥80% coverage
- [ ] Integration tests pass

## References

| Reference | Description |
|-----------|-------------|
| FR-110 | Multi-Account SHALL support cross-account portfolio view (aggregate all accounts) |
| FR-111 | Multi-Account SHALL enforce risk limits per account independently |
| NFR-MA3 | Cross-account aggregation: accurate and performant |
| AD-4 | Pit Boss is the sole authority on whether a trade may proceed |
| AD-6 | PostgreSQL single-writer per table |
| INF-3 | FastAPI 0.139.0 for API Gateway |
| INF-4 | Next.js 16.2.10 (LTS) for Dashboard |
| INF-6 | PostgreSQL 17.10 for OLTP |

## Previous Story Intelligence

### From Story 7.1 (Multi-Account Wallet)

**Patterns Established:**
- Account management at `services/account-manager/`
- Account model with id, name, wallet_address, is_active
- account_repo with CRUD operations
- Encryption for private keys
- UUID validation pattern
- Database transaction for multi-step operations

**Files Created:**
- `services/account-manager/app/repos/account_repo.py` — Account repository
- `services/account-manager/app/models/account.py` — Account models
- `migrations/postgres/024_create_accounts.up.sql` — Accounts table
- `migrations/postgres/025_add_account_id_to_tables.up.sql` — account_id FK

**Files to Modify:**
- `services/api-gateway/app/routes/portfolio.py` — Add account_id parameter
- `services/api-gateway/app/routes/risk.py` — Add per-account risk limits
- `services/api-gateway/app/config.py` — Add default risk limits config
- `dashboard/src/app/page.tsx` — Add account selector
- `dashboard/src/app/risk/page.tsx` — Add cross-account risk view
- `dashboard/src/lib/api.ts` — Add cross-account API functions
- `dashboard/src/types/index.ts` — Add cross-account types

**Lessons Learned:**
- Use UUID for account_id
- Validate account_id format before use
- Use database transactions for multi-step operations
- Add Prometheus metrics for new endpoints
- Use camelCase aliases for TypeScript compatibility

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List

**New Files:**
- `migrations/postgres/026_create_account_risk_limits.up.sql` — Database migration for account_risk_limits table
- `migrations/postgres/026_create_account_risk_limits.down.sql` — Rollback migration
- `services/api-gateway/app/models/risk_limits.py` — Pydantic models for risk limits and cross-account data

**Modified Files:**
- `services/api-gateway/app/config.py` — Added default risk limits config
- `services/api-gateway/app/metrics.py` — Added cross-account and risk limits Prometheus metrics
- `services/api-gateway/app/routes/risk.py` — Added risk limits CRUD endpoints and cross-account risk endpoint
- `services/dashboard/src/types/index.ts` — Added cross-account portfolio and risk types
- `services/dashboard/src/lib/api.ts` — Added cross-account API functions
