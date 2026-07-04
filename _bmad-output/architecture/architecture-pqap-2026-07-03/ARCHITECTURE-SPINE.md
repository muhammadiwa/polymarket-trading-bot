---
name: 'PQAP'
type: architecture-spine
purpose: build-substrate
altitude: initiative
paradigm: 'event-driven hexagonal'
scope: 'Full PQAP system'
status: draft
created: 2026-07-03
updated: 2026-07-03
binds: ['prd-pqap-2026-07-03', 'brainstorm-pqap-2026-07-02']
sources: ['prd-pqap-2026-07-03', 'addendum-pqap-2026-07-03', 'brief-pqap-2026-07-03', 'brainstorm-pqap-2026-07-02']
companions: []
---

# Architecture Spine — PQAP

## Design Paradigm

**Event-driven hexagonal architecture (ports & adapters).**

The core domain logic — opportunity detection, risk evaluation, position management, portfolio allocation — is isolated inside a hexagonal core. External dependencies (Polymarket CLOB API, Redis, PostgreSQL, NATS, Telegram) are plugged in through ports and adapters. Components communicate asynchronously via domain events on NATS, with synchronous RPC only where latency demands it (risk check before trade).

**Why this paradigm:**

1. **Testability.** Every port can be mocked. The arbitrage engine can be tested without touching Polymarket. The risk engine can be tested without Redis.
2. **Swappability.** Polymarket API changes? Swap the adapter. Want Kafka instead of NATS? Swap the transport adapter. The core doesn't care.
3. **Loose coupling via events.** Scanner emits `MarketPriceUpdated`. Arb engine subscribes. Execution engine subscribes. Dashboard subscribes. No component knows about the others.
4. **Latency-aware.** The paradigm doesn't force everything to be async. The Pit Boss risk check is a synchronous port call — the execution engine blocks on it because a 10ms check is cheaper than a blown risk limit.

**Layering:**

```
┌──────────────────────────────────────────────────────────┐
│                    DRIVERS (Adapters)                      │
│  Polymarket WS · REST · CLOB · Redis · PG · NATS · TG   │
├──────────────────────────────────────────────────────────┤
│                    PORTS (Interfaces)                      │
│  MarketDataPort · OrderPort · RiskPort · NotifyPort       │
│  StatePort · EventPort · MetricsPort                      │
├──────────────────────────────────────────────────────────┤
│                    DOMAIN CORE                             │
│  Scanner · ArbEngine · Execution · Position · Portfolio   │
│  Risk · Strategy · Analytics · Backtest                   │
├──────────────────────────────────────────────────────────┤
│                    APPLICATION SERVICES                    │
│  Orchestrator · Reconciler · CircuitBreaker · Scheduler   │
└──────────────────────────────────────────────────────────┘
```

---

## Invariants & Rules

### AD-1: Service Boundary — Scanner

**Binds:** FR-1 through FR-8 (Market Scanner)
**Prevents:** Scanner knowing about execution logic; stale data reaching downstream
**Rule:** The Scanner is the **sole producer** of market data events. No other component may subscribe directly to Polymarket WebSocket or REST APIs. All market data flows through Scanner → NATS → consumers. Scanner owns the market catalog (in-memory + Redis cache). Scanner is responsible for connection lifecycle, reconnection, state reconciliation after disconnect, and stale detection. Downstream consumers receive `MarketPriceUpdated`, `MarketDiscovered`, `MarketStale`, and `MarketRemoved` events.

### AD-2: Service Boundary — Arbitrage Engine

**Binds:** FR-9 through FR-16 (Arb Engine)
**Prevents:** Direct execution from detection; unfiltered opportunities reaching execution
**Rule:** The Arb Engine is a **pure function** of market state → scored opportunities. It subscribes to `MarketPriceUpdated` events, runs detection algorithms (YES+NO arb, cross-market arb, liquidity capture), scores each opportunity (`spread × liquidity × fill_probability`), and emits `OpportunityDetected` events. It **never** executes. It **never** modifies market state. It logs every opportunity (including filtered ones) to TimescaleDB for backtesting. Opportunities below configurable score threshold are logged but not emitted to execution.

### AD-3: Service Boundary — Execution Engine

**Binds:** FR-17 through FR-24 (Execution Engine)
**Prevents:** Unchecked trades; duplicate orders; partial execution without tracking
**Rule:** The Execution Engine is the **sole writer** to Polymarket CLOB API. It subscribes to `OpportunityDetected` events, performs a **synchronous** risk check against the Pit Boss (Redis), calculates order parameters, places orders, monitors fills, and emits `OrderPlaced`, `OrderFilled`, `OrderPartialFill`, `OrderCancelled`, `OrderFailed` events. Every order attempt gets a unique client order ID (UUID) for idempotency. Both legs of YES+NO arb are placed within 500ms; if one fails, the other is cancelled within 1s. Circuit breaker: 5 consecutive API errors halts all trading, requires manual resume.

### AD-4: Pit Boss — Centralized Risk Authority

