# Story 2.5: Notification Preferences & Throttling

## Story

As a quant trader,
I want to configure notification preferences and have non-critical notifications throttled,
So that I receive important alerts without notification spam.

## Status

not-started

## Acceptance Criteria

**Given** the notification center is running
**When** the user configures notification preferences (enable/disable per category: critical, warning, info, debug)
**Then** preferences are persisted and take effect immediately
**And** disabled categories are suppressed entirely

**Given** non-critical notifications are being generated rapidly
**When** the rate exceeds 10 per minute
**Then** non-critical notifications are throttled (max 10/min)
**And** critical notifications bypass throttling entirely
**And** notification history (last 1000) is maintained and queryable via API

## Technical Requirements

### Architecture Context

- **Service:** `notification` (Python) — INF-2
- **Database:** PostgreSQL for notification history and preferences
- **Event bus:** NATS (subscribes to `NotificationRequest` events)
- **Delivery:** Telegram (primary), email (secondary) — FR-80
- **Pattern:** Notification service subscribes to NATS, applies preferences and throttling, delivers to configured channels

### Key Components to Implement

1. **Notification Preferences Manager** (`internal/preferences/manager.go` or `services/notification/preferences/`)
   - CRUD for notification preferences per category — FR-83
   - Categories: critical, warning, info, debug
   - Per-channel enable/disable (Telegram, email)
   - Persist to PostgreSQL `notification_preferences` table
   - Immediate effect (preferences checked before each send)

