# Story 2.6: Admin Authentication & Scanner Metrics Export

## Story

As a quant trader,
I want the admin panel and dashboard to require authentication, and the scanner to export Prometheus metrics,
So that my system is secure and I can monitor scanner performance.

## Status

not-started

## Acceptance Criteria

**Given** the dashboard or admin panel is accessed
**When** no valid JWT session exists
**Then** the user is redirected to login
**And** session timeout is configurable
**And** CSRF protection is active on all state-changing endpoints

**Given** the scanner service is running
**When** Prometheus scrapes the `/metrics` endpoint
**Then** the following metrics are exported: markets tracked, price update latency, WebSocket connection status, stale market count
**And** all metric names follow the convention `pqap_scanner_{metric_name}_{unit}`
**And** values are accurate within 1 second

## Technical Requirements

### Architecture Context

- **Authentication:** JWT-based (AD-14) — secrets in Kubernetes Secrets, injected as env vars
- **Dashboard:** Next.js 16.2.10 (LTS) — INF-4
- **Admin Panel:** Part of dashboard or separate Next.js app
- **Scanner:** Go service — metrics via Prometheus client library
- **Monitoring:** Prometheus 3.12.0 scrapes `/metrics` endpoints — INF-9
- **Pattern:** All state-changing endpoints require valid JWT + CSRF token. Read-only endpoints require JWT only. Scanner exposes Prometheus metrics on `:9090/metrics`.

### Key Components to Implement

1. **Authentication Middleware** (`services/api-gateway/auth/`)
   - JWT validation middleware for all API routes — FR-116
   - CSRF token generation and validation — AD-14
   - Session management with configurable timeout
   - Login/logout endpoints
   - Password hashing (bcrypt)

2. **Login Page** (`dashboard/src/app/login/page.tsx`)
   - Username/password form
   - JWT token storage (httpOnly cookie)
   - Redirect to dashboard on success
   - Error handling for invalid credentials

3. **Auth Guard** (`dashboard/src/lib/auth/`)
   - Client-side auth check on page load
   - Redirect to login if no valid session
   - Token refresh before expiry
   - CSRF token inclusion in state-changing requests

4. **Scanner Prometheus Metrics** (`services/scanner/metrics/`)
   - Extend existing metrics from Story 1.1 — FR-8
   - Ensure all metrics follow `pqap_scanner_{metric_name}_{unit}` convention (INF-15)
   - Metrics endpoint on `:9090/metrics`
   - Accurate within 1 second

5. **Admin Panel Authentication** (`dashboard/src/app/admin/`)
   - Admin routes protected by JWT + admin role check
   - Separate admin login or role-based access
   - CSRF protection on all admin forms

### Data Models

**User:**
```go
type User struct {
    ID           string    `json:"id"`
    Username     string    `json:"username"`
    PasswordHash string    `json:"-"` // Never serialized
    Role         string    `json:"role"` // "admin", "viewer"
    CreatedAt    time.Time `json:"created_at"`
    LastLogin    time.Time `json:"last_login"`
}
```

**JWT Claims:**
```go
type JWTClaims struct {
    UserID   string `json:"user_id"`
    Username string `json:"username"`
    Role     string `json:"role"`
    jwt.RegisteredClaims
}
```

**CSRF Token:**
```go
type CSRFToken struct {
    Token     string
    ExpiresAt time.Time
}
```

**Scanner Metrics (from Story 1.1, extended):**
```go
// All metrics follow pqap_scanner_{metric_name}_{unit} convention (INF-15)
var (
    MarketsTracked = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "pqap_scanner_markets_tracked_total",
        Help: "Number of active markets being tracked",
    })
    UpdateLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name:    "pqap_scanner_update_latency_ms",
        Help:    "Price update processing latency in milliseconds",
        Buckets: []float64{10, 25, 50, 100, 250, 500, 1000},
    })
    WSConnectionStatus = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "pqap_scanner_ws_connection_status",
        Help: "WebSocket connection status (1=connected, 0=disconnected)",
    })
    StaleMarkets = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "pqap_scanner_stale_markets_total",
        Help: "Number of markets flagged as stale",
    })
    ReconnectCount = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "pqap_scanner_ws_reconnect_total",
        Help: "Total WebSocket reconnection attempts",
    })
    EventsPublished = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "pqap_scanner_events_published_total",
        Help: "Total NATS events published",
    })
)
```

