# Story 3.2: Strategy Manager — CRUD & Activation

Status: ready-for-dev

## Story

As a quant trader,
I want to create, read, update, and delete trading strategies and activate/deactivate them without restarting the bot,
So that I can dynamically adjust which strategies are running.

## Acceptance Criteria

1. **Given** the strategy manager API is available
   **When** the user creates a new strategy with parameters (thresholds, position sizing, risk limits)
   **Then** the strategy is persisted to the PostgreSQL `strategies` table
   **And** all parameters are validated before save — invalid values rejected with clear error messages
   **And** the `account_id` column is included (nullable) for future multi-account support

2. **Given** a strategy exists
   **When** the user activates or deactivates it
   **Then** the change takes effect within 1 second
   **And** no other strategies are affected
   **And** no service restart is required
   **And** a `StrategyUpdated` event is published to NATS

## Tasks / Subtasks

- [ ] Task 1: Database Schema (AC: #1)
  - [ ] Create `strategies` table migration
  - [ ] Include `account_id` (nullable) for future multi-account
- [ ] Task 2: Strategy CRUD API (AC: #1)
  - [ ] Create strategy-manager service structure
  - [ ] Implement POST /api/strategies (create)
  - [ ] Implement GET /api/strategies (list)
  - [ ] Implement GET /api/strategies/{id} (get by ID)
  - [ ] Implement PUT /api/strategies/{id} (update)
  - [ ] Implement DELETE /api/strategies/{id} (delete)
  - [ ] Validate all parameters before save
- [ ] Task 3: Strategy Activation (AC: #2)
  - [ ] Implement POST /api/strategies/{id}/activate
  - [ ] Implement POST /api/strategies/{id}/deactivate
  - [ ] Publish `StrategyUpdated` event to NATS
  - [ ] Changes take effect within 1 second
- [ ] Task 4: Testing
  - [ ] Unit tests for CRUD operations
  - [ ] Unit tests for activation/deactivation
  - [ ] Unit tests for parameter validation

## Dev Notes

### Architecture Context

- **Service:** New `strategy-manager` service (Python/FastAPI)
- **Database:** PostgreSQL `strategies` table (AD-6: single-writer)
- **Event bus:** NATS for `StrategyUpdated` events
- **Pattern:** REST API for CRUD, NATS for event propagation

### Key Architecture Rules

- **AD-6:** PostgreSQL single-writer: strategies table owned by Strategy Manager
- **INF-18:** Include `account_id` as nullable column for future multi-account support
- **NFR-SM1:** Config persistence in PostgreSQL
- **NFR-SM2:** Parameter validation before save

### Database Schema

```sql
-- Migration: migrations/postgres/011_create_strategies.up.sql
CREATE TABLE strategies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'inactive' CHECK (status IN ('active', 'inactive', 'paused')),
    
    -- Strategy parameters
    min_profit_threshold DECIMAL(10,6) NOT NULL DEFAULT 0.01,
    score_threshold DECIMAL(10,6) NOT NULL DEFAULT 0.01,
    max_position_size DECIMAL(20,8) NOT NULL DEFAULT 1000.0,
    max_daily_trades INT NOT NULL DEFAULT 50,
    risk_limit_pct DECIMAL(5,2) NOT NULL DEFAULT 5.0,
    
    -- Capital allocation
    capital_weight DECIMAL(5,2) NOT NULL DEFAULT 100.0,
    
    -- Multi-account support (nullable)
    account_id UUID,
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    activated_at TIMESTAMPTZ,
    
    CONSTRAINT valid_capital_weight CHECK (capital_weight >= 0 AND capital_weight <= 100),
    CONSTRAINT valid_risk_limit CHECK (risk_limit_pct > 0 AND risk_limit_pct <= 100)
);

CREATE INDEX idx_strategies_status ON strategies(status);
CREATE INDEX idx_strategies_account ON strategies(account_id);
```

### NATS Event Structure

```python
class StrategyUpdated:
    event_id: str  # UUID
    event_type: str  # "StrategyUpdated"
    timestamp: datetime  # ISO 8601 UTC
    source: str  # "strategy-manager"
    payload: StrategyUpdatedPayload

class StrategyUpdatedPayload:
    strategy_id: str
    name: str
    status: str  # "active", "inactive", "paused"
    action: str  # "created", "updated", "activated", "deactivated", "deleted"
    parameters: dict
```

### Prometheus Metrics

```
pqap_strategy_manager_strategies_total    # Gauge — total strategies
pqap_strategy_manager_active_total        # Gauge — active strategies
pqap_strategy_manager_operations_total    # Counter — CRUD operations (labels: operation, status)
```

### Testing Standards

- Unit tests for each CRUD endpoint
- Unit tests for parameter validation
- Unit tests for activation/deactivation
- Integration tests with PostgreSQL
- All tests follow existing patterns in `services/api-gateway/tests/`

### References

| Reference | Description |
|-----------|-------------|
| FR-70 | Manager SHALL support CRUD operations for strategies |
| FR-71 | Manager SHALL support strategy activation/deactivation without restart |
| FR-72 | Manager SHALL validate strategy parameters |
| AD-6 | PostgreSQL single-writer per table |
| INF-18 | Include account_id as nullable column |
| NFR-SM1 | Config persistence in PostgreSQL |
| NFR-SM2 | Parameter validation before save |

## Dev Agent Record

### Agent Model Used

mimo-v2.5-pro

### Debug Log References

- Created new strategy-manager service from scratch
- Followed existing api-gateway patterns for consistency

### Completion Notes List

- Task 1: DB migration created with strategies table
- Task 2: Full CRUD API implemented (POST, GET, PUT, DELETE)
- Task 3: Activation/deactivation with NATS event publishing
- Task 4: Pydantic validation on all inputs

### File List

**New files:**
- `migrations/postgres/011_create_strategies.up.sql`
- `migrations/postgres/011_create_strategies.down.sql`
- `services/strategy-manager/app/main.py`
- `services/strategy-manager/app/config.py`
- `services/strategy-manager/app/db.py`
- `services/strategy-manager/app/events.py`
- `services/strategy-manager/app/models/strategy.py`
- `services/strategy-manager/app/repos/strategy_repo.py`
- `services/strategy-manager/app/routes/strategies.py`
- `services/strategy-manager/app/middleware/auth.py`
- `services/strategy-manager/requirements.txt`
- `services/strategy-manager/Dockerfile`
