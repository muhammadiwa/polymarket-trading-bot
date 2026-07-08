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
  - [ ] Fetch current risk state from risk-manager Redis (Pit Boss state)
  - [ ] Fetch recent trade performance from analytics service
  - [ ] Compare current parameters against performance metrics
  - [ ] Generate conservative suggestions (only reduce risk, never increase)
  - [ ] Include rationale with specific data points
- [ ] Task 2: Read-only Safety Enforcement (AC: #2)
  - [ ] Verify all existing endpoints are read-only (already true)
  - [ ] Add documentation comment to each endpoint confirming read-only
  - [ ] No write endpoints to trades, strategies, or risk tables
- [ ] Task 3: API Endpoint (AC: #1)
  - [ ] POST /api/assistant/suggest-risk-parameters
  - [ ] Request: optional strategy_id filter
  - [ ] Response: list of suggestions with parameter, current, suggested, rationale
  - [ ] Suggestions are NOT auto-applied — user must manually update

## Dev Notes

### Architecture Context

- **Service:** `ai-assistant` (Python/FastAPI) — extends Story 6.3
- **Data source:** risk-manager Redis (Pit Boss state), analytics service API
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

### Files to CREATE

**`services/ai-assistant/app/engine/risk_advisor.py`**
- Risk parameter suggestion logic
- Conservative rule engine
- Rationale generator

### Risk State Data Source

```python
# Fetch current risk state from risk-manager API
async def get_risk_state() -> dict:
    """Read Pit Boss state from risk-manager Redis via API."""
    # GET /api/risk/state returns:
    # - daily_budget_remaining
    # - current_drawdown
    # - win_streak_current
    # - circuit_breaker_status
    # - market_limits
    # - strategy_limits
```

### Conservative Suggestion Rules

| Parameter | Direction | Trigger Condition | Rationale Format |
|-----------|-----------|-------------------|-----------------|
| Daily loss limit | Decrease only | Current drawdown > 5% | "Drawdown at X%, reducing limit to Y% limits daily exposure" |
| Max position per market | Decrease only | Any market > 80% utilization | "Market X at Y% utilization, reducing limit to Z%" |
| Max position per strategy | Decrease only | Any strategy > 80% utilization | "Strategy X at Y% utilization, reducing limit to Z%" |
| Score threshold | Increase only | Win rate < 60% | "Win rate at X%, raising threshold filters low-quality trades" |
| Slippage tolerance | Decrease only | Avg slippage > 0.5% | "Avg slippage at X%, tighter tolerance reduces execution cost" |

### Request/Response Format

**POST /api/assistant/suggest-risk-parameters:**
```json
Request: { "strategy_id": "optional-filter" }

Response: {
  "suggestions": [
    {
      "parameter": "daily_loss_limit",
      "current_value": "2.0",
      "suggested_value": "1.5",
      "direction": "decrease",
      "rationale": "Current drawdown is 7.2%, approaching the 10% circuit breaker. Reducing daily loss limit to 1.5% limits exposure during volatile periods.",
      "confidence": "high",
      "data_points": ["current_drawdown: 7.2%", "7_day_avg_drawdown: 4.1%"]
    }
  ],
  "analysis_timestamp": "2025-01-15T10:30:00Z",
  "read_only": true,
  "requires_approval": true
}
```

### Read-only Verification

All existing ai-assistant endpoints are read-only:
- POST /ask — queries data, no writes
- POST /explain-trade — reads trade data, no writes
- GET /history — reads conversation history, no writes
- POST /suggest-risk-parameters — generates suggestions, no writes (NEW)

No endpoints write to: trades, strategies, risk_events, positions tables.

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