### API Endpoints

| API | Method | URL | Purpose |
|-----|--------|-----|---------|
| Login | POST | `/api/auth/login` | Authenticate and receive JWT |
| Logout | POST | `/api/auth/logout` | Invalidate session |
| Refresh | POST | `/api/auth/refresh` | Refresh JWT token |
| CSRF Token | GET | `/api/auth/csrf` | Get CSRF token |
| Scanner Metrics | GET | `:9090/metrics` | Prometheus scrape endpoint |

### Security Configuration

```yaml
# Environment variables
AUTH_JWT_SECRET: <kubernetes-secret>      # JWT signing key
AUTH_JWT_EXPIRY: "24h"                    # Configurable session timeout
AUTH_CSRF_ENABLED: "true"                 # CSRF protection toggle
AUTH_BCRYPT_COST: "12"                    # Password hashing cost
SCANNER_METRICS_PORT: "9090"              # Prometheus metrics port
```

### Prometheus Metrics (Scanner - Complete Set)

```
# Market metrics
pqap_scanner_markets_tracked_total       # Gauge — active markets count
pqap_scanner_stale_markets_total         # Gauge — stale market count

# Performance metrics
pqap_scanner_update_latency_ms           # Histogram — price update processing latency

# Connection metrics
pqap_scanner_ws_connection_status        # Gauge — 1=connected, 0=disconnected
pqap_scanner_ws_reconnect_total          # Counter — reconnection attempts

# Event metrics
pqap_scanner_events_published_total      # Counter — NATS events published

# REST API metrics
pqap_scanner_rest_requests_total         # Counter — REST API requests
pqap_scanner_rest_errors_total           # Counter — REST API errors
pqap_scanner_rest_latency_ms             # Histogram — REST API latency
```

## Implementation Guide

### Step 1: Authentication Backend

- Create PostgreSQL table:
  ```sql
  CREATE TABLE users (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      username VARCHAR(100) UNIQUE NOT NULL,
      password_hash VARCHAR(255) NOT NULL,
      role VARCHAR(20) NOT NULL DEFAULT 'viewer',
      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      last_login TIMESTAMPTZ
  );
  ```
- Implement JWT middleware in FastAPI:
  - Validate JWT on every request
  - Extract user claims (ID, username, role)
  - Reject expired/invalid tokens with 401
- Implement CSRF middleware:
  - Generate CSRF token per session
  - Validate on POST/PUT/DELETE requests
  - Reject invalid tokens with 403
- Login endpoint:
  - Validate username/password (bcrypt)
  - Generate JWT with configurable expiry
  - Set httpOnly cookie
  - Update last_login timestamp

### Step 2: Login Page

- Create Next.js login page at `/login`
- Username/password form with validation
- Submit to `POST /api/auth/login`
- On success: store JWT in httpOnly cookie, redirect to dashboard
- On failure: display error message
- Responsive design (mobile-friendly login)

### Step 3: Client-side Auth Guard

- Implement auth check in Next.js middleware or layout component
- On every page load:
  1. Check for valid JWT cookie
  2. If expired, attempt refresh via `POST /api/auth/refresh`
  3. If refresh fails, redirect to `/login`
- Include CSRF token in all state-changing requests (fetch interceptor)
- Token refresh before expiry (e.g., refresh at 80% of TTL)

### Step 4: Scanner Metrics Export

- Extend existing Prometheus metrics from Story 1.1
- Ensure all metric names follow `pqap_scanner_{metric_name}_{unit}` (INF-15)
- Register all metrics with Prometheus client library
- Expose `/metrics` endpoint on port 9090
- Update metrics in real-time as scanner operates:
  - `markets_tracked_total` — updated on catalog changes
  - `update_latency_ms` — observed on every price update
  - `ws_connection_status` — updated on connect/disconnect
  - `stale_markets_total` — updated on stale detection
  - `reconnect_total` — incremented on each reconnect attempt
  - `events_published_total` — incremented on each NATS publish

