# Story 1.10: Telegram Notifications

## Metadata

| Field | Value |
|-------|-------|
| **Story ID** | 1.10 |
| **Story Key** | 1-10-telegram-notifications |
| **Epic** | Epic 1: Foundation — Bot Can Hunt |
| **Priority** | P0 (FR-80, FR-81), P1 (FR-82, FR-83), P2 (FR-84) |
| **Service** | `notification` (Python) |
| **Estimated Effort** | 3-5 days |

---

## User Story

**As a** quant trader,
**I want to** receive critical trading alerts via Telegram,
**So that** I'm immediately informed of emergency stops, circuit breakers, and significant trades.

---

## Acceptance Criteria

### AC-1: Telegram Delivery
**Given** the notification service is configured with a Telegram bot token and chat ID
**When** a `NotificationRequest` event is published to NATS on subject `pqap.notification.send`
**Then** the notification is delivered to the configured Telegram chat within 5 seconds
**And** delivery is confirmed via Telegram API response
**And** failed deliveries are retried with exponential backoff (max 3 retries)

### AC-2: Severity Categorization
**Given** a notification request is received
**When** the notification is processed
**Then** it is categorized into one of four severity levels: `critical`, `warning`, `info`, `debug`
**And** the category is determined by the event type:
- **Critical**: `EmergencyStop`, `CircuitBreakerTripped`, `APIFailure`, `DrawdownBreach`, `DailyBudgetExhausted`
- **Warning**: `DailyBudget80Percent`, `DrawdownApproaching`, `PositionLimitBreach`, `WinStreak`
- **Info**: `OrderFilled`, `TradeExecuted`, `StrategyOptimization`
- **Debug**: `SystemHealth`, `ReconnectionEvent`, `ReconciliationComplete`

### AC-3: Critical Notification Bypass
**Given** a critical notification is received
**When** the throttler evaluates the notification
**Then** the critical notification bypasses all throttling rules
**And** is delivered immediately regardless of rate limits
**And** critical notifications are never suppressed by user preferences

### AC-4: Non-Critical Throttling
**Given** non-critical notifications (warning, info, debug) are being generated
**When** the rate exceeds 10 notifications per minute
**Then** excess notifications are queued and delivered in the next minute window
**And** throttled notifications are logged with throttle reason
**And** the throttle window resets every 60 seconds (sliding window)

### AC-5: Configurable Preferences
**Given** the user wants to configure notification preferences
**When** preferences are updated via API or configuration
**Then** preferences are persisted to PostgreSQL
**And** changes take effect immediately (no restart required)
**And** the user can enable/disable each category independently:
- `critical_enabled` (default: `true`, cannot be set to `false`)
- `warning_enabled` (default: `true`)
- `info_enabled` (default: `true`)
- `debug_enabled` (default: `false`)

### AC-6: Notification History
**Given** notifications are being processed
**When** a notification is delivered (or throttled)
**Then** a record is written to the PostgreSQL `notifications` table with:
- `id` (UUID, primary key)
- `event_type` (string)
- `severity` (enum: critical, warning, info, debug)
- `title` (string)
- `message` (text)
- `channel` (enum: telegram, email)
- `status` (enum: delivered, failed, throttled)
- `delivered_at` (TIMESTAMPTZ, nullable)
- `created_at` (TIMESTAMPTZ)
- `metadata` (JSONB)
**And** only the last 1000 notifications are retained (older records are purged)

### AC-7: Message Formatting
**Given** a notification is ready to be sent
**When** the message is formatted for Telegram
**Then** the message includes:
- Severity emoji prefix (🔴 Critical, 🟡 Warning, 🔵 Info, ⚪ Debug)
- Title (bold)
- Timestamp (UTC)
- Message body with relevant details
- For critical notifications: all caps title for emphasis

---

## Technical Design

### Architecture Context

```
┌─────────────────────────────────────────────────────────────────┐
│                        NATS Event Bus                            │
│  pqap.notification.send (NotificationRequest)                   │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────────┐
│                    Notification Service (Python)                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │ NATS         │  │              │  │                      │   │
│  │ Subscriber   │──│ Categorizer  │──│ Throttler            │   │
│  │ (adapter)    │  │              │  │ (sliding window)     │   │
│  └──────────────┘  └──────────────┘  └──────────┬───────────┘   │
│                                                  │               │
│                                      ┌───────────▼───────────┐   │
│                                      │ Telegram Adapter      │   │
│                                      │ (python-telegram-bot) │   │
│                                      └───────────────────────┘   │
│                                                  │               │
│                                      ┌───────────▼───────────┐   │
│                                      │ PostgreSQL Adapter    │   │
│                                      │ (notification history)│   │
│                                      └───────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
```

