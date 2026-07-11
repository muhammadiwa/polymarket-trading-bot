# LAPORAN CODE REVIEW MENYELURUH — PQAP

**Proyek**: Polymarket Quant Arbitrage Platform (PQAP)
**Scope**: Seluruh kodebase (15 microservices + frontend dashboard)
**Reviewer**: 5 Parallel Adversarial Subagents
**Tanggal**: 2026-07-11

---

## RINGKASAN EKSEKUTIF

| Layer | Critical | High | Medium | Low | Total |
|-------|----------|------|--------|-----|-------|
| Blind Hunter (API Gateway) | 3 | 5 | 7 | 5 | **20** |
| Edge Case Hunter (AI Services) | 1 | 4 | 8 | 5 | **18** |
| Architecture Reviewer (Business Services) | 3 | 4 | 6 | 2 | **15** |
| Frontend Reviewer (Dashboard) | 1 | 5 | 14 | 17 | **37** |
| API Contract Auditor (Backend↔Frontend) | 3 | 7 | 4 | 9 | **23** |
| **TOTAL** | **11** | **25** | **39** | **38** | **113** |

**Setelah deduplikasi**: ~85 unique findings

---

## 🔴 CRITICAL ISSUES (11) — Showstoppers, harus diperbaiki segera

### C1. Missing `verify_jwt` function — API Gateway crash on startup
- **File**: `services/api-gateway/app/middleware/auth.py`
- **Detail**: 8 route modules import `verify_jwt` tapi fungsi ini tidak pernah didefinisikan. Hanya ada `extract_user`, `decode_jwt`, `validate_jwt_claims`.
- **Impact**: `ImportError` saat startup — seluruh API Gateway tidak bisa jalan.
- **Fix**: Tambahkan `async def verify_jwt(request: Request) -> dict: return extract_user(request)`

### C2. WS Auth Protocol Mismatch — Dashboard WS tidak pernah connect
- **Backend**: `services/api-gateway/app/routes/ws.py:47-64` — membaca token dari query param/header
- **Frontend**: `services/dashboard/src/lib/websocket.ts:55-68` — mengirim token sebagai message frame pertama
- **Impact**: Semua WebSocket connection ditutup dengan `WS_1008_POLICY_VIOLATION`. Data real-time (risk, opportunities, health) tidak mengalir.
- **Fix**: Ubah frontend untuk pass token sebagai query param: `new WebSocket(\`${WS_BASE}/ws/dashboard?token=${token}\`)`

### C3. Undefined `_sessions` dict — 4 replay routes crash
- **File**: `services/backtesting/app/routes/replay.py:118,170,182,198`
- **Detail**: `step_forward`, `update_speed`, `get_status`, `delete_session` menggunakan `_sessions.get()` tapi dict tidak pernah didefinisikan. Hanya `start_replay` yang menggunakan Redis.
- **Impact**: `NameError` pada setiap call ke 4 endpoint replay.
- **Fix**: Ganti dengan `_get_session(session_id)` / `_delete_session(session_id)` dari Redis.

### C4. Missing `detect_overfitting` import — AI Optimizer /analyze crash
- **File**: `services/ai-optimizer/app/routes/optimizer.py:72`
- **Detail**: Fungsi `detect_overfitting` digunakan tapi tidak pernah diimpor dari `app.engine.overfitting_detector`.
- **Impact**: `NameError` pada setiap request `/analyze`.
- **Fix**: Tambahkan import.

### C5. Missing `REDIS_URL` in backtesting config
- **File**: `services/backtesting/app/config.py`
- **Detail**: `replay.py` mereferensi `config.REDIS_URL` tapi tidak didefinisikan di Config class.
- **Impact**: `AttributeError` saat replay endpoint dipanggil.

### C6. Missing `asyncpg` import in strategy-manager
- **File**: `services/strategy-manager/app/routes/strategies.py:44`
- **Detail**: Menangkap `asyncpg.UniqueViolationError` tapi `asyncpg` tidak diimpor.
- **Impact**: `NameError` saat membuat strategy dengan nama duplikat.

### C7. `setShowPauseInput` undefined — QuickActions crash
- **File**: `services/dashboard/src/components/risk/QuickActions.tsx:54`
- **Detail**: Memanggil `setShowPauseInput(false)` tapi yang ada hanya `setShowPauseModal`.
- **Impact**: Runtime crash saat user mencoba pause.

