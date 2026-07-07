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
   **And** patterns include: market characteristics, time-of-day effects, score thresholds, position sizing

2. **Given** patterns are identified
   **When** the optimizer generates suggestions
   **Then** each suggestion includes: parameter change, expected impact (quantified), and methodology
   **And** suggestions are displayed for manual review — no auto-application
   **And** approval (or rejection) is logged

## Tasks / Subtasks

- [ ] Task 1: AI Optimizer Service Setup
  - [ ] Create `services/ai-optimizer/` Python/FastAPI service
  - [ ] Connect to PostgreSQL (trades table)
  - [ ] JWT authentication
- [ ] Task 2: Pattern Analysis Engine
  - [ ] Analyze winning vs losing trades
  - [ ] Identify statistically significant patterns (p < 0.05)
  - [ ] Pattern types: market characteristics, time-of-day, score thresholds, position sizing
- [ ] Task 3: Suggestion Generator
  - [ ] Generate parameter change suggestions
  - [ ] Quantify expected impact
  - [ ] Include methodology explanation
- [ ] Task 4: API Endpoints
  - [ ] POST /api/optimizer/analyze — run analysis
  - [ ] GET /api/optimizer/suggestions — list suggestions
  - [ ] POST /api/optimizer/suggestions/{id}/approve — approve suggestion
  - [ ] POST /api/optimizer/suggestions/{id}/reject — reject suggestion

## Dev Notes

### Architecture Context

- **Service:** New `ai-optimizer` (Python/FastAPI)
- **Database:** PostgreSQL (trades table from Epic 1)
- **Pattern:** Read-only analysis, no auto-application (NFR-AI2)

### Key Architecture Rules

- **FR-75:** Analyze trade history and identify patterns
- **FR-76:** Suggest parameter adjustments with expected impact
- **FR-77:** Require manual approval before applying suggestions
- **NFR-AI1:** Statistical significance p < 0.05
- **NFR-AI2:** No auto-application; manual approval required

### Database Schema

```sql
CREATE TABLE optimizer_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id VARCHAR(64) NOT NULL,
    pattern_type VARCHAR(50) NOT NULL,
    parameter_name VARCHAR(100) NOT NULL,
    current_value TEXT NOT NULL,
    suggested_value TEXT NOT NULL,
    expected_impact TEXT NOT NULL,
    methodology TEXT NOT NULL,
    confidence DECIMAL(5,4) NOT NULL,
    p_value DECIMAL(10,8),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    reviewed_by UUID,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_optimizer_suggestions_strategy ON optimizer_suggestions(strategy_id, status);
CREATE INDEX idx_optimizer_suggestions_created ON optimizer_suggestions(created_at DESC);
```

### Pattern Analysis Types

| Pattern | Analysis Method | Significance Test |
|---------|----------------|-------------------|
| Time-of-day | Win rate by hour | Chi-squared |
| Score threshold | Win rate by score bucket | T-test |
| Market characteristics | Win rate by market type | Chi-squared |
| Position sizing | PnL by size bucket | T-test |

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

### Debug Log References

### Completion Notes List

### File List
