# Story 2.1: Advanced Risk — Correlation Limits, Batasi Win & Metabolic Rate

## Story

As a quant trader,
I want correlation limits, win-streak breaker (Batasi Win), and system resource monitoring,
So that the bot avoids cascade risk from correlated positions, prevents overconfidence during win streaks, and stays within safe resource consumption.

## Status

ready-for-dev

## Acceptance Criteria

**Given** the risk manager is tracking open positions
**When** a new trade would result in more than the configurable maximum of correlated positions (default: 3)
**Then** the Pit Boss returns DENY with reason "correlation_limit_exceeded"
**And** the rejection is logged with the list of correlated positions

**Given** the risk manager is tracking consecutive wins
**When** the win streak reaches the configurable threshold (default: 5)
**Then** trading is paused (Batasi Win)
**And** a warning notification is sent
**And** trading resumes only after manual approval or configurable cooldown period

**Given** the risk manager is monitoring system resources
**When** CPU exceeds 80%, memory exceeds 1GB, or goroutine count exceeds threshold
**Then** a metabolic rate alert is published to NATS
**And** metrics are exported via Prometheus endpoint (`pqap_risk_*`)

## Technical Requirements

### Architecture Context

- **Service:** `risk-manager` (Go)
- **Paradigm:** Event-driven with Redis-based state coordination
- **State store:** Redis (Pit Boss state keys with 60s TTL, refreshed every 30s per AD-4)
- **Database:** PostgreSQL `risk_events` table (Risk Management is sole writer per AD-6)
- **Event bus:** NATS for metabolic rate alerts
- **Pattern:** Pit Boss is the **sole authority** on whether a trade may proceed (AD-4). All risk checks are synchronous Redis GETs from the execution engine.

### Key Components to Implement

1. **Correlation Engine** (`internal/risk/correlation_engine.go`)
   - Auto-detect correlation groups using 3-layer approach:
     - **Layer 1: Category-Based** — Polymarket market categories auto-group related markets
     - **Layer 2: Price Correlation** — Track price movements, compute Pearson correlation, group markets with correlation > 0.7
     - **Layer 3: Keyword-Based** — Auto-detect from market slug/title keywords (e.g., "btc-100k", "btc-ath" → group "btc-related")
   - Update correlation matrix every 1 hour
   - Store groups in PostgreSQL `correlation_groups` table
   - Reconstructable from price history on restart

2. **Correlation Tracker** (`internal/risk/correlation.go`)
   - Read auto-detected groups from Correlation Engine
   - Detect when new trade would exceed max correlated positions (default: 3) — FR-40
   - Return DENY with `correlation_limit_exceeded` reason
   - Log rejection with full correlated position list
   - Manual override: user can adjust groups via API (optional)

2. **Batasi Win (Win Streak Breaker)** (`internal/risk/batasi_win.go`)
   - Track consecutive win count from trade history
   - Pause trading when win streak reaches threshold (default: 5) — FR-41
   - Send warning notification via NATS
   - Support manual resume or configurable cooldown period
   - Persist win streak state in Redis for cross-component visibility

3. **Metabolic Rate Monitor** (`internal/risk/metabolic.go`)
   - Monitor CPU usage (threshold: 80%)
   - Monitor memory usage (threshold: 1GB)
   - Monitor goroutine count (configurable threshold) — FR-43
   - Publish `MetabolicRateAlert` events to NATS when thresholds exceeded
   - Export metrics via Prometheus endpoint

4. **Pit Boss State Writer** (`internal/risk/pitboss.go`)
   - Extend existing Pit Boss state in Redis to include:
     - Correlation limit status per market group
     - Batasi Win pause state
     - Metabolic rate status
   - Refresh state every 30s with 60s TTL (AD-4)
   - State reconstructable from PostgreSQL on restart (AD-8)

### Data Models

**CorrelationEngine:**
```go
type CorrelationEngine struct {
    priceHistory      map[string][]float64           // market_id → price history (last 24h)
    correlationMatrix map[string]map[string]float64  // market_id → market_id → correlation
    categoryGroups    map[string][]string            // category → market_ids
    keywordGroups     map[string][]string            // keyword → market_ids
    threshold         float64                        // correlation threshold (default: 0.7)
    updateInterval    time.Duration                  // matrix update interval (default: 1h)
}

type CorrelationGroup struct {
    ID              string    `json:"id"`              // Auto-generated group ID
    Name            string    `json:"name"`            // Human-readable name
    DetectionMethod string    `json:"detection_method"` // "category", "correlation", "keyword"
    MarketIDs       []string  `json:"market_ids"`
    MaxPositions    int       `json:"max_positions"`   // Configurable limit (default: 3)
    Confidence      float64   `json:"confidence"`      // 0.0-1.0 (higher = more confident)
    LastUpdated     time.Time `json:"last_updated"`
}
```