2. **Throttle Manager** (`internal/throttle/manager.go` or `services/notification/throttle/`)
   - Rate limiting: max 10 non-critical notifications per minute — FR-82
   - Critical notifications bypass throttling entirely
   - Sliding window algorithm (1-minute window)
   - Track per-category counts in Redis for fast access
   - Log throttled notifications (don't drop silently)

3. **Notification History** (`internal/history/store.go` or `services/notification/history/`)
   - Store last 1000 notifications — FR-84
   - Queryable via API (filter by category, date range)
   - PostgreSQL `notification_history` table with auto-cleanup

4. **Notification Delivery** (`internal/delivery/`)
   - Telegram delivery with retry and backoff — FR-80
   - Email delivery (secondary channel)
   - Delivery confirmation tracking
   - Failed delivery retry queue

5. **API Gateway Endpoints** (`services/api-gateway/`)
   - `GET /api/notifications/preferences` — get current preferences
   - `PUT /api/notifications/preferences` — update preferences
   - `GET /api/notifications/history` — query notification history
   - `GET /api/notifications/history/{id}` — get specific notification

### Data Models

**NotificationPreferences:**
```go
type NotificationPreferences struct {
    ID              string    `json:"id"`
    Categories      CategoryConfig `json:"categories"`
    Channels        ChannelConfig  `json:"channels"`
    UpdatedAt       time.Time `json:"updated_at"`
}

type CategoryConfig struct {
    Critical bool `json:"critical"` // Always true, cannot disable
    Warning  bool `json:"warning"`
    Info     bool `json:"info"`
    Debug    bool `json:"debug"`
}

type ChannelConfig struct {
    Telegram bool   `json:"telegram"`
    Email    bool   `json:"email"`
    ChatID   string `json:"chat_id"`   // Telegram chat ID
    EmailTo  string `json:"email_to"`  // Email address
}
```

**Notification:**
```go
type Notification struct {
    ID          string    `json:"id"`
    Category    string    `json:"category"`    // critical, warning, info, debug
    Title       string    `json:"title"`
    Message     string    `json:"message"`
    Channel     string    `json:"channel"`     // telegram, email
    Status      string    `json:"status"`      // sent, failed, throttled, suppressed
    SentAt      *time.Time `json:"sent_at"`
    CreatedAt   time.Time  `json:"created_at"`
}
```

**ThrottleState:**
```go
type ThrottleState struct {
    WindowStart   time.Time
    Category      string
    Count         int
    Limit         int    // 10 for non-critical
    IsThrottled   bool
}
```

### API Endpoints

| API | Method | URL | Purpose |
|-----|--------|-----|---------|
| Get Preferences | GET | `/api/notifications/preferences` | Current preferences |
| Update Preferences | PUT | `/api/notifications/preferences` | Update preferences |
| Notification History | GET | `/api/notifications/history` | Query history (paginated) |
| Notification Detail | GET | `/api/notifications/history/{id}` | Specific notification |

### NATS Subjects

```
pqap.notification.request       # NotificationRequest event (input)
pqap.notification.sent          # NotificationSent event (output)
pqap.notification.throttled     # NotificationThrottled event (output)
```

### Prometheus Metrics

```
pqap_notification_sent_total           # Counter — notifications sent (by category, channel)
pqap_notification_throttled_total      # Counter — notifications throttled
pqap_notification_suppressed_total     # Counter — notifications suppressed (disabled category)
pqap_notification_delivery_latency_ms  # Histogram — delivery latency
pqap_notification_delivery_failures_total  # Counter — delivery failures
pqap_notification_history_size         # Gauge — current history count
```

## Implementation Guide

### Step 1: Notification Preferences

- Create PostgreSQL table:
  ```sql
  CREATE TABLE notification_preferences (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      critical BOOLEAN NOT NULL DEFAULT TRUE,  -- Cannot be disabled
      warning BOOLEAN NOT NULL DEFAULT TRUE,
      info BOOLEAN NOT NULL DEFAULT TRUE,
      debug BOOLEAN NOT NULL DEFAULT FALSE,
      telegram_enabled BOOLEAN NOT NULL DEFAULT TRUE,
      email_enabled BOOLEAN NOT NULL DEFAULT FALSE,
      telegram_chat_id VARCHAR(255),
      email_address VARCHAR(255),
      updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  );
  ```
- Implement CRUD API
- Critical category always returns `true` regardless of user setting
- Preferences checked before each notification send
- Changes take effect immediately (no restart required)

### Step 2: Throttle Manager

- Implement sliding window rate limiter using Redis
- Key pattern: `pqap:throttle:{category}:{window_start}`
- TTL: 60 seconds (auto-cleanup)
- On notification request:
  1. Check if category is enabled in preferences
  2. If category is "critical", skip throttling
  3. Check current count in window
  4. If count >= 10, mark as throttled
  5. Log throttled notification (don't drop silently)
  6. Increment counter

### Step 3: Notification History

- Create PostgreSQL table:
  ```sql
  CREATE TABLE notification_history (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      category VARCHAR(20) NOT NULL,
      title VARCHAR(255) NOT NULL,
      message TEXT NOT NULL,
      channel VARCHAR(20) NOT NULL,
      status VARCHAR(20) NOT NULL,  -- sent, failed, throttled, suppressed
      sent_at TIMESTAMPTZ,
      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  );

  CREATE INDEX idx_notification_history_created ON notification_history(created_at DESC);
  CREATE INDEX idx_notification_history_category ON notification_history(category);
  ```
- Auto-cleanup: delete notifications older than the most recent 1000
- Queryable by category, date range, status

### Step 4: Notification Delivery

- **Telegram delivery:**
  - Use Telegram Bot API
  - Format message with markdown (category emoji, title, message)
  - Retry with exponential backoff on failure (3 attempts)
  - Track delivery confirmation

- **Email delivery:**
  - SMTP or transactional email service
  - HTML template with category color coding
  - Retry on failure

### Step 5: Processing Pipeline

```
NotificationRequest (NATS)
    ↓
[Preferences Check] → Suppressed? → Log + Store → END
    ↓
[Throttle Check] → Throttled? → Log + Store → END
    ↓
[Delivery] → Telegram + Email (parallel)
    ↓
[Store History] → PostgreSQL
    ↓
[Publish Result] → NATS
```

### Step 6: API Gateway Endpoints

- Implement in FastAPI
- JWT authentication required (AD-14)
- Preferences: GET/PUT with validation
- History: paginated query with filters (category, date range, status)

## Testing

### Unit Tests

- **Preferences manager:** CRUD, critical always enabled, immediate effect
- **Throttle manager:** Rate limiting, sliding window, critical bypass
- **Notification history:** Storage, query, auto-cleanup
- **Delivery:** Telegram send, email send, retry logic

### Integration Tests

- **Full pipeline:** Request → preferences → throttle → delivery → history
- **Throttle behavior:** 10+ notifications in 1 minute → first 10 sent, rest throttled
- **Critical bypass:** Critical notifications never throttled
- **Preferences suppression:** Disabled category → notification suppressed

### Test Files

```
tests/unit/notification/
├── preferences_test.py
├── throttle_test.py
├── history_test.py
└── delivery_test.py

tests/integration/
├── notification_pipeline_test.py
├── notification_throttle_test.py
└── notification_preferences_test.py
```

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| `python-telegram-bot` | latest | Telegram Bot API |
| `aiosmtplib` | latest | Async SMTP for email |
| `redis` | 8.8.0 | Throttle state (sliding window) |
| `psycopg2` or `asyncpg` | latest | PostgreSQL driver |
| `fastapi` | 0.139.0 | API Gateway endpoints |

## External Dependencies (Infrastructure)

| Service | Required | Purpose |
|---------|----------|---------|
| Telegram Bot API | Yes | Primary notification delivery |
| SMTP Server | Optional | Secondary email delivery |
| Redis | Yes | Throttle state (sliding window) |
| PostgreSQL | Yes | Preferences, notification history |
| NATS | Yes | NotificationRequest events |

## Definition of Done

- [ ] Notification preferences CRUD with immediate effect
- [ ] Critical notifications cannot be disabled
- [ ] Non-critical throttling at max 10/minute
- [ ] Critical notifications bypass throttling entirely
- [ ] Notification history maintained (last 1000) and queryable
- [ ] Telegram delivery with retry on failure
- [ ] Throttled notifications logged (not silently dropped)
- [ ] Prometheus metrics exported (`pqap_notification_*`)
- [ ] Unit tests pass with ≥80% coverage
- [ ] Integration tests pass
- [ ] API endpoints authenticated via JWT (AD-14)

## References

| Reference | Description |
|-----------|-------------|
| FR-80 | Center SHALL send notifications via Telegram (primary) and email (secondary) |
| FR-81 | Center SHALL categorize notifications: critical, warning, info, debug |
| FR-82 | Center SHALL support notification throttling (max 10 per minute for non-critical) |
| FR-83 | Center SHALL support configurable notification preferences (enable/disable per category) |
| FR-84 | Center SHALL maintain notification history (last 1000) |
| AD-14 | JWT auth on Dashboard/Admin Panel |
| NFR-N1 | Critical notification latency within 5s |
| NFR-N2 | Critical notification delivery rate 99.9% |
| NFR-N3 | Non-critical throttling max 10/min |
| INF-2 | Python 3.13.14 for notification service |
| INF-8 | Redis 8.8.0 for cache/coordination |