### C8. Missing backend routes — Cross-account portfolio 404
- **Frontend** memanggil:
  - `GET /api/portfolio/cross-account`
  - `GET /api/portfolio/accounts`
- **Backend**: Route tidak ada di `services/api-gateway/app/routes/portfolio.py`
- **Impact**: Halaman cross-account portfolio completely broken.

### C9. Frontend calls wrong endpoint — Per-account risk
- **File**: `services/dashboard/src/lib/api.ts:391-392`
- **Detail**: Memanggil `GET /api/risk/status?account_id=...` tapi backend tidak accept `account_id` parameter.
- **Impact**: Mengembalikan global risk state, bukan per-account.

### C10. Non-atomic read-modify-write on risk state
- **File**: `services/api-gateway/app/routes/risk.py:453-480`
- **Detail**: TOCTOU race condition pada safety-critical risk limits.
- **Impact**: Concurrent requests dapat overwrite perubahan satu sama lain.

### C11. Cascading parameter index corruption
- **File**: `services/api-gateway/app/routes/risk.py:650-656`
- **Detail**: Loop forward replacement `$1`→`$6` lalu `$6`→`$11` terakumulasi.
- **Impact**: Query SQL corrupt untuk 3+ parameter update.

---

## 🟠 HIGH ISSUES (25) — Signifikan, perlu perbaikan segera

### Security
| # | Issue | File | Impact |
|---|-------|------|--------|
| H1 | JWT stored in `localStorage` (XSS vulnerable) | `lib/api.ts:47` | Token theft via XSS |
| H2 | Missing AuthGuard on replay, orderbook, analytics pages | `app/replay/page.tsx` etc | Unauthenticated access |
| H3 | Internal API key comparison not constant-time | `routes/logs.py:31` | Timing attack |
| H4 | WS user ID key mismatch — rate limiting bypassed | `routes/ws.py:122` | All WS connections use "unknown" key |
| H5 | Notification service has no authentication | `services/notification/app/main.py` | Exposed health/metrics |

### Data Integrity
| # | Issue | File | Impact |
|---|-------|------|--------|
| H6 | Risk state TTL 60s — data loss | `routes/risk.py:290,351,407,480` | Risk overrides silently expire |
| H7 | Rate limiter dicts grow unbounded | `middleware/auth.py:38-40` | Memory leak |
| H8 | Emergency stop rate limiter no cleanup | `routes/risk.py:61-81` | Memory leak |
| H9 | Unbounded opportunity array via WS | `hooks/useOpportunityFeed.ts:82-85` | Memory leak over time |
| H10 | `return undefined as T` on 401 response | `lib/api.ts:90-91` | Silent data corruption |

### API Contract Mismatches (snake_case vs camelCase)
| # | Backend Model | Frontend Type | Impact |
|---|---------------|---------------|--------|
| H11 | `OpportunityResponse` (snake_case) | `Opportunity` (camelCase) | All fields `undefined` |
| H12 | `BacktestStatus/Summary/Trade` (snake_case) | camelCase types | All backtesting data broken |
| H13 | `AccountResponse` (snake_case) | `Account` (camelCase) | Account data broken |
| H14 | `ABTestResponse` (snake_case) | `ABTest` (camelCase) | A/B test data broken |
| H15 | `SuggestionResponse` (snake_case) | `Suggestion` (camelCase) | Suggestion data broken |
| H16 | `OverfittingAnalysisResponse` (snake_case) | `OverfittingAnalysis` (camelCase) | Analysis data broken |
| H17 | `ABTestResultSummary` (nested) | flat structure | A/B results broken |

### Architecture
| # | Issue | Service | Impact |
|---|-------|---------|--------|
| H18 | Notification uses aiohttp (not FastAPI) | notification | Maintenance burden |
| H19 | Inconsistent JWT `verify_exp` handling | analytics/strategy/portfolio | Security risk |
| H20 | `DATABASE_URL` vs `POSTGRES_URL` naming | notification | Config confusion |
| H21 | Missing `drawdownThreshold` in frontend update type | `types/index.ts:319` | Cannot update drawdown |
| H22 | `Decimal(0) or 0` returns `int(0)` | `ai-assistant/routes/assistant.py:166` | Type mismatch |
| H23 | `get_pool()` returns `None` after failed init | `ai-optimizer/db.py`, `ai-assistant/db.py` | AttributeError |
| H24 | Multi-user data leakage via empty user_id | `ai-assistant/engine/performance_qa.py:40` | Privacy violation |
| H25 | CORS only on account-manager | `account-manager/main.py` | Inconsistent security |