### Service Structure

```
services/notification/
├── app/
│   ├── __init__.py
│   ├── main.py              # Service entrypoint, lifecycle management
│   ├── config.py            # Configuration (env vars, defaults)
│   ├── models.py            # Pydantic models for notifications
│   ├── categorizer.py       # Severity classification logic
│   ├── throttler.py         # Sliding window rate limiter
│   ├── formatter.py         # Telegram message formatting
│   └── history.py           # Notification history management
├── adapters/
│   ├── __init__.py
│   ├── nats_subscriber.py   # NATS event subscription
│   ├── telegram_bot.py      # Telegram Bot API adapter
│   └── postgres_repo.py     # PostgreSQL repository
├── ports/
│   ├── __init__.py
│   ├── event_port.py        # Event subscription interface
│   ├── notify_port.py       # Notification delivery interface
│   └── storage_port.py      # History storage interface
├── Dockerfile
├── requirements.txt
└── pyproject.toml
```

### NATS Event Schema

**Subject:** `pqap.notification.send`

**Event Type:** `NotificationRequest`

```json
{
  "event_id": "uuid-v4",
  "event_type": "NotificationRequest",
  "timestamp": "2026-07-04T12:00:00Z",
  "source": "risk-manager",
  "payload": {
    "title": "Emergency Stop Triggered",
    "message": "Trading halted due to API death spiral. All open orders cancelled.",
    "severity": "critical",
    "metadata": {
      "reason": "api_death_spiple",
      "consecutive_errors": 5,
      "triggered_by": "circuit_breaker"
    }
  }
}
```

### Severity Classification Rules

| Event Type | Severity | Bypass Throttle | Emoji |
|------------|----------|-----------------|-------|
| `EmergencyStop` | critical | Yes | 🔴 |
| `CircuitBreakerTripped` | critical | Yes | 🔴 |
| `APIFailure` | critical | Yes | 🔴 |
| `DrawdownBreach` | critical | Yes | 🔴 |
| `DailyBudgetExhausted` | critical | Yes | 🔴 |
| `DailyBudget80Percent` | warning | No | 🟡 |
| `DrawdownApproaching` | warning | No | 🟡 |
| `PositionLimitBreach` | warning | No | 🟡 |
| `WinStreak` | warning | No | 🟡 |
| `OrderFilled` | info | No | 🔵 |
| `TradeExecuted` | info | No | 🔵 |
| `StrategyOptimization` | info | No | 🔵 |
| `SystemHealth` | debug | No | ⚪ |
| `ReconnectionEvent` | debug | No | ⚪ |
| `ReconciliationComplete` | debug | No | ⚪ |

### Throttler Implementation

**Algorithm:** Sliding window counter

```python
class Throttler:
    def __init__(self, max_per_minute: int = 10):
        self.max_per_minute = max_per_minute
        self.window_start = time.time()
        self.count = 0
    
    def should_allow(self, severity: str) -> bool:
        # Critical always bypasses
        if severity == "critical":
            return True
        
        # Reset window if expired
        now = time.time()
        if now - self.window_start >= 60:
            self.window_start = now
            self.count = 0
        
        # Check limit
        if self.count >= self.max_per_minute:
            return False
        
        self.count += 1
        return True
```

### Telegram Message Templates

**Critical:**
```
🔴 CRITICAL: {title}
━━━━━━━━━━━━━━━━━━━━
{message}

⏰ {timestamp_utc}
📋 Event: {event_type}
```

**Warning:**
```
🟡 Warning: {title}
{message}

⏰ {timestamp_utc}
```

**Info:**
```
🔵 Info: {title}
{message}

⏰ {timestamp_utc}
```

**Debug:**
```
⚪ Debug: {title}
{message}
```

### Database Schema

