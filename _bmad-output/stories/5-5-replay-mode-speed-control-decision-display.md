# Story 5.5: Replay Mode — Speed Control & Decision Display

Status: ready-for-dev

## Story

As a quant trader,
I want to replay historical market events at configurable speed and see the bot's decision-making process,
So that I can debug specific trades and understand bot behavior.

## Acceptance Criteria

1. **Given** historical market data exists for a selected date range
   **When** the user starts a replay
   **Then** market events are replayed at the selected speed (1x, 2x, 5x, 10x)
   **And** the replay is accurate — matches historical data exactly
   **And** playback is smooth at 10x speed

2. **Given** the replay is running
   **When** the bot processes each market event
   **Then** bot decisions are displayed in real-time: what was detected, what was decided, and why
   **And** the decision log matches the historical execution records

3. **Given** the user interacts with replay controls
   **When** pause, step-forward, or rewind is triggered
   **Then** controls are responsive within 100ms
   **And** state is consistent after any control action

4. **Given** risk events occurred during the historical period
   **When** those events are replayed
   **Then** risk events (circuit breaker triggers, limit breaches) are visually highlighted
   **And** details are available on click

## Tasks / Subtasks

- [ ] Task 1: Replay API Endpoint (AC: #1, #2)
  - [ ] Add POST /api/backtesting/replay endpoint
  - [ ] Accept date range and speed parameter
  - [ ] Return replay session ID
  - [ ] Stream events via WebSocket or SSE
- [ ] Task 2: Decision Display (AC: #2)
  - [ ] Log each decision: detected, decided, why
  - [ ] Include market context, score, risk check result
  - [ ] Stream to frontend in real-time
- [ ] Task 3: Replay Controls (AC: #3)
  - [ ] Pause/resume
  - [ ] Step-forward (one event at a time)
  - [ ] Speed control (1x, 2x, 5x, 10x)
- [ ] Task 4: Risk Event Highlighting (AC: #4)
  - [ ] Detect risk events in historical data
  - [ ] Highlight in replay UI
  - [ ] Show details on click

## Dev Notes

### Architecture Context

- **Service:** `backtesting` (Python/FastAPI) — extends Story 5.1
- **Frontend:** Dashboard replay page
- **Data source:** TimescaleDB (historical market data)
- **Pattern:** Stream events via WebSocket with speed control

### Key Architecture Rules

- **FR-96:** Replay at configurable speed (1x, 2x, 5x, 10x)
- **FR-97:** Display bot decisions in real-time
- **FR-98:** Pause, step-forward, rewind controls
- **FR-99:** Highlight risk events
- **NFR-RP1:** Replay accuracy — matches historical data exactly
- **NFR-RP2:** Smooth at 10x speed
- **NFR-RP3:** Controls responsive within 100ms

### Files to MODIFY

**`services/backtesting/app/routes/backtest.py`**
- Current: POST /run, GET /status, GET /results, GET /report, POST /sweep
- Change: Add POST /replay endpoint
- Preserve: All existing endpoints

**`services/backtesting/app/engine/backtest_engine.py`**
- Current: `run_backtest()` processes all data at once
- Change: Add `replay_events()` generator that yields events one at a time
- Preserve: Existing batch processing

### Files to CREATE

**`services/dashboard/src/app/replay/page.tsx`**
- Replay page with speed controls and decision display

**`services/dashboard/src/components/replay/ReplayControls.tsx`**
- Pause, step-forward, speed selector

**`services/dashboard/src/components/replay/DecisionLog.tsx`**
- Real-time decision display

### Replay Event Structure

```python
class ReplayEvent(BaseModel):
    event_type: str  # "market_update", "opportunity", "decision", "risk_event"
    timestamp: str
    data: dict
```

### Decision Display Format

```python
class DecisionDisplay(BaseModel):
    timestamp: str
    market_id: str
    detected: str  # "YES+NO arbitrage"
    decision: str  # "EXECUTE" / "SKIP" / "FILTER"
    reason: str    # "Score below threshold" / "Risk check denied"
    score: str
    risk_result: str
```

### References

| Reference | Description |
|-----------|-------------|
| FR-96 | Replay at configurable speed |
| FR-97 | Display bot decisions in real-time |
| FR-98 | Pause, step-forward, rewind controls |
| FR-99 | Highlight risk events |
| NFR-RP1 | Replay accuracy |
| NFR-RP2 | Smooth at 10x speed |
| NFR-RP3 | Controls responsive within 100ms |

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