---

## 🟡 MEDIUM ISSUES (39) — Perlu perhatian

### Backend
- Shallow health checks (analytics, strategy, backtesting)
- Per-batch NATS connection churn (analytics)
- No CSRF protection except account-manager
- Runtime DDL in notification service
- Inconsistent list response field naming (`accounts` vs `items`)
- Active alerts accumulation bug (health.py)
- Admin WS error exposes internal details
- No input validation on `trade_id` path parameter
- Database credentials exposed in process list
- Temp file not cleaned up on restore failure
- CORS_ORIGINS space handling
- Rate limiter O(n) scan under lock
- Unknown `pattern_type` falls through silently
- LLM empty data_points hallucination
- JSON regex fails on nested objects

### Frontend
- Dual `useRiskStatus()` instances (double API calls)
- No ErrorBoundary on dashboard page
- Unchecked WS type assertions
- No client-initiated WS heartbeat
- URL injection in analytics fetch
- Missing CSRF token on replay POST
- Inline data transformations (no memoization)
- `ServiceCard` and `OpportunityRow` not memoized
- `configValue: any` in SystemConfig types
- Duplicate type definitions (useOrderbook, DecisionLog)
- Private key in React state after submission
- Sensitive config values editable in plain text
- `downloadCSV`/`downloadJSON` no timeout

### API Contract
- WS context doesn't handle `portfolio_update`/`position_update`
- Backend returns extra health services not in frontend type
- `toSnakeCase()` doesn't handle acronyms correctly
- `RiskLimitsUpdate` missing `drawdownThreshold`

---

## 🟢 LOW ISSUES (38) — Code quality, bisa ditunda

- Dead code (`csv_stream()`, `simulate_trade_outcome`)
- Error messages leak internal architecture
- Unused imports
- Inconsistent user ID extraction pattern
- Array index used as React key
- Redundant `Decimal.set()` calls
- Ad-hoc modals instead of `ConfirmationModal`
- `any` types in TypeScript
- Missing request timeouts on download functions
- Parameter shadows module function
- Inconsistent DB pool error handling
- `float` for financial values (portfolio-manager)
- JWT token in WS visible in DevTools
- Hardcoded WS endpoint path
- Missing client-side rate limiting on login
- Error messages expose HTTP details

---

## REKOMENDASI PERBAIKAN PRIORITAS

### Batch 1 — Fix Immediately (Critical Runtime Crashes)
1. Tambahkan `verify_jwt` function di `api-gateway/middleware/auth.py`
2. Fix `_sessions` undefined di `backtesting/routes/replay.py`
3. Tambahkan missing imports (`detect_overfitting`, `asyncpg`, `REDIS_URL`)
4. Fix `setShowPauseInput` → `setShowPauseModal` di `QuickActions.tsx`

### Batch 2 — Fix WS & API Contract (Data Flow)
1. Ubah frontend WS untuk pass token sebagai query param
2. Tambahkan `alias_generator=to_camel` ke SEMUA backend Pydantic models
3. Tambahkan missing backend routes untuk cross-account portfolio
4. Fix `fetchPerAccountRisk` endpoint

### Batch 3 — Fix Security Issues
1. Migrate JWT dari localStorage ke HttpOnly cookies
2. Tambahkan AuthGuard ke replay, orderbook, analytics pages
3. Gunakan `hmac.compare_digest` untuk API key comparison
4. Fix WS user ID key mismatch

### Batch 4 — Fix Data Integrity
1. Perpanjang risk state TTL dari 60s ke 3600s
2. Fix rate limiter memory leaks (gunakan TTLCache)
3. Cap opportunity array di frontend
4. Fix `return undefined as T` → throw error

### Batch 5 — Architecture Alignment
1. Standarisasi notification service (FastAPI atau dokumentasi)
2. Standarisasi health check depth
3. Standarisasi CORS/CSRF middleware
4. Standarisasi config naming (`POSTGRES_URL`)
