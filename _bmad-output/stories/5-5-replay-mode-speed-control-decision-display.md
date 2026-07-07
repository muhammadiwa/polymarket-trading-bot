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
   **And** the replay is accurate — events are served in chronological order from TimescaleDB
   **And** playback is smooth at 10x speed (no dropped events)

2. **Given** the replay is running
   **When** the bot processes each market event
   **Then** bot decisions are displayed in real-time: what was detected, what was decided, and why
   **And** decisions are derived from the same backtest engine logic (deterministic)

3. **Given** the user interacts with replay controls
   **When** pause or step-forward is triggered
   **Then** controls are responsive within 100ms (frontend state change)
   **And** state is consistent after any control action

4. **Given** risk events occurred during the historical period
   **When** those events are replayed
   **Then** risk events (circuit breaker triggers, limit breaches) are visually highlighted
   **And** details are available on click

## Tasks / Subtasks

- [ ] Task 1: Replay API Endpoint (AC: #1, #2)
  - [ ] Add POST /api/backtesting/replay endpoint to existing backtesting service
  - [ ] Accept date range and speed parameter
  - [ ] Return replay session ID
  - [ ] Stream events via Server-Sent Events (SSE) — simpler than WebSocket for one-way streaming
- [ ] Task 2: Decision Display (AC: #2)
  - [ ] Log each decision: detected type, decision (EXECUTE/SKIP/FILTER), reason
  - [ ] Include market_id, score, risk check result
  - [ ] Stream as SSE event with type="decision"
- [ ] Task 3: Replay Controls (AC: #3)
  - [ ] Frontend: pause/resume (toggle SSE consumption)
  - [ ] Frontend: step-forward (request next event from API)
  - [ ] Frontend: speed selector (1x, 2x, 5x, 10x)
- [ ] Task 4: Risk Event Highlighting (AC: #4)
  - [ ] Detect risk events in historical data (circuit breaker, limit breach)
  - [ ] Stream as SSE event with type="risk_event"
  - [ ] Frontend: visual highlight + click for details

## Dev Notes

### Architecture Context

- **Service:** `backtesting` (Python/FastAPI) — extends Story 5.1
- **Frontend:** Dashboard replay page (Next.js)
- **Data source:** TimescaleDB (historical market data from `opportunities` table)
- **Pattern:** SSE (Server-Sent Events) for one-way server→client streaming

### Key Architecture Rules

- **FR-96:** Replay at configurable speed (1x, 2x, 5x, 10x)
- **FR-97:** Display bot decisions in real-time
- **FR-98:** Pause, step-forward controls
- **FR-99:** Highlight risk events
- **NFR-RP1:** Replay accuracy — events in chronological order
- **NFR-RP2:** Smooth at 10x speed (no dropped events)
- **NFR-RP3:** Controls responsive within 100ms

### Files to MODIFY

**`services/backtesting/app/routes/backtest.py`**
- Current: POST /run, GET /status, GET /results, GET /report, POST /sweep
- Change: Add POST /replay, GET /replay/{id}/events (SSE), POST /replay/{id}/step
- Preserve: All existing endpoints

**`services/backtesting/app/engine/backtest_engine.py`**
- Current: `run_backtest()` processes all data at once
- Change: Add `replay_events()` generator that yields events one at a time with delays
- Preserve: Existing batch processing

**`services/backtesting/app/models/backtest.py`**
- Current: BacktestRequest, BacktestResults, etc.
- Change: Add ReplayRequest, ReplayEvent models
- Preserve: All existing models

### Files to CREATE

**`services/dashboard/src/app/replay/page.tsx`**
- Replay page with speed controls and decision display

**`services/dashboard/src/components/replay/ReplayControls.tsx`**
- Pause, step-forward, speed selector

**`services/dashboard/src/components/replay/DecisionLog.tsx`**
- Real-time decision display

### SSE Streaming Choice

SSE (not WebSocket) because:
- One-way server→client streaming (replay is read-only)
- Simpler implementation (no upgrade handshake)
- Auto-reconnect built into EventSource API
- Works through proxies/firewalls

### Replay API

**POST /api/backtesting/replay:**
```json
Request: { "strategy_id": "...", "start_date": "2025-01-01", "end_date": "2025-01-31", "speed": 2 }
Response: { "session_id": "uuid", "status": "started" }
```

**GET /api/backtesting/replay/{session_id}/events (SSE):**
```
event: market_update
data: {"timestamp": "...", "market_id": "...", "spread": "0.05", "score": "0.02"}

event: decision
data: {"timestamp": "...", "market_id": "...", "detected": "YES+NO arbitrage", "decision": "EXECUTE", "reason": "Score above threshold", "score": "0.02"}

event: risk_event
data: {"timestamp": "...", "type": "circuit_breaker", "message": "5 consecutive API errors"}

event: done
data: {"total_events": 1234, "total_decisions": 56}
```

**POST /api/backtesting/replay/{session_id}/step:**
```json
Response: { "event": {...}, "has_more": true }
```

### Decision Display Format

```python
class DecisionDisplay(BaseModel):
    timestamp: str
    market_id: str
    detected: str  # "YES+NO arbitrage", "Cross-market arbitrage"
    decision: str  # "EXECUTE", "SKIP", "FILTER"
    reason: str    # "Score above threshold", "Risk check denied", "Below min profit"
    score: str
    risk_result: str  # "ALLOWED", "DENIED"
```

### Risk Event Detection

Risk events are detected by checking historical data for:
- Circuit breaker triggers (5+ consecutive API errors in trade history)
- Drawdown limit breaches (drawdown > threshold in risk metrics)
- Daily budget exhaustion (daily loss > limit)

These are flagged in the opportunity/trade data by the existing backtest engine.

### References

| Reference | Description |
|-----------|-------------|
| FR-96 | Replay at configurable speed |
| FR-97 | Display bot decisions in real-time |
| FR-98 | Pause, step-forward controls |
| FR-99 | Highlight risk events |
| NFR-RP1 | Replay accuracy |
| NFR-RP2 | Smooth at 10x speed |
| NFR-RP3 | Controls responsive within 100ms |

## Dev Agent Record

### Agent Model Used

mimo-v2.5-pro

### Completion Notes List

- Task 1: Replay engine with speed-controlled async generator
- Task 2: SSE streaming endpoint + step-forward endpoint
- Task 3: Frontend replay page with controls and decision log

### File List

**New files:**
- `services/backtesting/app/engine/replay_engine.py`
- `services/backtesting/app/routes/replay.py`
- `services/dashboard/src/app/replay/page.tsx`
- `services/dashboard/src/components/replay/ReplayControls.tsx`
- `services/dashboard/src/components/replay/DecisionLog.tsx`

**Modified files:**
- `services/backtesting/app/models/backtest.py` — added ReplayRequest, ReplayEvent, DecisionDisplay
- `services/backtesting/app/main.py` — registered replay router
