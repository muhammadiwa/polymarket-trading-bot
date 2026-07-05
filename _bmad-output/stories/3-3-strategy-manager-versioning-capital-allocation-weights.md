# Story 3.3: Strategy Manager — Versioning & Capital Allocation Weights

Status: ready-for-dev

## Story

As a quant trader,
I want strategy parameter changes tracked with version history and each strategy assigned capital allocation weights,
So that I can roll back bad changes and control how capital is distributed.

## Acceptance Criteria

1. **Given** a strategy's parameters are modified
   **When** the update is saved
   **Then** a new version is created in the version history
   **And** the previous version is preserved with timestamp and change summary
   **And** rollback to any previous version is supported
   **And** rollback itself creates a new version (audit trail preserved)

2. **Given** multiple strategies are active
   **When** capital allocation weights are assigned
   **Then** weights are validated by strategy-manager (must sum to 100% ±0.01%)
   **And** weights are adjustable at runtime via POST /api/strategies/weights
   **And** StrategyWeightsUpdated event is published to NATS for downstream consumers
   **And** new strategies default to 0% weight (user must explicitly assign)

## Tasks / Subtasks

- [ ] Task 1: Strategy Versioning Schema (AC: #1)
  - [ ] Create `strategy_versions` table migration (up + down)
  - [ ] Store full parameter snapshot per version (JSONB)
  - [ ] Store change summary (diff) per version
  - [ ] Version numbers are sequential per strategy (1, 2, 3...)
- [ ] Task 2: Version Tracking on Update (AC: #1)
  - [ ] Modify `update_strategy` in `strategy_repo.py` to create version before applying changes
  - [ ] Auto-generate change summary by comparing old vs new values
  - [ ] Extract `changed_by` (user_id) from JWT and store with version
  - [ ] Only create version if parameters actually changed
- [ ] Task 3: Version History API (AC: #1)
  - [ ] Implement GET /api/strategies/{id}/versions (list all versions, newest first)
  - [ ] Implement GET /api/strategies/{id}/versions/{version_id} (get specific version)
  - [ ] Implement POST /api/strategies/{id}/rollback/{version_id} (restore parameters from version)
  - [ ] Rollback creates a new version with change_summary = "Rollback to version N"
  - [ ] Rollback does NOT change strategy status (active/inactive preserved)
- [ ] Task 4: Capital Allocation Weights (AC: #2)
  - [ ] New strategies default to `capital_weight = 0` (not 100)
  - [ ] Implement POST /api/strategies/weights — accepts `{strategy_id: weight, ...}` dict
  - [ ] Validate sum == 100% with tolerance ±0.01% (Decimal comparison)
  - [ ] Reject if any strategy_id doesn't exist or is not active
  - [ ] Publish `StrategyWeightsUpdated` event to NATS on success
- [ ] Task 5: Testing
  - [ ] Unit tests for version creation on update
  - [ ] Unit tests for rollback (verify parameters restored, new version created)
  - [ ] Unit tests for weight validation (sum != 100% rejected, tolerance works)
  - [ ] Unit tests for edge cases (rollback to version 1, concurrent updates)

## Dev Notes

### Architecture Context

- **Service:** `strategy-manager` (Python/FastAPI) — extends Story 3.2
- **Database:** PostgreSQL `strategy_versions` table
- **Event bus:** NATS for `StrategyWeightsUpdated` events
- **Pattern:** Every parameter change creates a version; rollback replaces current params with version snapshot and creates a new version

### Key Architecture Rules

- **NFR-SM3:** Version history complete with rollback capability
- **FR-73:** Strategy versioning (track parameter changes)
- **FR-74:** Capital allocation weights per strategy
- **INF-11:** All monetary values use Decimal (for weight comparison)

### Files to MODIFY

**`services/strategy-manager/app/repos/strategy_repo.py`**
- Current: `update_strategy` does direct UPDATE
- Change: Before UPDATE, create version snapshot of current state
- Preserve: All existing CRUD logic

**`services/strategy-manager/app/routes/strategies.py`**
- Current: CRUD endpoints
- Change: Add version listing, version detail, rollback, weight endpoints
- Preserve: All existing endpoints

**`services/strategy-manager/app/models/strategy.py`**
- Current: StrategyCreate, StrategyUpdate, StrategyResponse
- Change: Add VersionResponse, WeightUpdateRequest models
- Preserve: All existing models

**`services/strategy-manager/app/events.py`**
- Current: StrategyEventPublisher
- Change: Add `publish_strategy_weights_updated` method
- Preserve: All existing event publishing

**`migrations/postgres/011_create_strategies.up.sql`**
- Current: `capital_weight` defaults to 100.0
- Change: Default to 0.0 (user must explicitly assign weights)

### New Files to CREATE

**`migrations/postgres/012_create_strategy_versions.up.sql`**
- `strategy_versions` table with JSONB parameters, change_summary, changed_by

**`migrations/postgres/012_create_strategy_versions.down.sql`**
- Drop `strategy_versions` table

**`services/strategy-manager/app/repos/version_repo.py`**
- Version CRUD operations
- Version number auto-increment
- Change summary generation

### Database Schema

```sql
-- Migration UP: migrations/postgres/012_create_strategy_versions.up.sql
CREATE TABLE strategy_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id UUID NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
    version_number INT NOT NULL,
    
    -- Full parameter snapshot
    parameters JSONB NOT NULL,
    
    -- Change summary
    change_summary TEXT NOT NULL DEFAULT '',
    changed_by UUID,  -- user_id from JWT
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_strategy_version UNIQUE (strategy_id, version_number)
);

CREATE INDEX idx_strategy_versions_strategy ON strategy_versions(strategy_id);
CREATE INDEX idx_strategy_versions_number ON strategy_versions(strategy_id, version_number DESC);

-- Migration DOWN: migrations/postgres/012_create_strategy_versions.down.sql
DROP TABLE IF EXISTS strategy_versions;
```

### Version Number Generation

Sequential per strategy, auto-generated:
```sql
SELECT COALESCE(MAX(version_number), 0) + 1 FROM strategy_versions WHERE strategy_id = $1
```

### Version Snapshot Structure

```json
{
    "name": "My Strategy",
    "description": "...",
    "min_profit_threshold": 0.02,
    "score_threshold": 0.015,
    "max_position_size": 500.0,
    "max_daily_trades": 30,
    "risk_limit_pct": 3.0,
    "capital_weight": 25.0,
    "status": "active"
}
```

### Change Summary Format

```
min_profit_threshold: 0.01 → 0.02
score_threshold: 0.01 → 0.015
max_daily_trades: 50 → 30
```

### Rollback Behavior

1. Read target version's parameters
2. Create new version with current state (audit trail)
3. Apply target version's parameters to strategy
4. Create another new version with "Rollback to version N" as change_summary
5. Strategy status (active/inactive) is NOT changed by rollback

### Weight Validation Logic

```python
from decimal import Decimal, ROUND_HALF_UP

def validate_weights(weights: dict[str, Decimal]) -> bool:
    total = sum(weights.values())
    # Allow ±0.01% tolerance for floating point
    return abs(total - Decimal("100")) <= Decimal("0.01")
```

- New strategies default to `capital_weight = 0`
- User must explicitly assign weights via POST /api/strategies/weights
- All active strategies' weights must sum to 100% ±0.01%

### NATS Event Structure

```python
class StrategyWeightsUpdated:
    event_id: str  # UUID
    event_type: str  # "StrategyWeightsUpdated"
    timestamp: datetime  # ISO 8601 UTC
    source: str  # "strategy-manager"
    payload: dict  # {"weights": {strategy_id: weight, ...}, "total": 100.0}
```

### Prometheus Metrics

```
pqap_strategy_manager_versions_total    # Counter — versions created
pqap_strategy_manager_rollbacks_total   # Counter — rollbacks performed
pqap_strategy_manager_weight_changes_total  # Counter — weight changes
```

### Testing Standards

- Unit tests for version creation on each update
- Unit tests for rollback (verify parameters restored, new version created)
- Unit tests for weight validation (must sum to 100% ±0.01%)
- Unit tests for edge cases (rollback to version 1, concurrent updates, empty weights)
- Integration tests with existing strategy CRUD

### References

| Reference | Description |
|-----------|-------------|
| FR-73 | Manager SHALL support strategy versioning (track parameter changes) |
| FR-74 | Manager SHALL assign capital allocation weights to each active strategy |
| NFR-SM3 | Version history complete with rollback |
| INF-11 | Decimal precision for weight calculations |

## Dev Agent Record

### Agent Model Used

mimo-v2.5-pro

### Debug Log References

- Extended existing strategy-manager service from Story 3.2
- Version repo uses JSONB for flexible parameter snapshots

### Completion Notes List

- Task 1: strategy_versions table with JSONB parameters, sequential version numbers
- Task 2: update_strategy creates version before applying changes
- Task 3: Version history API (list, get, rollback) with audit trail
- Task 4: Weight validation (sum=100% ±0.01%), NATS event publishing
- Task 5: Rollback preserves strategy status, creates before+after versions

### File List

**New files:**
- `migrations/postgres/012_create_strategy_versions.up.sql`
- `migrations/postgres/012_create_strategy_versions.down.sql`
- `services/strategy-manager/app/repos/version_repo.py`

**Modified files:**
- `migrations/postgres/011_create_strategies.up.sql` — capital_weight default changed to 0
- `services/strategy-manager/app/repos/strategy_repo.py` — version creation on update
- `services/strategy-manager/app/models/strategy.py` — added VersionResponse, WeightUpdateRequest
- `services/strategy-manager/app/events.py` — added publish_strategy_weights_updated
- `services/strategy-manager/app/routes/strategies.py` — added version + weight endpoints