```sql
-- migrations/postgres/006_create_notifications.up.sql
CREATE TYPE notification_severity AS ENUM ('critical', 'warning', 'info', 'debug');
CREATE TYPE notification_status AS ENUM ('delivered', 'failed', 'throttled');
CREATE TYPE notification_channel AS ENUM ('telegram', 'email');

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    severity notification_severity NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    channel notification_channel NOT NULL DEFAULT 'telegram',
    status notification_status NOT NULL,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'
);

-- Index for history queries
CREATE INDEX idx_notifications_created_at ON notifications (created_at DESC);
CREATE INDEX idx_notifications_severity ON notifications (severity);

-- Retention: keep only last 1000
-- Implemented via application logic (purge on insert when count > 1000)
```

### Configuration

Environment variables:

```bash
# Telegram Configuration
TELEGRAM_BOT_TOKEN=your-bot-token-here
TELEGRAM_CHAT_ID=your-chat-id-here

# NATS Configuration
NATS_URL=nats://nats:4222
NATS_SUBJECT=pqap.notification.send

# PostgreSQL Configuration
DATABASE_URL=postgresql://user:pass@postgres:5432/pqap

# Throttling
NOTIFICATION_MAX_PER_MINUTE=10
NOTIFICATION_HISTORY_LIMIT=1000

# Preferences (defaults)
NOTIFICATION_CRITICAL_ENABLED=true
NOTIFICATION_WARNING_ENABLED=true
NOTIFICATION_INFO_ENABLED=true
NOTIFICATION_DEBUG_ENABLED=false
```

### Prometheus Metrics

```
pqap_notification_sent_total{channel="telegram",severity="critical"} counter
pqap_notification_sent_total{channel="telegram",severity="warning"} counter
pqap_notification_sent_total{channel="telegram",severity="info"} counter
pqap_notification_sent_total{channel="telegram",severity="debug"} counter
pqap_notification_failed_total{channel="telegram",reason="api_error"} counter
pqap_notification_failed_total{channel="telegram",reason="timeout"} counter
pqap_notification_throttled_total{severity="warning"} counter
pqap_notification_throttled_total{severity="info"} counter
pqap_notification_throttled_total{severity="debug"} counter
pqap_notification_delivery_latency_seconds{channel="telegram"} histogram
pqap_notification_queue_size gauge
```

---

## Implementation Tasks

### Task 1: Project Setup
- [ ] Create `services/notification/` directory structure
- [ ] Create `pyproject.toml` with dependencies:
  - `python-telegram-bot>=20.0`
  - `nats-py>=2.0`
  - `asyncpg>=0.29.0`
  - `pydantic>=2.0`
  - `prometheus-client>=0.19.0`
- [ ] Create `Dockerfile` (Python 3.13-slim base)
- [ ] Create `requirements.txt`

### Task 2: Core Models
- [ ] Create `app/models.py` with Pydantic models:
  - `NotificationRequest` (event payload)
  - `NotificationRecord` (database record)
  - `NotificationPreferences` (user config)
  - `Severity` enum (critical, warning, info, debug)
  - `NotificationStatus` enum (delivered, failed, throttled)

### Task 3: Categorizer
- [ ] Create `app/categorizer.py`
- [ ] Implement event type → severity mapping
- [ ] Support custom severity overrides via configuration

### Task 4: Throttler
- [ ] Create `app/throttler.py`
- [ ] Implement sliding window rate limiter
- [ ] Critical bypass logic
- [ ] Window reset every 60 seconds

### Task 5: Telegram Adapter
- [ ] Create `adapters/telegram_bot.py`
- [ ] Implement `send_message()` with retry logic
- [ ] Implement message formatting per severity
- [ ] Handle Telegram API errors gracefully

### Task 6: NATS Subscriber
- [ ] Create `adapters/nats_subscriber.py`
- [ ] Subscribe to `pqap.notification.send`
- [ ] Deserialize `NotificationRequest` events
- [ ] Implement idempotent processing (dedup by `event_id`)

### Task 7: History Repository
- [ ] Create `adapters/postgres_repo.py`
- [ ] Implement `save_notification()` (INSERT)
- [ ] Implement `get_history()` (SELECT with pagination)
- [ ] Implement `purge_old()` (keep last 1000)

### Task 8: Preferences
- [ ] Create `app/config.py` with preference management
- [ ] Load preferences from environment/config
- [ ] Support runtime preference updates via API

### Task 9: Main Service
- [ ] Create `app/main.py` with service lifecycle
- [ ] Wire up all adapters and ports
- [ ] Implement graceful shutdown
- [ ] Add health check endpoint

