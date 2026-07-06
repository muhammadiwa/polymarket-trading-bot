# Story 4.3: Analytics — Anomaly Detection

Status: ready-for-dev

## Story

As a quant trader,
I want the analytics service to detect performance anomalies automatically,
So that I'm alerted when something unusual happens with my trading performance.

## Acceptance Criteria

1. **Given** the analytics service is continuously monitoring performance metrics
   **When** an anomaly is detected (sudden drop in win rate, unusual drawdown pattern)
   **Then** the anomaly is detected within 1 day of occurrence
   **And** an alert is sent via the existing notification center (NATS → notification service)
   **And** the anomaly is logged with context (metric, threshold, actual value, timestamp)

## Tasks / Subtasks

- [ ] Task 1: Anomaly Detection Logic (AC: #1)
  - [ ] Add `detect_anomalies()` to existing `analytics_repo.py`
  - [ ] Reuse `calculate_performance_metrics()` for current vs baseline comparison
  - [ ] Reuse `calculate_risk_metrics()` for drawdown anomaly detection
  - [ ] 5 detection rules (see below)
- [ ] Task 2: Config & DB (AC: #1)
  - [ ] Add anomaly thresholds to existing `config.py`
  - [ ] Create `anomaly_events` table migration (014)
  - [ ] Add anomaly logging to existing `analytics_repo.py`
- [ ] Task 3: API & Alert (AC: #1)
  - [ ] Add GET /api/analytics/anomalies to existing `routes/analytics.py`
  - [ ] Publish `AnomalyDetected` event to NATS (existing pattern)
  - [ ] Trigger notification via existing notification service (Epic 2)
- [ ] Task 4: Scheduled Monitoring
  - [ ] Add background task to existing `main.py` lifespan
  - [ ] Runs every 1 hour
  - [ ] Configurable via existing `config.py`

## Dev Notes

### Architecture Context

- **Service:** `analytics` (Python/FastAPI) — extends existing service from Story 4.1
- **Database:** PostgreSQL — extend existing connection from `db.py`
- **Event bus:** NATS — use existing notification pattern from `api-gateway/app/events.py`
- **Notification:** Reuse existing notification service (Epic 2, `services/notification/`)
- **Config:** Extend existing `services/analytics/app/config.py`

### Files to MODIFY (not create new)

**`services/analytics/app/repos/analytics_repo.py`**
- Current: Has `calculate_performance_metrics()`, `calculate_risk_metrics()`, `get_trades_in_range()`
- Change: Add `detect_anomalies()`, `log_anomaly()`, `get_anomalies()`
- Preserve: All existing calculation functions

**`services/analytics/app/routes/analytics.py`**
- Current: Has /pnl, /metrics, /risk, /summary, /histogram, /export endpoints
- Change: Add GET /api/analytics/anomalies endpoint
- Preserve: All existing endpoints

**`services/analytics/app/config.py`**
- Current: Has POSTGRES_URL, JWT_SECRET, SHARPE_RISK_FREE_RATE
- Change: Add anomaly threshold configs
- Preserve: All existing config

**`services/analytics/app/main.py`**
- Current: FastAPI app with lifespan
- Change: Add background anomaly check task in lifespan
- Preserve: Existing app setup

**`services/analytics/app/models/analytics.py`**
- Current: PnLResponse, PerformanceMetrics, RiskMetrics
- Change: Add AnomalyEvent model
- Preserve: All existing models

### Anomaly Detection Rules (reuse existing calculations)

| Rule | How to Detect | Reuse |
|------|---------------|-------|
| Win Rate Drop | Current win_rate < 7-day avg - 0.20 | `calculate_performance_metrics()` |
| Unusual Drawdown | current_drawdown > 2 * 7-day avg drawdown | `calculate_risk_metrics()` |
| Consecutive Losses | > 5 consecutive losses in recent trades | `get_trades_in_range()` |
| Profit Factor Drop | profit_factor < 0.5 (was > 1.5) | `calculate_performance_metrics()` |
| Sharpe Drop | sharpe_ratio < 0 (was > 1.0) | `calculate_performance_metrics()` |

### Config Additions (config.py)

```python
# Anomaly detection thresholds
ANOMALY_WIN_RATE_DROP: float = float(os.getenv("ANOMALY_WIN_RATE_DROP", "0.20"))
ANOMALY_DRAWDOWN_MULTIPLIER: float = float(os.getenv("ANOMALY_DRAWDOWN_MULTIPLIER", "2.0"))
ANOMALY_CONSECUTIVE_LOSSES: int = int(os.getenv("ANOMALY_CONSECUTIVE_LOSSES", "5"))
ANOMALY_PROFIT_FACTOR_LOW: float = float(os.getenv("ANOMALY_PROFIT_FACTOR_LOW", "0.5"))
ANOMALY_SHARPE_LOW: float = float(os.getenv("ANOMALY_SHARPE_LOW", "0"))
ANOMALY_CHECK_INTERVAL: int = int(os.getenv("ANOMALY_CHECK_INTERVAL_SECONDS", "3600"))
```

### Database Schema (migration 014)

```sql
CREATE TABLE anomaly_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    anomaly_type VARCHAR(50) NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    threshold_value DECIMAL(20,8) NOT NULL,
    actual_value DECIMAL(20,8) NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    confidence DECIMAL(5,4) NOT NULL DEFAULT 0.9,
    context JSONB DEFAULT '{}',
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_anomaly_events_type ON anomaly_events(anomaly_type, detected_at DESC);
CREATE INDEX idx_anomaly_events_severity ON anomaly_events(severity, detected_at DESC);
```

### NATS Event (reuse existing pattern)

```python
# Same pattern as events.py in api-gateway
event = {
    "event_id": str(uuid4()),
    "event_type": "AnomalyDetected",
    "timestamp": datetime.now(timezone.utc).isoformat(),
    "source": "analytics",
    "payload": {
        "anomaly_type": "win_rate_drop",
        "metric_name": "win_rate",
        "threshold_value": "0.20",
        "actual_value": "0.05",
        "severity": "high",
        "context": {"baseline": "0.25", "current": "0.05"},
    },
}
```

### Prometheus Metrics

```
pqap_anomalies_detected_total    # Counter — anomalies detected
pqap_anomaly_check_latency_ms   # Histogram — check latency
pqap_anomaly_alerts_sent_total   # Counter — alerts sent
```

### References

| Reference | Description |
|-----------|-------------|
| FR-61 | Detect performance anomalies (sudden drop in win rate, unusual drawdown) |
| INF-14 | Structured JSON logs |
| INF-17 | Event structure: event_id, event_type, timestamp, source, payload |

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