### Step 5: Admin Panel Protection

- Admin routes at `/admin/*`
- Require JWT with `role: "admin"` claim
- CSRF protection on all admin forms
- Admin-specific features:
  - User management (if multi-user)
  - System configuration
  - Log viewer

### Step 6: Security Hardening

- JWT secret stored in Kubernetes Secrets (AD-14)
- Injected as environment variable, never mounted as file
- Never logged (AD-14)
- Password hashing with bcrypt (cost 12)
- Rate limiting on login endpoint (5 attempts per minute)
- Session timeout configurable via `AUTH_JWT_EXPIRY`

## Testing

### Unit Tests

- **JWT middleware:** Valid token, expired token, invalid token, missing token
- **CSRF middleware:** Valid CSRF, missing CSRF, expired CSRF
- **Login endpoint:** Valid credentials, invalid credentials, rate limiting
- **Scanner metrics:** Correct metric values, naming convention compliance

### Integration Tests

- **Auth flow:** Login → access protected page → logout → redirect to login
- **CSRF flow:** Form submission with/without CSRF token
- **Metrics scrape:** Prometheus scrapes scanner `/metrics` endpoint successfully
- **Session timeout:** Token expires after configured duration

### Security Tests

- **JWT tampering:** Modified JWT rejected
- **CSRF attack:** Request without valid CSRF token rejected
- **Brute force:** Rate limiting blocks rapid login attempts
- **Secret exposure:** JWT secret not in logs, not in code, not in responses

### Test Files

```
tests/unit/auth/
├── jwt_middleware_test.py
├── csrf_middleware_test.py
├── login_endpoint_test.py
└── password_hash_test.py

tests/unit/scanner/
└── metrics_export_test.go

tests/integration/
├── auth_flow_test.py
├── csrf_protection_test.py
└── scanner_metrics_scrape_test.py

tests/security/
├── jwt_tampering_test.py
├── csrf_attack_test.py
└── brute_force_test.py
```

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| `python-jose[cryptography]` | latest | JWT encoding/decoding |
| `passlib[bcrypt]` | latest | Password hashing |
| `fastapi` | 0.139.0 | API Gateway with middleware |
| `prometheus-client` | latest (Go) | Scanner metrics export |
| `next` | 16.2.10 | Dashboard with auth |

## External Dependencies (Infrastructure)

| Service | Required | Purpose |
|---------|----------|---------|
| PostgreSQL | Yes | Users table |
| Prometheus | Yes | Metrics scraping |
| Kubernetes Secrets | Yes | JWT secret storage |

## Definition of Done

- [ ] JWT authentication enforced on all dashboard and admin panel routes
- [ ] Login page with username/password form
- [ ] Redirect to login when no valid JWT session exists
- [ ] Session timeout configurable via environment variable
- [ ] CSRF protection active on all state-changing endpoints
- [ ] Scanner exports all metrics on `/metrics` endpoint
- [ ] All metric names follow `pqap_scanner_{metric_name}_{unit}` convention
- [ ] Metrics accurate within 1 second
- [ ] JWT secret stored in Kubernetes Secrets, never logged
- [ ] Password hashing with bcrypt
- [ ] Rate limiting on login endpoint
- [ ] Unit tests pass with ≥80% coverage
- [ ] Integration tests pass
- [ ] Security tests pass (JWT tampering, CSRF, brute force)

## References

| Reference | Description |
|-----------|-------------|
| FR-8 | Scanner SHALL export metrics: markets tracked, update latency, connection status, stale count |
| FR-116 | Admin Panel SHALL require authentication (even for single user) |
| AD-14 | Secrets in Kubernetes Secrets; JWT auth on Dashboard/Admin Panel |
| AD-17 | Prometheus metrics on /metrics for all services |
| NFR-AP1 | Authentication security: session timeout, CSRF protection |
| INF-4 | Next.js 16.2.10 (LTS) for Dashboard |
| INF-3 | FastAPI 0.139.0 for API Gateway |
| INF-9 | Prometheus 3.12.0 for metrics |
| INF-15 | Prometheus metric naming: pqap_{service}_{metric_name}_{unit} |
