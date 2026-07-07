# Story 5.4: Paper Trading — Seamless Live Switch

Status: ready-for-dev

## Story

As a quant trader,
I want to seamlessly switch a strategy from paper trading to live trading with the same configuration,
So that I can go live with a validated strategy without re-entering parameters.

## Acceptance Criteria

1. **Given** a strategy has been running in paper trading mode
   **When** the user switches the execution mode from PAPER to LIVE via PUT /api/execution-mode
   **Then** the mode is persisted in Redis immediately
   **And** the mode change is logged with timestamp and user
   **And** a restart confirmation is required before execution engine picks up the new mode
   **And** no strategy configuration re-entry is required — strategies in PostgreSQL remain unchanged

2. **Given** the mode is switched from PAPER to LIVE
   **When** the system processes the mode change
   **Then** open paper positions are logged to the API response for the user to review
   **And** a warning notification is published to NATS (`pqap.notification.request`)
   **And** the notification warns that live trading will start after restart

3. **Given** the user confirms restart
   **When** POST /api/execution-mode/restart is called
   **Then** the restart is logged
   **And** the execution engine reads the new mode from Redis on next startup
   **And** live trading begins with the same strategy parameters (no config re-entry)

## Tasks / Subtasks

- [ ] Task 1: Mode Switch Enhancement (AC: #1)
  - [ ] Enhance PUT /api/execution-mode to return open paper positions count
  - [ ] Log mode change with full context (from, to, user, timestamp)
  - [ ] Return `restart_required: true` in response
- [ ] Task 2: Paper Position Handoff (AC: #2)
  - [ ] Query open paper positions count from DB on PAPER→LIVE switch
  - [ ] Publish warning notification to NATS
  - [ ] Include open positions count in response
- [ ] Task 3: Restart Confirmation (AC: #3)
  - [ ] Add POST /api/execution-mode/restart endpoint
  - [ ] Validate current mode before restart
  - [ ] Log restart confirmation
  - [ ] Return confirmation with current mode

## Dev Notes

### Architecture Context

- **Service:** api-gateway (Python) — extends Story 5.3
- **Database:** PostgreSQL `paper_positions` table (from Story 5.3)
- **Redis:** `pqap:execution_mode` key
- **NATS:** `pqap.notification.request` for warning notification
- **Pattern:** Mode switch sets Redis key; restart confirmation required per AD-12

### Key Architecture Rules

- **AD-12:** Mode switch requires restart to prevent accidental live trades
- **FR-94:** Seamless switch from paper to live with same config
- **NFR-PT3:** Mode switch within 1 second (after restart confirmation)

### Files to MODIFY

**`services/api-gateway/app/routes/execution_mode.py`**
- Current: GET/PUT /api/execution-mode
- Change: Enhance PUT to query paper positions, publish NATS warning, add POST /restart
- Preserve: Existing mode read/write logic

### Mode Switch Flow

```
1. PUT /api/execution-mode {mode: "LIVE"}
   → Sets Redis key
   → Queries open paper positions
   → Publishes NATS warning notification
   → Returns {mode, message, restart_required: true, open_paper_positions: N}
2. User reviews open paper positions
3. POST /api/execution-mode/restart {confirm: true}
   → Logs restart confirmation
   → Returns {mode, message: "Restart confirmed"}
4. Service restarts (external orchestrator reads new mode from Redis)
5. Execution engine reads LIVE mode, starts real trading
```

### NATS Warning Notification

```python
notif_event = {
    "event_id": str(uuid4()),
    "event_type": "NotificationRequest",
    "timestamp": datetime.now(timezone.utc).isoformat(),
    "source": "api-gateway",
    "payload": {
        "category": "risk",
        "title": "Mode Switch: PAPER → LIVE",
        "message": f"Live trading will start after restart. {open_count} paper positions open.",
        "channel": "telegram",
        "priority": "high",
        "bypass_throttle": True,
    },
}
await nc.publish("pqap.notification.request", json.dumps(notif_event).encode())
```

### References

| Reference | Description |
|-----------|-------------|
| FR-94 | Seamless switch to live with same strategy config |
| AD-12 | Mode switch requires restart |
| NFR-PT3 | Mode switch within 1 second |

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
