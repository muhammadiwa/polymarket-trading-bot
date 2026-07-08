# Story 6.4: AI Assistant — Read-only Safety & Risk Parameter Suggestions

Status: ready-for-dev

## Story

As a quant trader,
I want the AI assistant to suggest conservative risk parameter adjustments while being strictly read-only,
So that I get helpful suggestions without any risk of the assistant modifying my system.

## Acceptance Criteria

1. **Given** the AI assistant is analyzing current risk state
   **When** it generates risk parameter suggestions
   **Then** suggestions are conservative — never recommending increasing risk beyond current limits
   **And** suggestions include rationale based on current state and historical performance

2. **Given** the assistant is operational
   **When** any action is attempted
   **Then** the assistant CANNOT execute trades or modify configurations directly
   **And** all suggested actions require explicit human approval
   **And** the assistant has read-only access to analytics, positions, and trade history

## Tasks / Subtasks

- [ ] Task 1: Risk Parameter Suggestion Engine (AC: #1)
  - [ ] Analyze current risk state from risk-manager API
  - [ ] Generate conservative suggestions (only reduce risk, never increase)
  - [ ] Include rationale with data points
- [ ] Task 2: Read-only Safety Enforcement (AC: #2)
  - [ ] Verify all API endpoints are read-only (no write operations)
  - [ ] Ensure no trade execution or config modification capability
  - [ ] Document read-only constraint
- [ ] Task 3: API Endpoint
  - [ ] POST /api/assistant/suggest-risk-parameters
  - [ ] Returns suggestions with rationale (no auto-apply)

## Dev Notes

### Architecture Context

- **Service:** `ai-assistant` (Python/FastAPI) — extends Story 6.3
- **Data source:** risk-manager API (read-only), analytics service
- **Pattern:** Read-only analysis, suggestions only, no execution

### Key Architecture Rules

- **FR-102:** Suggest risk parameter adjustments based on current state
- **FR-103:** CANNOT execute trades or modify configurations directly
- **NFR-AA2:** Read-only access to analytics, positions, and trade history

### Files to MODIFY

**`services/ai-assistant/app/routes/assistant.py`**
- Current: POST /ask, POST /explain-trade, GET /history
- Change: Add POST /suggest-risk-parameters endpoint
- Preserve: All existing endpoints

**`services/ai-assistant/app/engine/performance_qa.py`**
- Current: answer_question() for general queries
- Change: Add suggest_risk_parameters() for risk analysis
- Preserve: Existing Q&A functionality

### Conservative Suggestion Rules

| Parameter | Suggestion Direction | Constraint |
|-----------|---------------------|------------|
| Daily loss limit | Only decrease | Never suggest > current |
| Max position per market | Only decrease | Never suggest > current |
| Max position per strategy | Only decrease | Never suggest > current |
| Score threshold | Only increase | Never suggest < current |
| Slippage tolerance | Only decrease | Never suggest > current |

### References

| Reference | Description |
|-----------|-------------|
| FR-102 | Suggest risk parameter adjustments based on current state |
| FR-103 | CANNOT execute trades or modify configurations directly |
| NFR-AA2 | Read-only access to analytics, positions, and trade history |

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
