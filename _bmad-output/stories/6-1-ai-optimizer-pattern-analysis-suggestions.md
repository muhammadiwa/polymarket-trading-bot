# Story 6.1: AI Optimizer — Pattern Analysis & Suggestions

Status: ready-for-dev

## Story

As a quant trader,
I want the AI optimizer to analyze my trade history and identify patterns in winning vs losing trades,
So that I can improve my strategy parameters based on data-driven insights.

## Acceptance Criteria

1. **Given** at least 100 trades exist in the trade history
   **When** the AI optimizer analyzes the data
   **Then** it identifies statistically significant patterns (p < 0.05) in winning vs losing trades
   **And** patterns include: time-of-day effects, score threshold effectiveness, position sizing impact
   **And** if fewer than 100 trades exist, the API returns 400 with "insufficient data" message

2. **Given** patterns are identified
   **When** the optimizer generates suggestions
   **Then** each suggestion includes: parameter name, current value, suggested value, expected impact (percentage change in win rate or PnL), and methodology (plain English explanation)
   **And** suggestions are stored in `optimizer_suggestions` table for manual review
   **And** no suggestion is auto-applied — user must explicitly approve via API
   **And** approval/rejection is logged with timestamp and user

## Tasks / Subtasks

- [ ] Task 1: AI Optimizer Service Setup
  - [ ] Create `services/ai-optimizer/` Python/FastAPI service
  - [ ] Connect to PostgreSQL (trades table)
  - [ ] JWT authentication
- [ ] Task 2: Pattern Analysis Engine
  - [ ] Analyze winning vs losing trades
  - [ ] Identify statistically significant patterns (p < 0.05)
  - [ ] Pattern types: time-of-day, score thresholds, position sizing
  - [ ] Return error if < 100 trades
- [ ] Task 3: Suggestion Generator
  - [ ] Generate parameter change suggestions
  - [ ] Quantify expected impact as percentage change
  - [ ] Include plain English methodology explanation
  - [ ] Store in optimizer_suggestions table
- [ ] Task 4: API Endpoints
  - [ ] POST /api/optimizer/analyze — run analysis for a strategy
  - [ ] GET /api/optimizer/suggestions — list suggestions (filter by status)
  - [ ] POST /api/optimizer/suggestions/{id}/approve — approve (log user + timestamp)
  - [ ] POST /api/optimizer/suggestions/{id}/reject — reject (log user + timestamp)

## Dev Notes

### Architecture Context

- **Service:** New `ai-optimizer` (Python/FastAPI)
- **Database:** PostgreSQL (trades table from Epic 1, optimizer_suggestions table)
- **Pattern:** Read-only analysis on trades, suggestions stored for manual review

### Key Architecture Rules

- **FR-75:** Analyze trade history and identify patterns
- **FR-76:** Suggest parameter adjustments with expected impact
- **FR-77:** Require manual approval before applying suggestions
- **NFR-AI1:** Statistical significance p < 0.05
- **NFR-AI2:** No auto-application; manual approval required

### Files to CREATE

**`services/ai-optimizer/app/main.py`** — FastAPI app
**`services/ai-optimizer/app/config.py`** — Config
**`services/ai-optimizer/app/db.py`** — DB connection
**`services/ai-optimizer/app/middleware/auth.py`** — JWT auth
**`services/ai-optimizer/app/models/optimizer.py`** — Pydantic models
**`services/ai-optimizer/app/repos/optimizer_repo.py`** — DB queries
**`services/ai-optimizer/app/routes/optimizer.py`** — API endpoints
**`services/ai-optimizer/app/engine/pattern_analyzer.py`** — Pattern analysis logic
**`services/ai-optimizer/requirements.txt`**
**`services/ai-optimizer/Dockerfile`**
**`migrations/postgres/018_create_optimizer_suggestions.up/down.sql`**

### Database Schema

```sql
CREATE TABLE optimizer_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id VARCHAR(64) NOT NULL,
    pattern_type VARCHAR(50) NOT NULL,
    parameter_name VARCHAR(100) NOT NULL,
    current_value TEXT NOT NULL,
    suggested_value TEXT NOT NULL,
    expected_impact TEXT NOT NULL,  -- e.g., "+12% win rate", "+$500 daily PnL"
    methodology TEXT NOT NULL,      -- plain English explanation
    confidence DECIMAL(5,4) NOT NULL,  -- 0.0-1.0
    p_value DECIMAL(10,8),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    reviewed_by UUID,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_optimizer_suggestions_strategy ON optimizer_suggestions(strategy_id, status);
```

### Pattern Analysis Types

| Pattern | Analysis Method | Significance Test | Expected Impact Format |
|---------|----------------|-------------------|----------------------|
| Time-of-day | Win rate by hour bucket | Chi-squared | "+X% win rate during HH:MM-HH:MM" |
| Score threshold | Win rate by score range | T-test | "+X% win rate if min_score raised to Y" |
| Position sizing | PnL per size bucket | T-test | "+$X daily PnL if size reduced to Y" |

### Suggestion Example

```json
{
  "strategy_id": "strat-1",
  "pattern_type": "time_of_day",
  "parameter_name": "trading_hours",
  "current_value": "00:00-23:59",
  "suggested_value": "08:00-20:00",
  "expected_impact": "+15% win rate, +$200 daily PnL",
  "methodology": "Trades between 8AM-8PM UTC show 72% win rate vs 54% outside hours (p=0.003, n=450). Restricting to profitable hours eliminates 46 losing trades.",
  "confidence": "0.85",
  "p_value": "0.003"
}
```

### Approval Flow

```
1. POST /api/optimizer/analyze → generates suggestions, stores in DB
2. GET /api/optimizer/suggestions?status=pending → user reviews
3. POST /api/optimizer/suggestions/{id}/approve → status='approved', reviewed_by=user_id, reviewed_at=now()
4. User manually applies approved suggestion to strategy-manager
```

Note: Auto-application is intentionally NOT implemented (NFR-AI2). User must manually update strategy parameters.

### References

| Reference | Description |
|-----------|-------------|
| FR-75 | Analyze trade history and identify patterns |
| FR-76 | Suggest parameter adjustments with expected impact |
| FR-77 | Require manual approval before applying |
| NFR-AI1 | Statistical significance p < 0.05 |
| NFR-AI2 | No auto-application; manual approval required |

## Dev Agent Record

### Agent Model Used

mimo-v2.5-pro

### Completion Notes List

- Task 1: ai-optimizer service setup (config, db, auth, main)
- Task 2: Pattern analysis engine (time-of-day, score threshold, position sizing)
- Task 3: Suggestion generator with statistical tests (Chi-squared, T-test)
- Task 4: API endpoints (POST /analyze, GET /suggestions, POST approve/reject)

### File List

**New files:**
- `services/ai-optimizer/app/main.py`
- `services/ai-optimizer/app/config.py`
- `services/ai-optimizer/app/db.py`
- `services/ai-optimizer/app/middleware/auth.py`
- `services/ai-optimizer/app/models/optimizer.py`
- `services/ai-optimizer/app/engine/pattern_analyzer.py`
- `services/ai-optimizer/app/repos/optimizer_repo.py`
- `services/ai-optimizer/app/routes/optimizer.py`
- `services/ai-optimizer/requirements.txt`
- `services/ai-optimizer/Dockerfile`
- `migrations/postgres/018_create_optimizer_suggestions.up/down.sql`
