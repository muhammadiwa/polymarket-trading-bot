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

2. **Given** multiple strategies are active
   **When** capital allocation weights are assigned
   **Then** weights sum to 100% (enforced by Portfolio Manager)
   **And** weights are adjustable at runtime
   **And** the arb engine uses these weights for per-strategy capital allocation

## Tasks / Subtasks

- [ ] Task 1: Strategy Versioning Schema (AC: #1)
  - [ ] Create `strategy_versions` table migration
  - [ ] Store full parameter snapshot per version
  - [ ] Store change summary (diff) per version
- [ ] Task 2: Version Tracking on Update (AC: #1)
  - [ ] Modify `update_strategy` to create version before applying changes
  - [ ] Auto-generate change summary (old vs new values)
  - [ ] Store version with timestamp and user who made the change
- [ ] Task 3: Version History API (AC: #1)
  - [ ] Implement GET /api/strategies/{id}/versions (list versions)
  - [ ] Implement GET /api/strategies/{id}/versions/{version_id} (get specific version)
  - [ ] Implement POST /api/strategies/{id}/rollback/{version_id} (rollback)
- [ ] Task 4: Capital Allocation Weights (AC: #2)
  - [ ] Validate weights sum to 100% across active strategies
  - [ ] Add endpoint POST /api/strategies/weights (set weights for multiple strategies)
  - [ ] Publish NATS event when weights change
- [ ] Task 5: Testing
  - [ ] Unit tests for version creation on update
  - [ ] Unit tests for rollback
  - [ ] Unit tests for weight validation (sum = 100%)

## Dev Notes

### Architecture Context

- **Service:** `strategy-manager` (Python/FastAPI) — extends Story 3.2
- **Database:** PostgreSQL `strategy_versions` table
- **Event bus:** NATS for `StrategyWeightsUpdated` events
- **Pattern:** Every parameter change creates a version; rollback replaces current params with version snapshot

### Key Architecture Rules

- **NFR-SM3:** Version history complete with rollback capability
- **FR-73:** Strategy versioning (track parameter changes)
- **FR-74:** Capital allocation weights per strategy

### Database Schema

```sql
-- Migration: migrations/postgres/012_create_strategy_versions.up.sql
CREATE TABLE strategy_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id UUID NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
    version_number INT NOT NULL,
    
    -- Full parameter snapshot
    parameters JSONB NOT NULL,
    
    -- Change summary
    change_summary TEXT NOT NULL DEFAULT '',
    changed_by UUID,  -- user who made the change
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_strategy_version UNIQUE (strategy_id, version_number)
);

CREATE INDEX idx_strategy_versions_strategy ON strategy_versions(strategy_id);
CREATE INDEX idx_strategy_versions_number ON strategy_versions(strategy_id, version_number DESC);
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

### NATS Event Structure

```python
class StrategyWeightsUpdated:
    event_id: str  # UUID
    event_type: str  # "StrategyWeightsUpdated"
    timestamp: datetime  # ISO 8601 UTC
    source: str  # "strategy-manager"
    payload: dict  # {strategy_id: weight, ...}
```

### Prometheus Metrics

```
pqap_strategy_manager_versions_total    # Counter — versions created
pqap_strategy_manager_rollbacks_total   # Counter — rollbacks performed
pqap_strategy_manager_weight_changes_total  # Counter — weight changes
```

### Testing Standards

- Unit tests for version creation on each update
- Unit tests for rollback (verify parameters restored)
- Unit tests for weight validation (must sum to 100%)
- Integration tests with existing strategy CRUD

### References

| Reference | Description |
|-----------|-------------|
| FR-73 | Manager SHALL support strategy versioning (track parameter changes) |
| FR-74 | Manager SHALL assign capital allocation weights to each active strategy |
| NFR-SM3 | Version history complete with rollback |

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