**CorrelationState:**
```go
type CorrelationState struct {
    MarketGroup    string          // Group identifier (e.g., "election-2026")
    MarketIDs      []string        // Markets in this correlation group
    OpenPositions  int             // Current open positions in group
    MaxPositions   int             // Configurable limit (default: 3)
    IsExceeded     bool
}

type CorrelationRejection struct {
    MarketID        string
    CorrelatedWith  []string        // List of correlated open positions
    Reason          string          // "correlation_limit_exceeded"
    Timestamp       time.Time
}
```

**BatasiWinState:**
```go
type BatasiWinState struct {
    CurrentStreak    int           // Current consecutive wins
    Threshold        int           // Configurable (default: 5)
    IsPaused         bool
    PausedAt         *time.Time
    CooldownMinutes  int           // Configurable cooldown (0 = manual resume only)
    ResumeAfter      *time.Time
}
```

**MetabolicRateMetrics:**
```go
type MetabolicRateMetrics struct {
    CPUPercent     float64
    MemoryBytes    uint64
    GoroutineCount int
    Timestamp      time.Time
    IsAlert        bool
}
```

**Events:**
```go
type MetabolicRateAlert struct {
    EventID   string            `json:"event_id"`
    EventType string            `json:"event_type"` // "MetabolicRateAlert"
    Timestamp time.Time         `json:"timestamp"`
    Source    string            `json:"source"`      // "risk-manager"
    Payload   MetabolicRateMetrics `json:"payload"`
}

type BatasiWinTriggered struct {
    EventID      string    `json:"event_id"`
    EventType    string    `json:"event_type"` // "BatasiWinTriggered"
    Timestamp    time.Time `json:"timestamp"`
    Source       string    `json:"source"`
    Payload      BatasiWinState `json:"payload"`
}
```

### NATS Subject Hierarchy

```
pqap.risk.correlation.rejected    # CorrelationRejection
pqap.risk.batasi.triggered        # BatasiWinTriggered
pqap.risk.metabolic.alert         # MetabolicRateAlert
```

### Prometheus Metrics (AD-17)

```
pqap_risk_correlated_positions_total      # Gauge — current correlated position count per group
pqap_risk_correlation_rejections_total    # Counter — correlation limit rejections
pqap_risk_batasi_win_streak_current       # Gauge — current win streak count
pqap_risk_batasi_win_pauses_total         # Counter — Batasi Win pause events
pqap_risk_metabolic_cpu_percent           # Gauge — current CPU usage
pqap_risk_metabolic_memory_bytes          # Gauge — current memory usage
pqap_risk_metabolic_goroutines_total      # Gauge — current goroutine count
pqap_risk_metabolic_alerts_total          # Counter — metabolic rate alerts
```

## Implementation Guide

### Step 1: Configuration

Add to risk manager configuration:
```go
type RiskConfig struct {
    // Correlation limits
    MaxCorrelatedPositions int    `env:"RISK_MAX_CORRELATED_POSITIONS" default:"3"`

    // Batasi Win
    BatasiWinThreshold     int    `env:"RISK_BATASI_WIN_THRESHOLD" default:"5"`
    BatasiWinCooldownMin   int    `env:"RISK_BATASI_WIN_COOLDOWN_MIN" default:"0"` // 0 = manual only

    // Metabolic rate
    MetabolicCPUPercent    float64 `env:"RISK_METABOLIC_CPU_PERCENT" default:"80"`
    MetabolicMemoryBytes   uint64  `env:"RISK_METABOLIC_MEMORY_BYTES" default:"1073741824"` // 1GB
    MetabolicGoroutines    int     `env:"RISK_METABOLIC_GOROUTINES" default:"10000"`
}
```

### Step 2: Correlation Engine (Auto-Detect)

- Implement `CorrelationEngine` with 3-layer detection:
  - **Layer 1: Category-Based** — Read market categories from Polymarket API, auto-group related markets
  - **Layer 2: Price Correlation** — Track price history (last 24h), compute Pearson correlation every 1h, group markets with correlation > 0.7
  - **Layer 3: Keyword-Based** — Parse market slugs for keywords (btc, eth, trump, fed, etc.), auto-group related markets
- Store detected groups in PostgreSQL `correlation_groups` table
- Update groups every 1 hour
- Expose groups via API for user review (optional manual override)

### Step 3: Correlation Tracker

- Read auto-detected groups from Correlation Engine
- On every Pit Boss state refresh (30s), evaluate all open positions against correlation limits
- When execution engine calls Pit Boss check, include correlation validation
- Return DENY with `correlation_limit_exceeded` if limit would be breached
- Log rejection to PostgreSQL `risk_events` table

### Step 4: Batasi Win

- Implement `BatasiWinMonitor` that reads trade history from PostgreSQL
- Count consecutive wins (trades where PnL > 0)
- On threshold reached:
  - Set `batasi_win_paused` flag in Redis Pit Boss state
  - Publish `BatasiWinTriggered` event to NATS
  - Trigger warning notification