**Binds:** FR-38 through FR-47 (Risk Management)
**Prevents:** Trades bypassing risk checks; inconsistent risk state across components
**Rule:** The Pit Boss is the **sole authority** on whether a trade may proceed. It lives as a set of risk state keys in Redis, managed by the Risk Management service. Before every trade, the Execution Engine performs a **synchronous GET** on the Pit Boss risk state (< 10ms). The Pit Boss evaluates: daily budget remaining, position limits, correlation limits, win streak (Batasi Win), drawdown circuit breaker, and metabolic rate. Returns `ALLOW` or `DENY` with reason. No trade bypasses the Pit Boss. The Risk Management service is the **sole writer** to Pit Boss keys. Other components are **read-only** consumers of risk state.

### AD-5: State Reconciliation Engine

**Binds:** FR-6, FR-26, brainstorm "State Reconciliation Engine"
**Prevents:** Silent data corruption; drift between internal state and exchange state
**Rule:** Reconciliation is a **continuous background process**, not a one-time check. Three reconciliation loops run independently:

1. **Market data reconciliation** (Scanner): After every WebSocket reconnect, fetch full orderbook snapshot via REST and compare. Alert if prices differ by more than 1 tick.
2. **Position reconciliation** (Position Manager): Every 60s, fetch positions from Polymarket API and compare with internal state. Alert on any discrepancy.
3. **Order reconciliation** (Execution Engine): Every 30s, fetch open orders from Polymarket API and compare with internal order state. Cancel orphaned orders.

All reconciliation events are logged. Persistent mismatches (> 3 consecutive) trigger emergency stop.

### AD-6: Data Ownership — PostgreSQL

**Binds:** FR-62 through FR-65 (Trade History), FR-70 through FR-74 (Strategy Manager)
**Prevents:** Multiple services writing to the same table; schema drift
**Rule:** PostgreSQL is the **source of truth** for:

- **trades** table — written by Execution Engine only (append-only, immutable)
- **strategies** table — written by Strategy Manager only
- **positions** table — written by Position Manager only
- **risk_events** table — written by Risk Management only
- **accounts** table — written by Portfolio Manager only
- **markets** table — written by Scanner only

Each table has a single writer service. Reads are unrestricted. Schema migrations are managed by a dedicated migration tool (golang-migrate or Alembic), version-controlled, and applied at deployment.

### AD-7: Data Ownership — TimescaleDB

**Binds:** FR-16 (opportunity logging), FR-56 through FR-61 (Analytics)
**Prevents:** Hot-path writes competing with analytical queries
**Rule:** TimescaleDB (hypertable extension on PostgreSQL) stores **time-series data only**:

- **market_prices** — hypertable, partitioned by time (1-day chunks). Written by Scanner. Retention: 1 year raw, 5 years aggregated.
- **opportunities** — hypertable, partitioned by time. Written by Arb Engine. Retention: 3 years.
- **system_metrics** — hypertable, partitioned by time. Written by all services via Prometheus exporter. Retention: 90 days raw, 2 years aggregated.

Analytics queries run against TimescaleDB, never against the OLTP tables. Continuous aggregates pre-compute daily/weekly/monthly PnL summaries.

### AD-8: Data Ownership — Redis

**Binds:** FR-46 (risk state), brainstorm "Centralized Risk Monitor via Redis"
**Prevents:** Stale risk state; inconsistent reads; Redis as primary data store
**Rule:** Redis is an **ephemeral cache and coordination layer**, never the source of truth. Contents:

- **Pit Boss risk state** — TTL 60s, refreshed by Risk Management service every 30s. If TTL expires, trading halts (fail-safe).
- **Market catalog cache** — TTL 5s, refreshed by Scanner. Dashboard reads from cache for speed.
- **Session state** — Dashboard sessions, API rate limit counters.
- **Pub/Sub** — Used only for low-latency intra-service signals (emergency stop broadcast). Not used for event streaming (that's NATS).

Redis state is **reconstructable** from PostgreSQL on restart. No data that lives only in Redis.

### AD-9: Event Bus — NATS

**Binds:** All FRs with async communication
**Prevents:** Point-to-point coupling; lost events; event ordering issues
**Rule:** NATS is the **primary event bus** for all asynchronous communication. Subject hierarchy:

```
pqap.market.{market_id}.price     # MarketPriceUpdated
pqap.market.discovered             # MarketDiscovered
pqap.market.stale                  # MarketStale
pqap.opportunity.detected          # OpportunityDetected
pqap.order.placed                  # OrderPlaced
pqap.order.filled                  # OrderFilled
pqap.order.cancelled               # OrderCancelled
pqap.order.failed                  # OrderFailed
pqap.position.opened               # PositionOpened
pqap.position.closed               # PositionClosed
pqap.position.updated              # PositionUpdated
pqap.risk.alert                    # RiskAlert
pqap.risk.emergency                # EmergencyStop
pqap.system.health                 # SystemHealth
pqap.notification.send             # NotificationRequest
```

Events are **fire-and-forget** with at-least-once delivery. Consumers are **idempotent** (deduplicate by event UUID). NATS JetStream is used for durable subscriptions where event loss is unacceptable (order fills, risk alerts). Subject-based routing enables fan-out without consumer coupling.

### AD-10: Communication Pattern — Sync vs Async

**Binds:** FR-18, FR-45, FR-19
**Prevents:** Blocking on slow external calls; async risk checks allowing unchecked trades
**Rule:**

| Interaction | Pattern | Justification |
|---|---|---|
| Execution → Pit Boss | **Sync RPC** (Redis GET) | Risk check must complete before trade; 10ms budget |
| Scanner → Polymarket WS | **Sync** (WebSocket) | Connection management requires sync lifecycle |
| Execution → Polymarket CLOB | **Sync HTTP** | Order placement requires response |
| Scanner → NATS | **Async** (publish) | Market data is fire-and-forget fan-out |
| Arb Engine → NATS | **Async** (publish) | Opportunities are fire-and-forget |
| Execution → NATS | **Async** (publish) | Order events are fire-and-forget |
| Position Manager → Polymarket API | **Sync HTTP** (periodic) | Reconciliation requires request-response |
| Risk Management → Redis | **Sync** (write) | Risk state updates must be confirmed |
| Notification → Telegram | **Async** (fire-and-forget) | Notification delivery is best-effort |
| Dashboard → Backend | **WebSocket** (push) | Real-time updates to frontend |

### AD-11: Error Handling — Circuit Breaker Pattern

**Binds:** FR-21 (execution circuit breaker), FR-44 (emergency stop)
**Prevents:** API death spiral; cascading failures; resource exhaustion
**Rule:** Every external call (Polymarket API, Redis, PostgreSQL, Telegram) is wrapped in a circuit breaker with three states:

- **Closed** (normal): Requests pass through. Failure counter increments on error.
- **Open** (tripped): After N consecutive failures (configurable, default 5), all requests fail immediately for a cooldown period (default 60s).
- **Half-open** (probe): After cooldown, one probe request is sent. If it succeeds, circuit closes. If it fails, circuit reopens.

Additionally, a **global emergency stop** is triggered by:
- Polymarket API unreachable for > 5 minutes
- Data corruption detected by reconciliation
- Daily budget exhausted
- Drawdown circuit breaker tripped

Emergency stop: halt all trading, cancel all open orders, alert Juragan via Telegram (critical notification, bypasses throttling).

### AD-12: Paper Trading — Execution Mode Flag

**Binds:** FR-91 through FR-95 (Paper Trading)
**Prevents:** Paper trades affecting live positions; mode confusion
**Rule:** A global `execution_mode` enum (`LIVE`, `PAPER`, `REPLAY`) is stored in Redis and passed to all components at startup. In `PAPER` mode:

- The Execution Engine simulates order fills based on real orderbook depth (no CLOB API calls).
- Simulated positions are tracked in a separate `paper_positions` table.
- Simulated PnL is tracked separately from live PnL.
- The Dashboard clearly labels paper vs live data.
- Switching modes requires a restart (no hot-switch to prevent accidental live trades during paper testing).

### AD-13: Strategy Isolation

**Binds:** FR-104 through FR-107 (Multi-Strategy)
**Prevents:** One strategy's failure affecting others; resource starvation
**Rule:** Each strategy runs as a **logical goroutine group** within the Go services (not a separate process — overkill for single-user). Isolation is enforced by:

- Separate capital allocation (checked by Portfolio Manager)
- Separate risk limits (checked by Pit Boss)
- Separate position tracking (positions tagged with strategy_id)
- Separate performance metrics (aggregated by Analytics)
- Panic recovery: a strategy goroutine that panics is caught, logged, and deactivated without crashing the service

Shared resources (market data, risk oversight, NATS) are not isolated — they're infrastructure.

### AD-14: Secret Management

**Binds:** Addendum "Security Best Practices"
**Prevents:** Secret leakage; unauthorized API access; key compromise
**Rule:**

- Private keys and API secrets are stored in **Kubernetes Secrets** (encrypted at rest via etcd encryption).
- Secrets are injected as environment variables at pod startup. They are **never** mounted as files.
- Secrets are **never** logged, **never** included in error messages, **never** committed to version control.
- API key rotation is manual (quarterly recommendation). Rotation procedure documented in runbook.
- Dashboard and Admin Panel require authentication (JWT with configurable session timeout). Even for single-user, auth prevents accidental exposure if port-forwarded.

### AD-15: Deployment Topology

**Binds:** Addendum "Deployment Architecture"
**Prevents:** Single point of failure; resource contention; scaling confusion
**Rule:** PQAP deploys as a **single Kubernetes namespace** with the following pods:

| Pod | Replicas | Language | Resources |
|---|---|---|---|
| scanner | 1 | Go | 256Mi RAM, 0.25 CPU |
| execution-engine | 1 | Go | 256Mi RAM, 0.25 CPU |
| risk-manager | 1 | Go | 128Mi RAM, 0.1 CPU |
| position-manager | 1 | Go | 128Mi RAM, 0.1 CPU |
| arb-engine | 1 | Go | 256Mi RAM, 0.25 CPU |
| portfolio-manager | 1 | Python | 256Mi RAM, 0.25 CPU |
| analytics | 1 | Python | 512Mi RAM, 0.5 CPU |
| api-gateway | 1 | Python (FastAPI) | 256Mi RAM, 0.25 CPU |
| notification | 1 | Python | 128Mi RAM, 0.1 CPU |
| dashboard | 1 | Next.js | 256Mi RAM, 0.25 CPU |
| redis | 1 (StatefulSet) | — | 256Mi RAM, 0.1 CPU |
| postgresql | 1 (StatefulSet) | — | 512Mi RAM, 0.5 CPU |
| nats | 1 (StatefulSet) | — | 128Mi RAM, 0.1 CPU |

Single replicas for all services (single-user, personal use). Horizontal scaling is not a design goal. Vertical scaling via resource limit adjustments.

### AD-16: Capital Scaling Tiers

**Binds:** FR-31 through FR-37 (Portfolio Manager), Addendum "Capital Scaling Strategy"
**Prevents:** Oversized positions on small capital; premature strategy activation
**Rule:** Capital tier is a **derived value** (total capital → tier lookup), not a stored configuration. Tier determines:

- Which strategies are active
- Max position size as % of capital
- Risk budget (daily loss limit %)

Tier promotion requires capital above threshold for 7 consecutive days. Demotion is immediate on capital drop. Manual override allowed with warning. The Portfolio Manager recalculates tier on every capital change event.

### AD-17: Observability

**Binds:** FR-8, FR-43 (metabolic rate), FR-52 (system health)
**Prevents:** Blind spots in system behavior; undetected degradation
**Rule:** Every service exports Prometheus metrics on `/metrics`. Key metric families:

- **Latency histograms:** Order placement, risk check, price update processing
- **Counters:** Trades executed, opportunities detected, orders failed, circuit breaker trips
- **Gauges:** Open positions, daily PnL, capital utilization, WebSocket connection status
- **System:** CPU, memory, goroutine count, GC pauses

Grafana dashboards visualize all metrics. Alertmanager fires on threshold breaches (see monitoring table in addendum). Structured logging (JSON) to stdout, collected by Kubernetes log aggregation.

### AD-18: Database Migrations

**Binds:** All FRs requiring persistence
**Prevents:** Schema drift; broken deployments; data loss
**Rule:** All schema changes are managed by **golang-migrate** (for Go services) and **Alembic** (for Python services), both pointing at the same PostgreSQL instance. Migrations are:

- Version-controlled in the `migrations/` directory
- Forward-only (no down migrations in production)
- Applied at deployment time by init containers
- Tested in staging environment before production
- Never destructive (columns are deprecated, not dropped, until confirmed unused)

---

## Consistency Conventions

| Convention | Rule |
|---|---|
| **Event naming** | Past tense verb + noun: `MarketPriceUpdated`, `OrderFilled`, `PositionClosed` |
| **Event schema** | All events include: `event_id` (UUID), `event_type` (string), `timestamp` (ISO 8601 UTC), `source` (service name), `payload` (JSON) |
| **Idempotency** | All event consumers deduplicate by `event_id`. All order operations use `client_order_id` (UUID). |
| **Decimal precision** | All monetary values use `decimal.Decimal` (Go) / `Decimal` (Python) — never `float64`. Prices: 4 decimal places. Quantities: 8 decimal places. PnL: 8 decimal places. |
| **Time** | All timestamps are UTC, stored as `TIMESTAMPTZ`. Display timezone configurable (default: UTC). |
| **Error naming** | Errors are values, not strings. Use typed errors: `ErrInsufficientBalance`, `ErrRiskDenied`, `ErrSlippageExceeded`. |
| **Configuration** | All configurable values have sensible defaults. Runtime-adjustable via Redis (risk params) or PostgreSQL (strategy configs). No hardcoded magic numbers. |
| **Logging** | Structured JSON logs. Every log entry includes: `timestamp`, `level`, `service`, `request_id` (if applicable), `message`, `context` (key-value pairs). |
| **Metrics naming** | Prometheus convention: `pqap_{service}_{metric_name}_{unit}` (e.g., `pqap_execution_order_latency_ms`). |

---

## Stack

| Layer | Technology | Version (verified 2026-07-03) | Purpose |
|---|---|---|---|
| **Execution runtime** | Go | 1.26.4 | Scanner, Arb Engine, Execution, Position, Risk — concurrency, low latency |
| **AI/Analytics runtime** | Python | 3.13.14 | Portfolio, Analytics, Backtest, AI Optimizer, Notifications — ML ecosystem |
| **API framework** | FastAPI | 0.139.0 | API Gateway — async Python web framework |
| **Frontend** | Next.js | 16.2.10 (LTS) | Dashboard, Admin — React SSR |
| **Cache / Coordination** | Redis | 8.8.0 | Pit Boss state, market cache, session state |
| **OLTP Database** | PostgreSQL | 17.10 | Trades, positions, strategies, risk events |
| **Time-Series** | TimescaleDB | 2.x (on PG 17) | Market prices, opportunities, system metrics |
| **Event Bus** | NATS | 2.10+ (JetStream) | Async event streaming between services |
| **Container** | Docker | 24+ | Containerization |
| **Orchestration** | Kubernetes | 1.36.2 | Container orchestration |
| **Metrics** | Prometheus | 3.12.0 | Metrics collection and alerting |
| **Dashboards** | Grafana | 13.0.3 | Operational monitoring dashboards |
| **Monitoring** | Alertmanager | (bundled with Prometheus) | Alert routing and deduplication |
| **Blockchain** | Polygon (Chain ID 137) | — | Polymarket settlement layer |
| **Stablecoin** | USDC (Polygon) | — | Trading collateral |
| **Notifications** | Telegram Bot API | — | Primary alert channel |
| **Go HTTP client** | `net/http` + `websocket/gorilla` | — | Polymarket API communication |
| **Go Redis** | `go-redis/redis/v9` | — | Pit Boss interaction |
| **Go NATS** | `nats.go` | — | Event bus |
| **Go PG driver** | `pgx/v5` | — | PostgreSQL driver |
| **Python Polymarket SDK** | `polymarket-client` | — | Polymarket API (analytics layer) |
| **Python ML** | `scikit-learn`, `pandas`, `numpy` | — | AI Strategy Optimizer |
| **Python Telegram** | `python-telegram-bot` | — | Notification delivery |

---

## Structural Seed

```
pqap/
├── services/
│   ├── scanner/                    # Go — Market data ingestion
│   │   ├── cmd/
│   │   │   └── main.go
│   │   ├── internal/
│   │   │   ├── websocket/          # Polymarket WS client
│   │   │   │   ├── client.go       # Connection lifecycle, reconnect
│   │   │   │   ├── subscriber.go   # Market subscription management
│   │   │   │   └── reconciler.go   # Post-reconnect state reconciliation
│   │   │   ├── rest/               # Polymarket REST client
│   │   │   │   ├── client.go       # HTTP client with circuit breaker
│   │   │   │   └── batch.go        # Batched market data fetching
│   │   │   ├── catalog/            # In-memory market catalog
│   │   │   │   ├── catalog.go      # Market state management
│   │   │   │   └── stale.go        # Stale detection logic
│   │   │   └── ports/              # Port interfaces
│   │   │       ├── market_data.go  # MarketDataPort
│   │   │       └── event.go        # EventPort
│   │   └── adapters/               # Concrete implementations
│   │       ├── nats_publisher.go
│   │       └── redis_cache.go
│   │
│   ├── arb-engine/                 # Go — Opportunity detection
│   │   ├── cmd/
│   │   │   └── main.go
│   │   ├── internal/
│   │   │   ├── detector/           # Detection algorithms
│   │   │   │   ├── simple_arb.go   # YES+NO arbitrage
│   │   │   │   ├── cross_market.go # Cross-market arbitrage
│   │   │   │   └── liquidity.go    # Liquidity capture
│   │   │   ├── scorer/             # Opportunity scoring
│   │   │   │   └── scorer.go       # spread × liquidity × fill_probability
│   │   │   ├── filter/             # Threshold filtering
│   │   │   │   └── filter.go
│   │   │   └── ports/
│   │   │       ├── market_data.go
│   │   │       └── event.go
│   │   └── adapters/
│   │       ├── nats_subscriber.go
│   │       └── nats_publisher.go
│   │
│   ├── execution-engine/           # Go — Order execution
│   │   ├── cmd/
│   │   │   └── main.go
│   │   ├── internal/
│   │   │   ├── executor/           # Order placement logic
│   │   │   │   ├── executor.go     # Core execution flow
│   │   │   │   ├── atomic.go       # YES+NO atomic execution
│   │   │   │   └── slippage.go     # Slippage protection
│   │   │   ├── monitor/            # Fill monitoring
│   │   │   │   └── fill_monitor.go
│   │   │   ├── circuit_breaker/    # API circuit breaker
│   │   │   │   └── breaker.go
│   │   │   └── ports/
│   │   │       ├── order.go        # OrderPort (CLOB API)
│   │   │       ├── risk.go         # RiskPort (Pit Boss check)
│   │   │       └── event.go
│   │   └── adapters/
│   │       ├── polymarket_clob.go  # Polymarket CLOB adapter
│   │       ├── redis_risk.go       # Pit Boss risk check adapter
│   │       └── nats_publisher.go
│   │
│   ├── risk-manager/               # Go — Centralized risk authority
│   │   ├── cmd/
│   │   │   └── main.go
│   │   ├── internal/
│   │   │   ├── pitboss/            # Pit Boss logic
│   │   │   │   ├── pitboss.go      # Risk evaluation engine
│   │   │   │   ├── daily_budget.go # Daily loss limit
│   │   │   │   ├── position_limit.go
│   │   │   │   ├── correlation.go  # Correlation limits
│   │   │   │   ├── win_streak.go   # Batasi Win
│   │   │   │   ├── drawdown.go     # Drawdown circuit breaker
│   │   │   │   └── metabolic.go    # System resource monitor
│   │   │   ├── emergency/          # Emergency stop
│   │   │   │   └── emergency.go
│   │   │   └── ports/
│   │   │       ├── risk_state.go   # RiskStatePort (Redis)
│   │   │       └── event.go
│   │   └── adapters/
│   │       ├── redis_writer.go
│   │       └── nats_publisher.go
│   │
│   ├── position-manager/           # Go — Position tracking
│   │   ├── cmd/
│   │   │   └── main.go
│   │   ├── internal/
│   │   │   ├── tracker/            # Position lifecycle
│   │   │   │   ├── tracker.go      # Open, monitor, close, settle
│   │   │   │   ├── pnl.go          # PnL calculation
│   │   │   │   └── reconciler.go   # API state reconciliation
│   │   │   └── ports/
│   │   │       ├── position.go     # PositionPort (Polymarket API)
│   │   │       └── event.go
│   │   └── adapters/
│   │       ├── polymarket_account.go
│   │       ├── postgres_repo.go
│   │       └── nats_subscriber.go
│   │
│   ├── portfolio-manager/          # Python — Capital management
│   │   ├── app/
│   │   │   ├── main.py
│   │   │   ├── capital.py          # Capital tracking, tier system
│   │   │   ├── allocation.py       # Strategy weight allocation
│   │   │   └── scaling.py          # Auto-tier promotion/demotion
│   │   └── adapters/
│   │       ├── postgres_repo.py
│   │       └── nats_client.py
│   │
│   ├── analytics/                  # Python — Performance analytics
│   │   ├── app/
│   │   │   ├── main.py
│   │   │   ├── metrics.py          # PnL, Sharpe, drawdown, VaR
│   │   │   ├── anomaly.py          # Performance anomaly detection
│   │   │   └── export.py           # CSV/JSON export
│   │   └── adapters/
│   │       ├── timescale_repo.py
│   │       └── nats_client.py
│   │
│   ├── backtest/                   # Python — Backtesting engine
│   │   ├── app/
│   │   │   ├── main.py
│   │   │   ├── replayer.py         # Historical data replay
│   │   │   ├── simulator.py        # Execution simulation
│   │   │   └── reporter.py         # Backtest reports
│   │   └── adapters/
│   │       ├── timescale_repo.py
│   │       └── strategy_runner.py
│   │
│   ├── ai-optimizer/               # Python — AI strategy optimization
│   │   ├── app/
│   │   │   ├── main.py
│   │   │   ├── analyzer.py         # Trade pattern analysis
│   │   │   ├── suggester.py        # Parameter suggestions
│   │   │   └── overfit_detector.py # Overfitting detection
│   │   └── adapters/
│   │       ├── postgres_repo.py
│   │       └── sklearn_model.py
│   │
│   ├── notification/               # Python — Alert delivery
│   │   ├── app/
│   │   │   ├── main.py
│   │   │   ├── telegram.py         # Telegram bot adapter
│   │   │   ├── throttler.py        # Rate limiting
│   │   │   └── categorizer.py      # Severity classification
│   │   └── adapters/
│   │       ├── telegram_bot.py
│   │       └── nats_subscriber.py
│   │
│   ├── api-gateway/                # Python (FastAPI) — API layer
│   │   ├── app/
│   │   │   ├── main.py
│   │   │   ├── routes/
│   │   │   │   ├── portfolio.py
│   │   │   │   ├── positions.py
│   │   │   │   ├── trades.py
│   │   │   │   ├── strategies.py
│   │   │   │   ├── risk.py
│   │   │   │   ├── analytics.py
│   │   │   │   └── admin.py
│   │   │   ├── websocket/          # WebSocket server for dashboard
│   │   │   │   └── hub.go → ws.py
│   │   │   └── middleware/
│   │   │       ├── auth.py
│   │   │       └── rate_limit.py
│   │   └── adapters/
│   │       ├── postgres_repo.py
│   │       ├── redis_client.py
│   │       └── nats_client.py
│   │
│   └── dashboard/                  # Next.js — Frontend
│       ├── src/
│       │   ├── app/
│       │   │   ├── page.tsx         # Portfolio overview
│       │   │   ├── positions/
│       │   │   ├── trades/
│       │   │   ├── analytics/
│       │   │   ├── risk/
│       │   │   ├── strategies/
│       │   │   ├── orderbook/
│       │   │   ├── replay/
│       │   │   └── admin/
│       │   ├── components/
│       │   │   ├── PortfolioCard.tsx
│       │   │   ├── PositionTable.tsx
│       │   │   ├── OpportunityFeed.tsx
│       │   │   ├── RiskStatus.tsx
│       │   │   ├── PnLChart.tsx
│       │   │   ├── OrderbookViewer.tsx
│       │   │   └── EmergencyStop.tsx
│       │   ├── hooks/
│       │   │   ├── useWebSocket.ts
│       │   │   └── usePortfolio.ts
│       │   └── lib/
│       │       ├── api.ts
│       │       └── ws.ts
│       └── public/
│
├── shared/
│   ├── proto/                      # Shared event schemas
│   │   ├── events.go               # Go event types
│   │   └── events.py               # Python event types
│   ├── models/                     # Shared domain models
│   │   ├── market.go / market.py
│   │   ├── opportunity.go / opportunity.py
│   │   ├── order.go / order.py
│   │   ├── position.go / position.py
│   │   └── risk.go / risk.py
│   └── constants/                  # Shared constants
│       ├── defaults.go / defaults.py
│       └── errors.go / errors.py
│
├── migrations/                     # Database migrations
│   ├── postgres/
│   │   ├── 001_create_markets.up.sql
│   │   ├── 002_create_trades.up.sql
│   │   ├── 003_create_positions.up.sql
│   │   ├── 004_create_risk_events.up.sql
│   │   ├── 005_create_strategies.up.sql
│   │   └── ...
│   └── timescale/
│       ├── 001_create_market_prices.up.sql
│       ├── 002_create_opportunities.up.sql
│       └── 003_create_system_metrics.up.sql
│
├── config/
│   ├── default.yaml                # Default configuration
│   ├── production.yaml             # Production overrides
│   └── paper.yaml                  # Paper trading overrides
│
├── deploy/
│   ├── docker/
│   │   ├── scanner.Dockerfile
│   │   ├── execution-engine.Dockerfile
│   │   ├── risk-manager.Dockerfile
│   │   ├── position-manager.Dockerfile
│   │   ├── arb-engine.Dockerfile
│   │   ├── portfolio-manager.Dockerfile
│   │   ├── analytics.Dockerfile
│   │   ├── backtest.Dockerfile
│   │   ├── ai-optimizer.Dockerfile
│   │   ├── notification.Dockerfile
│   │   ├── api-gateway.Dockerfile
│   │   └── dashboard.Dockerfile
│   └── k8s/
│       ├── namespace.yaml
│       ├── configmap.yaml
│       ├── secrets.yaml             # Template only — real secrets via sealed-secrets
│       ├── scanner-deployment.yaml
│       ├── execution-engine-deployment.yaml
│       ├── risk-manager-deployment.yaml
│       ├── position-manager-deployment.yaml
│       ├── arb-engine-deployment.yaml
│       ├── portfolio-manager-deployment.yaml
│       ├── analytics-deployment.yaml
│       ├── api-gateway-deployment.yaml
│       ├── notification-deployment.yaml
│       ├── dashboard-deployment.yaml
│       ├── redis-statefulset.yaml
│       ├── postgres-statefulset.yaml
│       ├── nats-statefulset.yaml
│       ├── prometheus-config.yaml
│       └── grafana-configmap.yaml
│
├── monitoring/
│   ├── prometheus/
│   │   └── prometheus.yaml
│   └── grafana/
│       ├── dashboards/
│       │   ├── trading-overview.json
│       │   ├── risk-monitor.json
│       │   ├── system-health.json
│       │   └── strategy-performance.json
│       └── provisioning/
│
├── tests/
│   ├── unit/
│   │   ├── scanner/
│   │   ├── arb-engine/
│   │   ├── execution-engine/
│   │   └── risk-manager/
│   ├── integration/
│   │   ├── scanner_arb_test.go
│   │   ├── arb_execution_test.go
│   │   └── pit_boss_test.go
│   └── e2e/
│       └── full_cycle_test.go
│
├── scripts/
│   ├── setup-wallet.py             # Wallet setup and allowance configuration
│   ├── seed-data.py                # Development seed data
│   └── backtest-runner.py          # CLI backtest runner
│
├── go.mod
├── go.sum
├── pyproject.toml                  # Python project (uv/poetry)
├── package.json                    # Dashboard (Next.js)
├── docker-compose.yaml             # Local development
├── Makefile                        # Build, test, lint commands
└── README.md
```

---

## Capability → Architecture Map

| PRD Feature | Primary Service(s) | Supporting Services | Data Store | Events Consumed | Events Produced |
|---|---|---|---|---|---|
| **4.1 Market Scanner** | `scanner` | — | Redis (cache), TimescaleDB (history) | — | `MarketPriceUpdated`, `MarketDiscovered`, `MarketStale`, `MarketRemoved` |
| **4.2 Arbitrage Engine** | `arb-engine` | — | TimescaleDB (opportunity log) | `MarketPriceUpdated`, `MarketStale` | `OpportunityDetected` |
| **4.3 Execution Engine** | `execution-engine` | `risk-manager` (Pit Boss check) | PostgreSQL (trades) | `OpportunityDetected` | `OrderPlaced`, `OrderFilled`, `OrderCancelled`, `OrderFailed` |
| **4.4 Position Manager** | `position-manager` | — | PostgreSQL (positions) | `OrderFilled`, `OrderCancelled` | `PositionOpened`, `PositionClosed`, `PositionUpdated` |
| **4.5 Portfolio Manager** | `portfolio-manager` | — | PostgreSQL (accounts, allocation) | `PositionOpened`, `PositionClosed` | `CapitalUpdated`, `TierChanged` |
| **4.6 Risk Management** | `risk-manager` | — | Redis (Pit Boss state), PostgreSQL (risk events) | `OrderFilled`, `PositionOpened`, `PositionClosed` | `RiskAlert`, `EmergencyStop`, `RiskStateUpdated` |
| **4.7 Dashboard** | `dashboard` | `api-gateway` | — (reads via API) | — (WebSocket from api-gateway) | — (user actions via API) |
| **4.8 Analytics** | `analytics` | — | TimescaleDB (queries), PostgreSQL (trades) | `PositionClosed` | — (serves via API) |
| **4.9 Trade History** | `execution-engine` (write), `api-gateway` (read) | — | PostgreSQL (trades) | `OrderFilled` | — |
| **4.10 Orderbook Viewer** | `dashboard` | `scanner` (data source) | Redis (cache) | — | — |
| **4.11 Strategy Manager** | `api-gateway` (CRUD), `arb-engine` (consumer) | — | PostgreSQL (strategies) | — | `StrategyUpdated` |
| **4.12 AI Strategy Optimizer** | `ai-optimizer` | `analytics` (data source) | PostgreSQL (trades, strategies) | — | `OptimizationSuggestion` |
| **4.13 Notification Center** | `notification` | — | PostgreSQL (history) | `RiskAlert`, `EmergencyStop`, `OrderFilled`, `OptimizationSuggestion` | — (delivers to Telegram) |
| **4.14 Backtesting** | `backtest` | `arb-engine` (reuses detection logic) | TimescaleDB (historical data) | — | — |
| **4.15 Paper Trading** | `execution-engine` (paper mode) | `position-manager` | PostgreSQL (paper_positions) | `OpportunityDetected` | `PaperOrderFilled`, `PaperPositionClosed` |
| **4.16 Replay Mode** | `dashboard` | `backtest` (data source) | TimescaleDB (historical data) | — | — |
| **4.17 AI Assistant** | `api-gateway` (LLM integration) | `analytics`, `position-manager` | — (reads via API) | — | — |
| **4.18 Multi-Strategy** | `arb-engine`, `execution-engine`, `portfolio-manager` | `risk-manager` | PostgreSQL (strategies, positions) | `StrategyUpdated` | — |
| **4.19 Multi-Account** | `portfolio-manager`, `execution-engine` | — | PostgreSQL (accounts) | — | — (deferred) |
| **4.20 Admin Panel** | `dashboard` (admin routes) | `api-gateway` | PostgreSQL (config) | — | — (deferred) |

---

## Deferred

| Feature | Phase | Reason | Architectural Impact |
|---|---|---|---|
| **Multi-Account (4.19)** | Phase 3 | Single-user, single wallet for v1. Adds account isolation complexity. | Add `account_id` foreign key to positions, trades, strategies tables. Execution Engine needs wallet routing. Low impact if schema is designed with account_id from day one (include as nullable FK, default to single account). |
| **Admin Panel (4.20)** | Phase 2 | Dashboard covers essential config. Admin panel is operational polish. | Add admin routes to api-gateway. Auth already in place. Low impact. |
| **AI Assistant (4.17)** | Phase 3 | Requires LLM integration and sufficient trade history. Not core value. | Add LLM adapter to api-gateway. Read-only access to analytics. Low impact. |
| **AI Strategy Optimizer (4.12)** | Phase 3 | Requires >100 trades for meaningful analysis. ML pipeline adds complexity. | Add `ai-optimizer` service. Uses existing analytics data. Medium impact — needs ML library dependencies. |
| **Replay Mode (4.16)** | Phase 2 | Debugging tool, not core trading. | Add replay routes to dashboard + data source from TimescaleDB. Low impact. |
| **Cross-Market Arb (4.2 partial)** | Phase 2 | Requires market relationship mapping. Higher complexity than simple arb. | Add `detector/cross_market.go`. Needs relationship data in PostgreSQL. Medium impact. |
| **Liquidity Capture (4.2 partial)** | Phase 2 | Requires limit order management and fill probability modeling. | Add `detector/liquidity.go`. Needs orderbook depth analysis. Medium impact. |
| **Correlation Limits** | Phase 2 | Requires correlation matrix computation and storage. | Add `correlation.go` to risk-manager. Needs market relationship data. Low impact. |
| **Batasi Win (Win Streak Breaker)** | Phase 2 | Requires consecutive win tracking. | Add `win_streak.go` to risk-manager. Simple counter in Redis. Low impact. |
| **Metabolic Rate Monitor** | Phase 2 | Requires system resource monitoring integration. | Add `metabolic.go` to risk-manager. Uses Prometheus node_exporter. Low impact. |
| **Orderbook Viewer (4.10)** | Phase 2 | Visualization tool, not core trading. | Add orderbook page to dashboard. Scanner already has the data. Low impact. |
| **Backtesting (4.14)** | Phase 2 | Core value but not needed for initial trading. Historical data collection starts from day one. | Add `backtest` service. Reuses arb-engine detection logic. Medium impact. |
| **Paper Trading (4.15)** | Phase 2 | Safety net for strategy testing. Not needed for first live trades. | Add `PAPER` mode to execution-engine. Separate position tracking. Medium impact. |

**Design decision:** Include `account_id` as a nullable column in all relevant tables from day one. This makes multi-account a schema migration (add NOT NULL constraint + default) rather than a table restructure. Cost: near zero. Benefit: saves a painful migration later.