### Task 10: Metrics
- [ ] Add Prometheus metrics export
- [ ] Track: sent, failed, throttled, latency
- [ ] Expose `/metrics` endpoint

### Task 11: Testing
- [ ] Unit tests for categorizer
- [ ] Unit tests for throttler
- [ ] Unit tests for formatter
- [ ] Integration test with mock Telegram API
- [ ] Integration test with mock NATS

### Task 12: Documentation
- [ ] Update architecture docs with notification service details
- [ ] Document configuration options
- [ ] Document Telegram bot setup instructions

---

## Dependencies

### Upstream (Events Consumed)
| Service | Event | NATS Subject |
|---------|-------|--------------|
| risk-manager | `RiskAlert` | `pqap.risk.alert` |
| risk-manager | `EmergencyStop` | `pqap.risk.emergency` |
| execution-engine | `OrderFilled` | `pqap.order.filled` |
| execution-engine | `OrderFailed` | `pqap.order.failed` |
| ai-optimizer | `OptimizationSuggestion` | `pqap.optimization.suggestion` |

### Downstream (None)
- Notification service is a terminal consumer — it delivers to Telegram/email only

### Infrastructure
| Dependency | Purpose |
|------------|---------|
| NATS | Event subscription |
| PostgreSQL | Notification history, preferences |
| Telegram Bot API | Message delivery |
| Prometheus | Metrics export |

---

## Risks & Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Telegram API rate limits | Notifications delayed | Medium | Implement per-chat rate limiting (30 msg/sec per chat) |
| Telegram bot token compromised | Unauthorized notifications | Low | Rotate token quarterly; monitor for unusual activity |
| NATS connection lost | No notifications delivered | Low | Reconnect with backoff; queue locally during disconnect |
| PostgreSQL down | No history saved | Low | Log to stdout as fallback; retry writes with backoff |
| Chat ID changes | Notifications sent to wrong chat | Low | Validate chat ID on startup; fail fast if invalid |

---

## Testing Strategy

### Unit Tests
- `test_categorizer.py`: Verify event type → severity mapping
- `test_throttler.py`: Verify rate limiting, window reset, critical bypass
- `test_formatter.py`: Verify message formatting per severity

### Integration Tests
- `test_telegram_adapter.py`: Mock Telegram API, verify delivery
- `test_nats_subscriber.py`: Mock NATS, verify event processing
- `test_history_repo.py`: Test PostgreSQL read/write

### E2E Tests
- Full notification flow: NATS event → categorize → throttle → deliver → history

---

## Definition of Done

- [ ] Notification service deployed and running
- [ ] Telegram bot configured and sending messages
- [ ] All severity levels working (critical, warning, info, debug)
- [ ] Throttling working for non-critical notifications
- [ ] Critical bypass working (never throttled)
- [ ] Notification history stored in PostgreSQL
- [ ] Prometheus metrics exported
- [ ] Unit tests passing (>80% coverage)
- [ ] Integration tests passing
- [ ] Documentation updated

---

## References

| Reference | Description |
|-----------|-------------|
| FR-80 | Telegram delivery requirement |
| FR-81 | Severity categorization requirement |
| FR-82 | Throttling requirement |
| FR-83 | Configurable preferences requirement |
| FR-84 | Notification history requirement |
| NFR-N1 | Critical notification latency (<5s) |
| NFR-N2 | Critical notification delivery rate (99.9%) |
| NFR-N3 | Non-critical throttling (max 10/min) |
| AD-9 | NATS event bus conventions |
| AD-10 | Communication patterns (notification is async fire-and-forget) |
| AD-11 | Circuit breaker pattern (Telegram API) |
| AD-17 | Observability (Prometheus metrics) |
| INF-2 | Python 3.13.14 for AI/analytics services |
| INF-14 | Structured JSON logging |

---

## Notes

### Security Considerations
- Telegram bot token stored in Kubernetes Secrets (AD-14)
- Token injected as environment variable, never logged
- Chat ID validated on startup
- No PII in notification messages (only trading data)

### Future Enhancements (Out of Scope)
- Email secondary channel (FR-80 mentions email as secondary)
- Rich Telegram messages with inline keyboards
- Notification grouping/batching
- Per-notification-type preferences
- Webhook integration for other platforms