- Support resume via:
  - Manual API call (user triggers resume)
  - Cooldown timer (if configured, auto-resume after N minutes)
- On resume, reset win streak counter

### Step 5: Metabolic Rate Monitor

- Implement `MetabolicMonitor` goroutine that runs every 10s
- Use `runtime.NumGoroutine()` for goroutine count
- Use `github.com/shirou/gopsutil/v3` for CPU and memory monitoring
- On threshold breach:
  - Publish `MetabolicRateAlert` event to NATS
  - Update Prometheus gauges
  - Log warning (structured JSON per INF-14)
- Export all metrics via Prometheus endpoint

### Step 6: Pit Boss State Extension

- Extend Pit Boss Redis state to include:
  - `correlation_exceeded` boolean per market group
  - `batasi_win_paused` boolean
  - `metabolic_alert` boolean
- State refreshed every 30s with 60s TTL (existing pattern from AD-4)
- Execution engine reads these fields during synchronous risk check

### Step 7: NATS Event Publishing

- Publish events to defined subjects
- All events include: `event_id` (UUID), `event_type`, `timestamp` (ISO 8601 UTC), `source`, `payload` (INF-17)
- Use JetStream for durable delivery

## Testing

### Unit Tests

- **Correlation tracker:** Limit enforcement, group calculation, rejection logging
- **Batasi Win:** Streak counting, pause trigger, cooldown resume, manual resume
- **Metabolic monitor:** Threshold detection, alert publishing, metric export
- **Pit Boss state:** State assembly, TTL refresh, reconstruction from PostgreSQL

### Integration Tests

- **Correlation → Pit Boss:** Verify DENY returned when correlation limit exceeded
- **Batasi Win → Notification:** Verify warning notification sent on pause
- **Metabolic → NATS:** Verify alert published on threshold breach
- **Pit Boss → Execution Engine:** Verify execution engine reads correlation/batasi state

### Test Files

```
tests/unit/risk/
├── correlation_test.go
├── batasi_win_test.go
├── metabolic_monitor_test.go
└── pitboss_state_test.go

tests/integration/
├── risk_correlation_pitboss_test.go
├── risk_batasi_notification_test.go
└── risk_metabolic_nats_test.go
```

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| `github.com/shirou/gopsutil/v3` | latest | CPU and memory monitoring |
| `github.com/nats-io/nats.go` | latest | NATS event publishing |
| `github.com/redis/go-redis/v9` | latest | Pit Boss state in Redis |
| `github.com/prometheus/client_golang` | latest | Metrics export |
| `go.uber.org/zap` | latest | Structured logging |
| `github.com/google/uuid` | latest | Event ID generation |

## External Dependencies (Infrastructure)

| Service | Required | Purpose |
|---------|----------|---------|
| Redis | Yes | Pit Boss state (correlation, batasi, metabolic) |
| PostgreSQL | Yes | risk_events table, trade history for win streak |
| NATS | Yes | Event publishing (alerts, batasi triggers) |

## Definition of Done

- [ ] Correlation Engine auto-detects groups via 3-layer approach (category, price correlation, keyword)
- [ ] Correlation matrix updates every 1 hour
- [ ] Correlation groups stored in PostgreSQL
- [ ] Correlation limits enforced — Pit Boss returns DENY when max correlated positions exceeded
- [ ] Correlation rejections logged to PostgreSQL with full context
- [ ] Batasi Win pauses trading after configurable consecutive wins (default: 5)
- [ ] Warning notification sent on Batasi Win pause
- [ ] Trading resumes via manual approval or configurable cooldown
- [ ] Metabolic rate monitor tracks CPU, memory, goroutine count
- [ ] Metabolic alerts published to NATS on threshold breach
- [ ] All metrics exported via Prometheus (`pqap_risk_*`)
- [ ] Pit Boss state includes correlation, batasi, and metabolic fields
- [ ] Pit Boss state refreshes every 30s with 60s TTL
- [ ] Unit tests pass with ≥80% coverage
- [ ] Integration tests pass
- [ ] Structured JSON logging implemented (INF-14)

## References

| Reference | Description |
|-----------|-------------|
| FR-40 | System SHALL enforce max correlated positions (default: 3, configurable) |
| FR-41 | System SHALL implement Batasi Win (win streak breaker): pause after N consecutive wins (default: 5) |
| FR-43 | System SHALL implement metabolic rate monitor: track CPU, memory, goroutine counts |
| AD-4 | Pit Boss is sole authority on trade approval; lives as Redis keys with 60s TTL |
| AD-6 | PostgreSQL single-writer: risk_events table owned by Risk Management |
| AD-8 | Redis is ephemeral cache; reconstructable from PostgreSQL |
| AD-17 | Prometheus metrics on /metrics for all services |
| INF-14 | Structured JSON logs with timestamp, level, service, request_id, message, context |
| INF-15 | Prometheus metric naming: pqap_{service}_{metric_name}_{unit} |
| INF-17 | All events include: event_id, event_type, timestamp, source, payload |
