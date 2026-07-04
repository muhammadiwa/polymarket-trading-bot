---
stepsCompleted: ['step-01', 'step-02', 'step-03', 'step-04']
inputDocuments:
  - path: _bmad-output/prds/prd-pqap-2026-07-03/prd.md
    type: prd
    name: PQAP PRD
  - path: _bmad-output/architecture/architecture-pqap-2026-07-03/ARCHITECTURE-SPINE.md
    type: architecture
    name: PQAP Architecture Spine
---

# Polymarket Quant Arbitrage Platform (PQAP) - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for PQAP — a production-grade automated arbitrage trading platform for Polymarket prediction markets. Requirements are decomposed from the PRD (116 functional requirements) and Architecture Spine (18 architectural decisions) into implementable stories.

## Requirements Inventory

### Functional Requirements

| ID | Feature | Requirement | Priority |
|----|---------|-------------|----------|
| FR-1 | Market Scanner | Scanner SHALL connect to Polymarket WebSocket API and subscribe to all active binary markets | P0 |
| FR-2 | Market Scanner | Scanner SHALL maintain an internal market catalog with: market ID, slug, current YES/NO prices, spread, volume, liquidity depth | P0 |
| FR-3 | Market Scanner | Scanner SHALL detect new markets within 60 seconds of their appearance on Polymarket | P0 |
| FR-4 | Market Scanner | Scanner SHALL mark markets as "stale" if no price updates received within configurable threshold (default: 30s) | P0 |
| FR-5 | Market Scanner | Scanner SHALL implement automatic WebSocket reconnection with exponential backoff (initial: 1s, max: 60s) | P0 |
| FR-6 | Market Scanner | Scanner SHALL reconcile state after reconnection by fetching current orderbook snapshot | P0 |
| FR-7 | Market Scanner | Scanner SHALL batch REST API calls when fetching multiple market data (up to 100 markets per request) | P0 |
| FR-8 | Market Scanner | Scanner SHALL export metrics: markets tracked, update latency, connection status, stale count | P1 |
| FR-9 | Arbitrage Engine | Engine SHALL detect simple YES+NO arbitrage when YES_price + NO_price < $1.00 - min_profit_threshold | P0 |
| FR-10 | Arbitrage Engine | Engine SHALL detect cross-market arbitrage between related markets | P2 |
| FR-11 | Arbitrage Engine | Engine SHALL calculate opportunity score: spread × liquidity × fill_probability | P0 |
| FR-12 | Arbitrage Engine | Engine SHALL filter opportunities below configurable score threshold (default: 0.01) | P0 |
| FR-13 | Arbitrage Engine | Engine SHALL estimate fill probability based on orderbook depth and historical fill rates | P0 |
| FR-14 | Arbitrage Engine | Engine SHALL detect when market resolution is imminent (within 1 hour) and reduce confidence score | P1 |
| FR-15 | Arbitrage Engine | Engine SHALL identify correlated markets and flag potential cascade risk | P2 |
| FR-16 | Arbitrage Engine | Engine SHALL log all detected opportunities (including filtered ones) for backtesting analysis | P1 |
| FR-17 | Execution Engine | Engine SHALL place limit orders (GTC) by default with configurable time-in-force override | P0 |
| FR-18 | Execution Engine | Engine SHALL validate order against risk budget before placement | P0 |
| FR-19 | Execution Engine | Engine SHALL implement slippage protection: reject if price moves beyond tolerance (default: 1%) | P0 |
| FR-20 | Execution Engine | Engine SHALL handle partial fills: track filled quantity and decide cancel/wait based on strategy | P0 |
| FR-21 | Execution Engine | Engine SHALL implement circuit breaker: halt trading after N consecutive API errors (default: 5) | P0 |
| FR-22 | Execution Engine | Engine SHALL implement idempotent order placement (client order ID prevents duplicates) | P0 |
| FR-23 | Execution Engine | Engine SHALL execute both legs of YES+NO arbitrage atomically (both succeed or both cancel) | P0 |
| FR-24 | Execution Engine | Engine SHALL log every order attempt with: timestamp, market, side, price, size, result, latency | P0 |
| FR-25 | Position Manager | Manager SHALL track all open positions with: market, side, entry price, current price, quantity, unrealized PnL | P0 |
| FR-26 | Position Manager | Manager SHALL reconcile position state with Polymarket API every 60 seconds | P0 |
| FR-27 | Position Manager | Manager SHALL detect market resolution and automatically settle positions | P0 |
| FR-28 | Position Manager | Manager SHALL calculate unrealized PnL using current market prices | P0 |
| FR-29 | Position Manager | Manager SHALL alert when position exceeds configured limits | P0 |
| FR-30 | Position Manager | Manager SHALL support manual position exit (close at market) | P0 |
| FR-31 | Portfolio Manager | Manager SHALL track total capital: deposits + realized PnL + unrealized PnL | P0 |
| FR-32 | Portfolio Manager | Manager SHALL allocate capital across strategies based on configurable weights | P1 |
| FR-33 | Portfolio Manager | Manager SHALL enforce per-strategy position limits (configurable, default: 20% of capital) | P0 |
| FR-34 | Portfolio Manager | Manager SHALL enforce per-market position limits (configurable, default: 10% of capital) | P0 |
| FR-35 | Portfolio Manager | Manager SHALL auto-adjust position limits based on capital tier | P1 |
| FR-36 | Portfolio Manager | Manager SHALL calculate capital utilization rate (% of capital deployed) | P1 |
| FR-37 | Portfolio Manager | Manager SHALL support manual capital rebalancing (adjust strategy weights) | P2 |
| FR-38 | Risk Management | System SHALL enforce daily loss limit (default: 2% of capital, configurable) | P0 |
| FR-39 | Risk Management | System SHALL enforce max position per market (default: 10% of capital, configurable) | P0 |
| FR-40 | Risk Management | System SHALL enforce max correlated positions (default: 3, configurable) | P2 |
| FR-41 | Risk Management | System SHALL implement Batasi Win (win streak breaker): pause after N consecutive wins (default: 5) | P2 |
| FR-42 | Risk Management | System SHALL implement drawdown circuit breaker: halt if drawdown exceeds threshold (default: 10%) | P0 |
| FR-43 | Risk Management | System SHALL implement metabolic rate monitor: track CPU, memory, goroutine counts | P2 |
| FR-44 | Risk Management | System SHALL implement emergency stop: immediate halt on critical failures (API death spiral, data corruption) | P0 |
| FR-45 | Risk Management | Pit Boss SHALL be consulted before every trade; trades rejected if Pit Boss returns deny | P0 |
| FR-46 | Risk Management | System SHALL maintain risk state in Redis for cross-component access | P0 |
| FR-47 | Risk Management | System SHALL log all risk decisions (approve/deny) with full context | P0 |
| FR-48 | Dashboard | Dashboard SHALL display portfolio overview: total capital, daily PnL, total PnL, utilization rate | P0 |
| FR-49 | Dashboard | Dashboard SHALL display all active positions with real-time PnL | P0 |
| FR-50 | Dashboard | Dashboard SHALL display live opportunity feed (detected and executed) | P1 |
| FR-51 | Dashboard | Dashboard SHALL display risk status: daily budget, drawdown, win streak, circuit breaker status | P0 |
| FR-52 | Dashboard | Dashboard SHALL display system health: connection status, CPU, memory, error rate | P1 |
| FR-53 | Dashboard | Dashboard SHALL provide quick actions: emergency stop, pause/resume, risk param adjustment | P0 |
| FR-54 | Dashboard | Dashboard SHALL be responsive (desktop-first, tablet-compatible) | P1 |
| FR-55 | Dashboard | Dashboard SHALL support dark mode | P2 |
| FR-56 | Analytics | Analytics SHALL calculate and display PnL by: day, week, month, strategy, market | P1 |
| FR-57 | Analytics | Analytics SHALL calculate win rate, average win/loss, profit factor, Sharpe ratio | P1 |
| FR-58 | Analytics | Analytics SHALL calculate max drawdown, current drawdown, VaR (95%) | P1 |
| FR-59 | Analytics | Analytics SHALL visualize PnL over time (line chart), distribution (histogram), by strategy (pie) | P2 |
| FR-60 | Analytics | Analytics SHALL export data to CSV for external analysis | P2 |
| FR-61 | Analytics | Analytics SHALL detect performance anomalies (sudden drop in win rate, unusual drawdown) | P2 |
| FR-62 | Trade History | History SHALL record every trade with: timestamp, market, side, price, quantity, fill status, PnL, strategy, latency | P0 |
| FR-63 | Trade History | History SHALL support filtering by: date range, market, strategy, side, PnL sign | P1 |
| FR-64 | Trade History | History SHALL support export to CSV and JSON | P2 |
| FR-65 | Trade History | History SHALL be immutable (no edits, no deletes) | P0 |
| FR-66 | Orderbook Viewer | Viewer SHALL display real-time orderbook for selected market (bids, asks, spread) | P2 |
| FR-67 | Orderbook Viewer | Viewer SHALL display orderbook depth chart (cumulative bid/ask at each price level) | P2 |
| FR-68 | Orderbook Viewer | Viewer SHALL display recent trades (last 100) with price, size, timestamp | P2 |
| FR-69 | Orderbook Viewer | Viewer SHALL support multiple market tabs (up to 5 simultaneous) | P2 |
| FR-70 | Strategy Manager | Manager SHALL support CRUD operations for strategies | P1 |
| FR-71 | Strategy Manager | Manager SHALL support strategy activation/deactivation without restart | P1 |
| FR-72 | Strategy Manager | Manager SHALL validate strategy parameters (thresholds, limits, weights) | P1 |
| FR-73 | Strategy Manager | Manager SHALL support strategy versioning (track parameter changes) | P2 |
| FR-74 | Strategy Manager | Manager SHALL assign capital allocation weights to each active strategy | P1 |
| FR-75 | AI Strategy Optimizer | Optimizer SHALL analyze trade history and identify patterns in winning vs losing trades | P3 |
| FR-76 | AI Strategy Optimizer | Optimizer SHALL suggest parameter adjustments with expected impact | P3 |
| FR-77 | AI Strategy Optimizer | Optimizer SHALL require manual approval before applying suggestions | P3 |
| FR-78 | AI Strategy Optimizer | Optimizer SHALL A/B test suggestions in paper trading before recommending for live | P3 |
| FR-79 | AI Strategy Optimizer | Optimizer SHALL detect overfitting and warn when suggestions are too specific to historical data | P3 |
| FR-80 | Notification Center | Center SHALL send notifications via Telegram (primary) and email (secondary) | P0 |
| FR-81 | Notification Center | Center SHALL categorize notifications: critical, warning, info, debug | P0 |
| FR-82 | Notification Center | Center SHALL support notification throttling (max 10 per minute for non-critical) | P1 |
| FR-83 | Notification Center | Center SHALL support configurable notification preferences (enable/disable per category) | P1 |
| FR-84 | Notification Center | Center SHALL maintain notification history (last 1000) | P2 |
| FR-85 | Backtesting | Backtesting SHALL replay historical market data through strategy engine | P2 |
| FR-86 | Backtesting | Backtesting SHALL simulate realistic execution: slippage (configurable), partial fills, latency (configurable) | P2 |
| FR-87 | Backtesting | Backtesting SHALL calculate all performance metrics (PnL, win rate, Sharpe, drawdown, etc.) | P2 |
| FR-88 | Backtesting | Backtesting SHALL support parameter sweeps (test multiple configurations in batch) | P2 |
| FR-89 | Backtesting | Backtesting SHALL generate detailed reports with charts and trade-by-trade breakdown | P2 |
| FR-90 | Backtesting | Backtesting SHALL detect lookahead bias (using future data) and prevent it | P2 |
| FR-91 | Paper Trading | Paper trading SHALL use real market data but simulated execution | P2 |
| FR-92 | Paper Trading | Paper trading SHALL simulate fills based on real orderbook depth | P2 |
| FR-93 | Paper Trading | Paper trading SHALL track simulated PnL independently from live PnL | P2 |
| FR-94 | Paper Trading | Paper trading SHALL support seamless switch to live trading (same strategy config) | P2 |
| FR-95 | Paper Trading | Paper trading SHALL log all simulated trades with same detail as live trades | P2 |
| FR-96 | Replay Mode | Replay SHALL replay historical market events at configurable speed (1x, 2x, 5x, 10x) | P2 |
| FR-97 | Replay Mode | Replay SHALL display bot decisions in real-time (what it detected, what it decided, why) | P2 |
| FR-98 | Replay Mode | Replay SHALL support pause, step-forward, and rewind | P2 |
| FR-99 | Replay Mode | Replay SHALL highlight risk events (circuit breaker triggers, limit breaches) | P2 |
| FR-100 | AI Assistant | Assistant SHALL answer questions about trading performance using actual data | P3 |
| FR-101 | AI Assistant | Assistant SHALL explain bot decisions by referencing trade logs and decision context | P3 |
| FR-102 | AI Assistant | Assistant SHALL suggest risk parameter adjustments based on current state | P3 |
| FR-103 | AI Assistant | Assistant SHALL NOT execute trades or modify configurations directly | P3 |
| FR-104 | Multi-Strategy | System SHALL support running multiple strategies simultaneously | P1 |
| FR-105 | Multi-Strategy | System SHALL isolate strategy failures (one strategy crash doesn't affect others) | P1 |
| FR-106 | Multi-Strategy | System SHALL enforce per-strategy capital allocation | P1 |
| FR-107 | Multi-Strategy | System SHALL aggregate strategy performance for portfolio-level metrics | P1 |
| FR-108 | Multi-Account | System SHALL support multiple Polymarket wallet configurations | P3 |
| FR-109 | Multi-Account | System SHALL isolate account state (positions, PnL, risk) per wallet | P3 |
| FR-110 | Multi-Account | System SHALL support cross-account portfolio view (aggregate all accounts) | P3 |
| FR-111 | Multi-Account | System SHALL enforce risk limits per account independently | P3 |
| FR-112 | Admin Panel | Panel SHALL provide system configuration interface (API keys, risk defaults, notification settings) | P2 |
| FR-113 | Admin Panel | Panel SHALL provide system health dashboard (CPU, memory, disk, network, connections) | P2 |
| FR-114 | Admin Panel | Panel SHALL provide log viewer with filtering and search | P2 |
| FR-115 | Admin Panel | Panel SHALL provide database management (backup, restore, cleanup) | P2 |
| FR-116 | Admin Panel | Panel SHALL require authentication (even for single user) | P2 |

### NonFunctional Requirements

| ID | Feature | Requirement | Target |
|----|---------|-------------|--------|
| NFR-S1 | Market Scanner | Price update processing latency | Within 50ms of receipt |
| NFR-S2 | Market Scanner | Concurrent market subscription throughput | 500+ markets |
| NFR-S3 | Market Scanner | WebSocket connection uptime | 99.9% |
| NFR-S4 | Market Scanner | Market catalog memory footprint | ≤512MB for 1000 markets |
| NFR-A1 | Arbitrage Engine | Opportunity detection latency | Within 100ms of price update |
| NFR-A2 | Arbitrage Engine | Zero false negatives for profitable YES+NO arb | 100% detection |
| NFR-A3 | Arbitrage Engine | Score calculation determinism | Reproducible |
| NFR-E1 | Execution Engine | Order placement latency | Within 200ms of decision |
| NFR-E2 | Execution Engine | Order placement success rate | 99.9% (excluding API outages) |
| NFR-E3 | Execution Engine | YES+NO atomic execution window | Within 500ms |
| NFR-E4 | Execution Engine | Zero duplicate orders | Under any failure scenario |
| NFR-P1 | Position Manager | Position state accuracy vs Polymarket API | Within 1% |
| NFR-P2 | Position Manager | PnL update latency | Within 1s of price change |
| NFR-P3 | Position Manager | State mismatch detection | Within 60s |
| NFR-PM1 | Portfolio Manager | Capital calculation accuracy | $0.01 |
| NFR-PM2 | Portfolio Manager | Allocation sum consistency | Always 100% |
| NFR-PM3 | Portfolio Manager | Limit enforcement latency | Within 100ms of order attempt |
| NFR-R1 | Risk Management | Risk check latency | Within 10ms of trade request |
| NFR-R2 | Risk Management | Risk state consistency | Via Redis across all components |
| NFR-R3 | Risk Management | Pit Boss availability | 99.99% |
| NFR-R4 | Risk Management | Risk decision auditability | Complete log with full context |
| NFR-D1 | Dashboard | Real-time update latency | Within 2s |
| NFR-D2 | Dashboard | Page load performance | <3s on 3G |
| NFR-D3 | Dashboard | Accessibility | WCAG 2.1 AA |
| NFR-AN1 | Analytics | Financial calculation accuracy | $0.01 |
| NFR-AN2 | Analytics | Chart render performance | Within 2s for 1 year data |
| NFR-AN3 | Analytics | Time-series storage | TimescaleDB |
| NFR-TH1 | Trade History | Query response time | <1s for 10k trades |
| NFR-TH2 | Trade History | Data retention | Minimum 3 years |
| NFR-TH3 | Trade History | Storage | PostgreSQL with proper indexing |
| NFR-OV1 | Orderbook Viewer | Orderbook update latency | Within 100ms |
| NFR-OV2 | Orderbook Viewer | Memory per market tab | <100MB |
| NFR-SM1 | Strategy Manager | Config persistence | PostgreSQL |
| NFR-SM2 | Strategy Manager | Parameter validation | Before save |
| NFR-SM3 | Strategy Manager | Version history | Complete with rollback |
| NFR-AI1 | AI Strategy Optimizer | Statistical significance | p < 0.05 |
| NFR-AI2 | AI Strategy Optimizer | Safety | No auto-application; manual approval |
| NFR-AI3 | AI Strategy Optimizer | Validation | A/B testing in paper trading |
| NFR-N1 | Notification Center | Critical notification latency | Within 5s |
| NFR-N2 | Notification Center | Critical notification delivery rate | 99.9% |
| NFR-N3 | Notification Center | Non-critical throttling | Max 10/min |
| NFR-BT1 | Backtesting | 1-year data backtest time | <10 minutes |
| NFR-BT2 | Backtesting | Determinism | Same input → same output |
| NFR-BT3 | Backtesting | Simulation accuracy | Within 10% of live behavior |
| NFR-PT1 | Paper Trading | Isolation | Never affects live positions/capital |
| NFR-PT2 | Paper Trading | Fill simulation realism | Within 10% of actual behavior |
| NFR-PT3 | Paper Trading | Live/paper switch time | Within 1s |
| NFR-RP1 | Replay Mode | Replay accuracy | Matches historical data exactly |
| NFR-RP2 | Replay Mode | Playback performance | Smooth at 10x speed |
| NFR-RP3 | Replay Mode | Control responsiveness | Within 100ms |
| NFR-AA1 | AI Assistant | Numerical accuracy | Verified against database |
| NFR-AA2 | AI Assistant | Safety | Read-only access |
| NFR-AA3 | AI Assistant | Response latency | Within 5s for simple queries |
| NFR-MS1 | Multi-Strategy | Failure isolation | No cascade to other strategies |
| NFR-MS2 | Multi-Strategy | Resource budget | Configurable CPU/memory per strategy |
| NFR-MS3 | Multi-Strategy | Portfolio metric consistency | Consistent with strategy-level metrics |
| NFR-MA1 | Multi-Account | Account state isolation | Completely independent |
| NFR-MA2 | Multi-Account | Wallet key security | Encrypted at rest; never shared |
| NFR-MA3 | Multi-Account | Cross-account aggregation | Accurate and performant |
| NFR-AP1 | Admin Panel | Authentication security | Session timeout; CSRF protection |
| NFR-AP2 | Admin Panel | Log search performance | <1s for 1M entries |
| NFR-AP3 | Admin Panel | Backup automation | Daily with 30-day retention |

### Additional Requirements

| ID | Source | Category | Requirement |
|----|--------|----------|-------------|
| AD-1 | Architecture | Service Boundary | Scanner is the sole producer of market data events; no other component may subscribe directly to Polymarket APIs; all market data flows through Scanner → NATS → consumers |
| AD-2 | Architecture | Service Boundary | Arb Engine is a pure function of market state → scored opportunities; it never executes and never modifies market state |
| AD-3 | Architecture | Service Boundary | Execution Engine is the sole writer to Polymarket CLOB API; every order gets a unique client order ID (UUID) for idempotency |
| AD-4 | Architecture | Risk Authority | Pit Boss is the sole authority on whether a trade may proceed; lives as risk state keys in Redis with 60s TTL; Risk Management is sole writer |
| AD-5 | Architecture | State Reconciliation | Three continuous reconciliation loops: market data (after reconnect), position (every 60s), order (every 30s); persistent mismatches (>3 consecutive) trigger emergency stop |
| AD-6 | Architecture | Data Ownership | PostgreSQL single-writer per table: trades (Execution Engine), strategies (Strategy Manager), positions (Position Manager), risk_events (Risk Management), accounts (Portfolio Manager), markets (Scanner) |
| AD-7 | Architecture | Data Ownership | TimescaleDB for time-series only: market_prices (1yr raw, 5yr aggregated), opportunities (3yr), system_metrics (90d raw, 2yr aggregated); analytics queries run against TimescaleDB only |
| AD-8 | Architecture | Data Ownership | Redis is ephemeral cache/coordination only; Pit Boss state (TTL 60s, refreshed every 30s), market catalog cache (TTL 5s), session state; all Redis state reconstructable from PostgreSQL |
| AD-9 | Architecture | Event Bus | NATS is primary event bus with defined subject hierarchy; fire-and-forget with at-least-once delivery; consumers idempotent by event UUID; JetStream for durable subscriptions (order fills, risk alerts) |
| AD-10 | Architecture | Communication | Sync patterns: Execution → Pit Boss (Redis GET, <10ms), Scanner → Polymarket WS, Execution → CLOB API; Async patterns: Scanner → NATS, Arb → NATS, Execution → NATS |
| AD-11 | Architecture | Error Handling | Circuit breaker (closed/open/half-open) on all external calls; global emergency stop on: API unreachable >5min, data corruption, daily budget exhausted, drawdown breaker tripped |
| AD-12 | Architecture | Execution Mode | Global execution_mode enum (LIVE/PAPER/REPLAY) in Redis; PAPER mode uses simulated fills from real orderbook; separate paper_positions table; mode switch requires restart |
| AD-13 | Architecture | Strategy Isolation | Each strategy as logical goroutine group; separate capital allocation, risk limits, position tracking, performance metrics; panic recovery without service crash |
| AD-14 | Architecture | Security | Secrets in Kubernetes Secrets (encrypted at rest); injected as env vars, never mounted as files; never logged; JWT auth on Dashboard/Admin Panel |
| AD-15 | Architecture | Deployment | Single Kubernetes namespace: scanner, execution-engine, risk-manager, position-manager, arb-engine (Go), portfolio-manager, analytics, api-gateway, notification (Python), dashboard (Next.js), redis, postgresql, nats |
| AD-16 | Architecture | Capital Scaling | Capital tier derived from total capital; tier determines active strategies, max position size %, risk budget; promotion requires 7 consecutive days above threshold; demotion immediate |
| AD-17 | Architecture | Observability | Prometheus metrics on /metrics for all services; latency histograms, counters, gauges, system metrics; Grafana dashboards; structured JSON logs to stdout |
| AD-18 | Architecture | Migrations | golang-migrate (Go) and Alembic (Python) for schema management; version-controlled in migrations/; forward-only in production; applied by init containers; never destructive |
| DEP-1 | Architecture | Deployment | Docker containers for all services with individual Dockerfiles |
| DEP-2 | Architecture | Deployment | Kubernetes manifests: deployments, statefulsets, configmaps, secrets templates |
| DEP-3 | Architecture | Deployment | docker-compose.yaml for local development |
| DEP-4 | Architecture | Deployment | Makefile for build, test, lint commands |
| DEP-5 | Architecture | Infrastructure | Prometheus + Grafana + Alertmanager for monitoring stack |
| DEP-6 | Architecture | Infrastructure | Grafana dashboards: trading-overview, risk-monitor, system-health, strategy-performance |
| INF-1 | Architecture | Tech Stack | Go 1.26.4 for execution-critical services (scanner, arb-engine, execution-engine, risk-manager, position-manager) |
| INF-2 | Architecture | Tech Stack | Python 3.13.14 for AI/analytics services (portfolio-manager, analytics, backtest, ai-optimizer, notification) |
| INF-3 | Architecture | Tech Stack | FastAPI 0.139.0 for API Gateway |
| INF-4 | Architecture | Tech Stack | Next.js 16.2.10 (LTS) for Dashboard |
| INF-5 | Architecture | Tech Stack | Redis 8.8.0 for cache/coordination |
| INF-6 | Architecture | Tech Stack | PostgreSQL 17.10 for OLTP |
| INF-7 | Architecture | Tech Stack | TimescaleDB 2.x on PG 17 for time-series |
| INF-8 | Architecture | Tech Stack | NATS 2.10+ (JetStream) for event bus |
| INF-9 | Architecture | Tech Stack | Prometheus 3.12.0 for metrics |
| INF-10 | Architecture | Tech Stack | Grafana 13.0.3 for dashboards |
| INF-11 | Architecture | Conventions | Decimal precision: all monetary values use Decimal (never float64); prices 4dp, quantities 8dp, PnL 8dp |
| INF-12 | Architecture | Conventions | All timestamps UTC as TIMESTAMPTZ; display timezone configurable |
| INF-13 | Architecture | Conventions | Typed errors (ErrInsufficientBalance, ErrRiskDenied, ErrSlippageExceeded) |
| INF-14 | Architecture | Conventions | Structured JSON logs with timestamp, level, service, request_id, message, context |
| INF-15 | Architecture | Conventions | Prometheus metric naming: pqap_{service}_{metric_name}_{unit} |
| INF-16 | Architecture | Conventions | Event naming: past tense verb + noun (MarketPriceUpdated, OrderFilled) |
| INF-17 | Architecture | Conventions | All events include: event_id (UUID), event_type, timestamp (ISO 8601 UTC), source, payload |
| INF-18 | Architecture | Schema Design | Include account_id as nullable column in all relevant tables from day one for future multi-account support |

### UX Design Requirements

No UX design document exists yet. Dashboard requirements are defined in FR-48 through FR-55 and FR-66 through FR-69 (Orderbook Viewer).

### FR Coverage Map

| FR | Epic | Description |
|----|------|-------------|
| FR-1 | Epic 1 | Scanner: WebSocket connection to Polymarket |
| FR-2 | Epic 1 | Scanner: Internal market catalog |
| FR-3 | Epic 1 | Scanner: New market detection (60s) |
| FR-4 | Epic 1 | Scanner: Stale market detection (30s) |
| FR-5 | Epic 1 | Scanner: Auto reconnect with backoff |
| FR-6 | Epic 1 | Scanner: State reconciliation after reconnect |
| FR-7 | Epic 1 | Scanner: Batch REST API calls |
| FR-8 | Epic 2 | Scanner: Export metrics (Prometheus) |
| FR-9 | Epic 1 | Arb Engine: Simple YES+NO arbitrage detection |
| FR-10 | Epic 3 | Arb Engine: Cross-market arbitrage detection |
| FR-11 | Epic 1 | Arb Engine: Opportunity scoring |
| FR-12 | Epic 1 | Arb Engine: Score threshold filtering |
| FR-13 | Epic 1 | Arb Engine: Fill probability estimation |
| FR-14 | Epic 3 | Arb Engine: Near-resolution detection |
| FR-15 | Epic 3 | Arb Engine: Correlated market identification |
| FR-16 | Epic 3 | Arb Engine: Opportunity logging for backtesting |
| FR-17 | Epic 1 | Execution: Limit orders (GTC) |
| FR-18 | Epic 1 | Execution: Risk budget validation |
| FR-19 | Epic 1 | Execution: Slippage protection |
| FR-20 | Epic 1 | Execution: Partial fill handling |
| FR-21 | Epic 1 | Execution: Circuit breaker (5 errors) |
| FR-22 | Epic 1 | Execution: Idempotent order placement |
| FR-23 | Epic 1 | Execution: Atomic YES+NO execution |
| FR-24 | Epic 1 | Execution: Order audit trail |
| FR-25 | Epic 1 | Position: Track open positions |
| FR-26 | Epic 1 | Position: API reconciliation (60s) |
| FR-27 | Epic 1 | Position: Market resolution settlement |
| FR-28 | Epic 1 | Position: Unrealized PnL calculation |
| FR-29 | Epic 1 | Position: Limit breach alerts |
| FR-30 | Epic 1 | Position: Manual exit |
| FR-31 | Epic 1 | Portfolio: Capital tracking |
| FR-32 | Epic 3 | Portfolio: Strategy weight allocation |
| FR-33 | Epic 1 | Portfolio: Per-strategy position limits |
| FR-34 | Epic 1 | Portfolio: Per-market position limits |
| FR-35 | Epic 3 | Portfolio: Auto tier adjustment |
| FR-36 | Epic 3 | Portfolio: Capital utilization rate |
| FR-37 | Epic 3 | Portfolio: Manual rebalancing |
| FR-38 | Epic 1 | Risk: Daily loss limit (2%) |
| FR-39 | Epic 1 | Risk: Max position per market (10%) |
| FR-40 | Epic 2 | Risk: Max correlated positions |
| FR-41 | Epic 2 | Risk: Batasi Win (win streak breaker) |
| FR-42 | Epic 1 | Risk: Drawdown circuit breaker (10%) |
| FR-43 | Epic 2 | Risk: Metabolic rate monitor |
| FR-44 | Epic 1 | Risk: Emergency stop |
| FR-45 | Epic 1 | Risk: Pit Boss consultation |
| FR-46 | Epic 1 | Risk: Redis risk state |
| FR-47 | Epic 1 | Risk: Risk decision logging |
| FR-48 | Epic 2 | Dashboard: Portfolio overview |
| FR-49 | Epic 2 | Dashboard: Active positions |
| FR-50 | Epic 2 | Dashboard: Live opportunity feed |
| FR-51 | Epic 2 | Dashboard: Risk status |
| FR-52 | Epic 2 | Dashboard: System health |
| FR-53 | Epic 2 | Dashboard: Quick actions |
| FR-54 | Epic 2 | Dashboard: Responsive design |
| FR-55 | Epic 2 | Dashboard: Dark mode |
| FR-56 | Epic 4 | Analytics: PnL by day/week/month/strategy/market |
| FR-57 | Epic 4 | Analytics: Win rate, Sharpe, profit factor |
| FR-58 | Epic 4 | Analytics: Drawdown, VaR |
| FR-59 | Epic 4 | Analytics: Charts (line, histogram, pie) |
| FR-60 | Epic 4 | Analytics: CSV export |
| FR-61 | Epic 4 | Analytics: Anomaly detection |
| FR-62 | Epic 1 | Trade History: Record every trade |
| FR-63 | Epic 4 | Trade History: Filtering |
| FR-64 | Epic 4 | Trade History: CSV/JSON export |
| FR-65 | Epic 1 | Trade History: Immutable records |
| FR-66 | Epic 4 | Orderbook: Real-time display |
| FR-67 | Epic 4 | Orderbook: Depth chart |
| FR-68 | Epic 4 | Orderbook: Recent trades |
| FR-69 | Epic 4 | Orderbook: Multiple market tabs |
| FR-70 | Epic 3 | Strategy Manager: CRUD operations |
| FR-71 | Epic 3 | Strategy Manager: Activation/deactivation |
| FR-72 | Epic 3 | Strategy Manager: Parameter validation |
| FR-73 | Epic 3 | Strategy Manager: Versioning |
| FR-74 | Epic 3 | Strategy Manager: Capital allocation weights |
| FR-75 | Epic 6 | AI Optimizer: Pattern analysis |
| FR-76 | Epic 6 | AI Optimizer: Parameter suggestions |
| FR-77 | Epic 6 | AI Optimizer: Manual approval |
| FR-78 | Epic 6 | AI Optimizer: A/B testing in paper |
| FR-79 | Epic 6 | AI Optimizer: Overfitting detection |
| FR-80 | Epic 1 | Notification: Telegram delivery |
| FR-81 | Epic 1 | Notification: Severity categorization |
| FR-82 | Epic 2 | Notification: Throttling |
| FR-83 | Epic 2 | Notification: Configurable preferences |
| FR-84 | Epic 2 | Notification: History (last 1000) |
| FR-85 | Epic 5 | Backtesting: Historical data replay |
| FR-86 | Epic 5 | Backtesting: Realistic execution simulation |
| FR-87 | Epic 5 | Backtesting: Performance metrics |
| FR-88 | Epic 5 | Backtesting: Parameter sweeps |
| FR-89 | Epic 5 | Backtesting: Detailed reports |
| FR-90 | Epic 5 | Backtesting: Lookahead bias detection |
| FR-91 | Epic 5 | Paper Trading: Real data, simulated execution |
| FR-92 | Epic 5 | Paper Trading: Realistic fill simulation |
| FR-93 | Epic 5 | Paper Trading: Separate PnL tracking |
| FR-94 | Epic 5 | Paper Trading: Seamless switch to live |
| FR-95 | Epic 5 | Paper Trading: Identical trade logging |
| FR-96 | Epic 5 | Replay: Configurable speed (1x-10x) |
| FR-97 | Epic 5 | Replay: Bot decision display |
| FR-98 | Epic 5 | Replay: Pause/step/rewind controls |
| FR-99 | Epic 5 | Replay: Risk event highlighting |
| FR-100 | Epic 6 | AI Assistant: Performance Q&A |
| FR-101 | Epic 6 | AI Assistant: Decision explanation |
| FR-102 | Epic 6 | AI Assistant: Risk parameter suggestions |
| FR-103 | Epic 6 | AI Assistant: Read-only (no execution) |
| FR-104 | Epic 3 | Multi-Strategy: Simultaneous execution |
| FR-105 | Epic 3 | Multi-Strategy: Failure isolation |
| FR-106 | Epic 3 | Multi-Strategy: Per-strategy capital allocation |
| FR-107 | Epic 3 | Multi-Strategy: Portfolio aggregation |
| FR-108 | Epic 7 | Multi-Account: Wallet configurations |
| FR-109 | Epic 7 | Multi-Account: Account state isolation |
| FR-110 | Epic 7 | Multi-Account: Cross-account view |
| FR-111 | Epic 7 | Multi-Account: Per-account risk limits |
| FR-112 | Epic 7 | Admin: System configuration |
| FR-113 | Epic 7 | Admin: Health dashboard |
| FR-114 | Epic 7 | Admin: Log viewer |
| FR-115 | Epic 7 | Admin: Database management |
| FR-116 | Epic 2 | Admin: Authentication |

## Epic List

### Epic 1: Foundation — Bot Can Hunt
Bot dapat terhubung ke Polymarket, mendeteksi arbitrage sederhana, mengeksekusi trade pertama, dan melindungi modal dengan risk management dasar.

**FRs covered:** FR-1,2,3,4,5,6,7,9,11,12,13,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,33,34,38,39,42,44,45,46,47,62,65,80,81 (39 FRs)

**Services:** scanner, arb-engine, execution-engine, position-manager, risk-manager (core), notification

---

### Epic 2: Risk Shield & Monitoring
Bot memiliki perlindungan risiko lengkap dan user dapat memantau semuanya dari dashboard real-time.

**FRs covered:** FR-8,40,41,43,48,49,50,51,52,53,54,55,82,83,84,116 (16 FRs)

**Services:** risk-manager (advanced), dashboard, api-gateway, notification (advanced)

---

### Epic 3: Advanced Hunting Strategies
Bot dapat menangkap peluang lebih kompleks: cross-market arbitrage, liquidity capture, multi-strategy, dan strategy management.

**FRs covered:** FR-10,14,15,16,32,35,36,37,70,71,72,73,74,104,105,106,107 (17 FRs)

**Services:** arb-engine (advanced), portfolio-manager, strategy-manager

---

### Epic 4: Intelligence & Analytics
User dapat menganalisis performa trading, melihat orderbook, dan mengekspor data untuk analisis eksternal.

**FRs covered:** FR-56,57,58,59,60,61,63,64,66,67,68,69 (12 FRs)

**Services:** analytics, trade-history (advanced), orderbook-viewer

---

### Epic 5: Backtesting & Paper Trading
User dapat menguji strategi dengan data historis, simulasi paper trading, dan replay mode untuk debugging.

**FRs covered:** FR-85,86,87,88,89,90,91,92,93,94,95,96,97,98,99 (15 FRs)

**Services:** backtest, paper-trading, replay-mode

---

### Epic 6: AI Strategy Optimization
Bot dapat belajar dari trade history, menyarankan optimasi parameter, dan user dapat berinteraksi via AI assistant.

**FRs covered:** FR-75,76,77,78,79,100,101,102,103 (9 FRs)

**Services:** ai-optimizer, ai-assistant

---

### Epic 7: Scaling & Enterprise
User dapat menjalankan multiple account dan mengelola sistem melalui admin panel.

**FRs covered:** FR-108,109,110,111,112,113,114,115 (8 FRs)

**Services:** multi-account, admin-panel

---

**Total: 7 Epics, 116 FRs ter-cover**

---

## Stories

### Epic 1: Foundation — Bot Can Hunt

---

### Story 1.1: Scanner WebSocket Connection & Market Catalog

As a quant trader,
I want the bot to connect to Polymarket via WebSocket and maintain a real-time market catalog,
So that I never miss an arbitrage opportunity due to stale or missing market data.

**Acceptance Criteria:**

**Given** the scanner service starts and Polymarket WebSocket API is available
**When** the scanner connects and subscribes to active binary markets
**Then** a WebSocket connection is established within 5 seconds
**And** the internal market catalog is populated with market ID, slug, current YES/NO prices, spread, volume, and liquidity depth for all active markets
**And** the catalog is updated within 100ms of every price change from the WebSocket stream
**And** new markets are detected and added to the catalog within 60 seconds of appearing on Polymarket
**And** market data events (`MarketPriceUpdated`, `MarketDiscovered`) are published to NATS for downstream consumers

**References:** FR-1, FR-2, FR-3, AD-1, NFR-S1, NFR-S2

---

### Story 1.2: Market Scanner — Stale Detection, Reconnect & Batching

As a quant trader,
I want the scanner to detect stale markets and automatically reconnect with state reconciliation,
So that downstream components never act on outdated market data.

**Acceptance Criteria:**

**Given** the scanner is connected and tracking markets
**When** a market receives no price updates for 30 seconds (configurable threshold)
**Then** the market is flagged as "stale" in the catalog
**And** a `MarketStale` event is published to NATS
**And** stale markets are excluded from opportunity scoring by the arb engine

**Given** the WebSocket connection drops
**When** the disconnect is detected
**Then** the scanner attempts reconnection with exponential backoff (initial: 1s, max: 60s)
**And** after reconnection, a full orderbook snapshot is fetched via REST API for all tracked markets
**And** internal state is reconciled against the snapshot — prices differing by more than 1 tick trigger an alert
**And** REST API calls are batched (up to 100 markets per request) to minimize API usage

**References:** FR-4, FR-5, FR-6, FR-7, AD-1, AD-5, NFR-S3

---

### Story 1.3: Arbitrage Detection & Opportunity Scoring

As a quant trader,
I want the bot to detect simple YES+NO arbitrage opportunities and score them,
So that only high-quality opportunities are passed to the execution engine.

**Acceptance Criteria:**

**Given** the arb engine is subscribed to `MarketPriceUpdated` events via NATS
**When** a price update shows YES_price + NO_price < $1.00 - min_profit_threshold
**Then** an opportunity is detected within 100ms of the price update
**And** the opportunity score is calculated as: spread × liquidity × fill_probability
**And** fill probability is estimated based on orderbook depth and historical fill rates
**And** opportunities with score below the configurable threshold (default: 0.01) are logged but not emitted to execution
**And** opportunities above threshold emit an `OpportunityDetected` event to NATS
**And** all detected opportunities (including filtered) are logged to TimescaleDB for backtesting analysis

**References:** FR-9, FR-11, FR-12, FR-13, FR-16, AD-2, NFR-A1, NFR-A2, NFR-A3

---

### Story 1.4: Execution Engine — Order Placement & Slippage Protection

As a quant trader,
I want the execution engine to place limit orders with slippage protection and risk validation,
So that trades are executed at favorable prices and never exceed my risk budget.

**Acceptance Criteria:**

**Given** the execution engine receives an `OpportunityDetected` event
**When** the engine prepares to execute the trade
**Then** a synchronous risk check is performed against the Pit Boss (Redis GET, < 10ms)
**And** if Pit Boss returns DENY, the order is rejected and logged with the denial reason
**And** if approved, a GTC limit order is placed via Polymarket CLOB API within 200ms of the decision
**And** every order gets a unique client order ID (UUID) for idempotency — no duplicate orders under any failure scenario
**And** if the price moves beyond slippage tolerance (default: 1%) before placement, the trade is rejected
**And** partial fills are tracked: filled quantity logged, remaining quantity handled per strategy (cancel or wait)
**And** every order attempt is logged with: timestamp, market, side, price, size, result, latency
**And** on fill, an `OrderFilled` event is published; on failure, `OrderFailed` is published

**References:** FR-17, FR-18, FR-19, FR-20, FR-22, FR-24, AD-3, AD-10, NFR-E1, NFR-E2, NFR-E4

---

### Story 1.5: Execution Engine — Atomic YES+NO & Circuit Breaker

As a quant trader,
I want YES+NO arbitrage legs executed atomically and a circuit breaker to halt on repeated API failures,
So that I never end up with a half-filled arb position or blow up during an API outage.

**Acceptance Criteria:**

**Given** a YES+NO arbitrage opportunity is approved for execution
**When** the execution engine places both legs
**Then** both orders are placed within a 500ms window
**And** if one leg fails, the other is cancelled within 1 second
**And** if one leg is partially filled and the other fails, the partial fill is tracked and logged

**Given** the Polymarket CLOB API returns consecutive errors
**When** the error count reaches the configurable threshold (default: 5)
**Then** the circuit breaker trips and all trading is halted
**And** an alert is sent via Telegram (critical notification, bypasses throttling)
**And** trading remains halted until manual resume is initiated by the user
**And** the circuit breaker state is logged and queryable

**References:** FR-21, FR-23, AD-3, AD-11, NFR-E3

---

### Story 1.6: Position Tracking & PnL Calculation

As a quant trader,
I want the bot to track all open positions with real-time PnL and reconcile with Polymarket,
So that I always know my true exposure and profit/loss.

**Acceptance Criteria:**

**Given** an `OrderFilled` event is received
**When** the position manager processes it
**Then** a new position is created with: market, side, entry price, current price, quantity, unrealized PnL
**And** the position is stored in the PostgreSQL `positions` table

**Given** positions are open and prices are updating
**When** a `MarketPriceUpdated` event is received
**Then** unrealized PnL is recalculated within 1 second using current market prices
**And** if position exceeds configured limits, an alert is sent within 5 seconds

**Given** the position manager is running
**When** every 60 seconds elapse
**Then** position state is reconciled with the Polymarket API
**And** any discrepancy is detected, alerted, and logged
**And** persistent mismatches (>3 consecutive) trigger emergency stop

**Given** a market resolution is detected
**When** the position manager processes it
**Then** the position is automatically settled, PnL is finalized, and the position moves to history

**And** manual position exit (close at market) is supported — exit order placed within 1 second of command

**References:** FR-25, FR-26, FR-27, FR-28, FR-29, FR-30, AD-5, NFR-P1, NFR-P2, NFR-P3

---

### Story 1.7: Risk Management — Pit Boss & Daily Budget

As a quant trader,
I want a centralized Pit Boss that enforces daily loss limits and position limits before every trade,
So that the bot never loses more than I can afford in a single day.

**Acceptance Criteria:**

**Given** the risk manager service is running
**When** it initializes and periodically (every 30 seconds)
**Then** Pit Boss risk state keys are written to Redis with a 60-second TTL
**And** the state includes: daily budget remaining, max position per market (default: 10% of capital), max position per strategy (default: 20% of capital)

**Given** the Pit Boss state is in Redis
**When** the execution engine performs a synchronous risk check before a trade
**Then** the check completes within 10ms
**And** if daily loss limit (default: 2% of capital) is reached, the Pit Boss returns DENY
**And** if per-market position limit would be exceeded, the Pit Boss returns DENY
**And** if per-strategy position limit would be exceeded, the Pit Boss returns DENY
**And** all risk decisions (approve/deny) are logged to PostgreSQL `risk_events` table with full context
**And** the Pit Boss state in Redis is reconstructable from PostgreSQL on restart

**References:** FR-38, FR-39, FR-45, FR-46, FR-47, AD-4, AD-8, NFR-R1, NFR-R2, NFR-R3, NFR-R4

---

### Story 1.8: Risk Management — Emergency Stop & Drawdown Breaker

As a quant trader,
I want an emergency stop and drawdown circuit breaker that halt all trading on critical failures,
So that a bad market day or system failure doesn't wipe out my capital.

**Acceptance Criteria:**

**Given** the risk manager is monitoring portfolio drawdown
**When** drawdown exceeds the configurable threshold (default: 10%)
**Then** all trading is halted immediately
**And** all open orders are cancelled
**And** a critical alert is sent via Telegram
**And** manual resume is required to restart trading

**Given** a critical failure is detected (API death spiral, data corruption detected by reconciliation, daily budget exhausted)
**When** the emergency stop is triggered
**Then** all trading is halted within 1 second
**And** all open orders are cancelled
**And** an `EmergencyStop` event is published to NATS
**And** a critical Telegram notification is sent (bypasses all throttling)
**And** the emergency stop reason and full context are logged

**References:** FR-42, FR-44, AD-5, AD-11

---

### Story 1.9: Trade History Recording

As a quant trader,
I want every trade attempt recorded immutably with full details,
So that I have a complete audit trail for analysis, debugging, and tax reporting.

**Acceptance Criteria:**

**Given** the execution engine processes an order (success or failure)
**When** the trade result is finalized
**Then** a record is written to the PostgreSQL `trades` table with: timestamp (UTC TIMESTAMPTZ), market, side, price, quantity, fill status, PnL, strategy ID, latency
**And** all required fields are populated — no NULL values for required fields
**And** the `trades` table is append-only (immutable) — no UPDATE or DELETE operations are permitted
**And** all monetary values use Decimal precision (prices: 4dp, quantities: 8dp, PnL: 8dp)
**And** the `account_id` column is included (nullable, default null) for future multi-account support

**References:** FR-62, FR-65, AD-6, INF-11, INF-12, INF-18, NFR-TH2, NFR-TH3

---

### Story 1.10: Telegram Notifications

As a quant trader,
I want to receive critical trading alerts via Telegram,
So that I'm immediately informed of emergency stops, circuit breakers, and significant trades.

**Acceptance Criteria:**

**Given** the notification service is configured with a Telegram bot token
**When** a notification event is published to NATS (`NotificationRequest`)
**Then** the notification is categorized by severity: critical, warning, info, debug
**And** critical notifications (emergency stop, circuit breaker trip, API failure) are delivered within 5 seconds
**And** critical notifications bypass any throttling rules
**And** the notification is delivered to the configured Telegram chat
**And** delivery is confirmed; failed deliveries are retried with backoff
**And** the notification is stored in PostgreSQL history table

**References:** FR-80, FR-81, NFR-N1, NFR-N2

---

### Epic 2: Risk Shield & Monitoring

---

### Story 2.1: Advanced Risk — Correlation Limits, Batasi Win & Metabolic Rate

As a quant trader,
I want correlation limits, win-streak breaker (Batasi Win), and system resource monitoring,
So that the bot avoids cascade risk from correlated positions, prevents overconfidence during win streaks, and stays within safe resource consumption.

**Acceptance Criteria:**

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

**References:** FR-40, FR-41, FR-43, AD-4

---

### Story 2.2: Dashboard — Portfolio Overview & Position Display

As a quant trader,
I want a real-time dashboard showing my portfolio overview and all active positions,
So that I can monitor my trading performance at a glance.

**Acceptance Criteria:**

**Given** the dashboard is loaded and the user is authenticated
**When** the portfolio overview page renders
**Then** it displays: total capital, daily PnL, total PnL, capital utilization rate
**And** data is accurate within 1% of backend values
**And** updates are pushed via WebSocket within 2 seconds of any change

**Given** there are active positions
**When** the positions view renders
**Then** all active positions are displayed with: market, side, entry price, current price, quantity, unrealized PnL
**And** PnL updates in real-time (within 2 seconds of price change)
**And** the position list matches the backend position manager state

**And** the dashboard is responsive — usable on 1024px+ screens with no horizontal scrolling
**And** page loads within 3 seconds on a 3G connection

**References:** FR-48, FR-49, FR-54, NFR-D1, NFR-D2, NFR-D3

---

### Story 2.3: Dashboard — Risk Status & Quick Actions

As a quant trader,
I want to see real-time risk status and have quick action controls on the dashboard,
So that I can monitor risk exposure and take immediate action when needed.

**Acceptance Criteria:**

**Given** the dashboard is loaded
**When** the risk status section renders
**Then** it displays: daily budget remaining, current drawdown, win streak count, circuit breaker status (open/closed)
**And** risk data matches the backend risk manager state
**And** updates are pushed within 2 seconds of any change

**Given** the dashboard quick actions are available
**When** the user triggers an action (emergency stop, pause/resume trading, adjust risk parameters)
**Then** the action executes within 1 second
**And** confirmation is required for critical actions (emergency stop, resume after circuit breaker)
**And** risk parameter adjustments are persisted and logged

**References:** FR-51, FR-53, NFR-D1

---

### Story 2.4: Dashboard — System Health & Opportunity Feed

As a quant trader,
I want to see system health metrics and a live feed of detected opportunities,
So that I can verify the bot is operating correctly and see what it's finding.

**Acceptance Criteria:**

**Given** the dashboard is loaded
**When** the system health section renders
**Then** it displays: WebSocket connection status, CPU usage, memory usage, error rate
**And** health metrics update every 5 seconds
**And** metrics are accurate within 10% of actual values

**Given** the arb engine is detecting opportunities
**When** the opportunity feed renders
**Then** it displays a live stream of detected and executed opportunities
**And** the feed updates within 1 second of opportunity detection
**And** historical opportunities are scrollable
**And** each entry shows: market, score, spread, timestamp, status (detected/executed/filtered)

**References:** FR-50, FR-52, NFR-D1

---

### Story 2.5: Notification Preferences & Throttling

As a quant trader,
I want to configure notification preferences and have non-critical notifications throttled,
So that I receive important alerts without notification spam.

**Acceptance Criteria:**

**Given** the notification center is running
**When** the user configures notification preferences (enable/disable per category: critical, warning, info, debug)
**Then** preferences are persisted and take effect immediately
**And** disabled categories are suppressed entirely

**Given** non-critical notifications are being generated rapidly
**When** the rate exceeds 10 per minute
**Then** non-critical notifications are throttled (max 10/min)
**And** critical notifications bypass throttling entirely
**And** notification history (last 1000) is maintained and queryable via API

**References:** FR-82, FR-83, FR-84, NFR-N3

---

### Story 2.6: Admin Authentication & Scanner Metrics Export

As a quant trader,
I want the admin panel and dashboard to require authentication, and the scanner to export Prometheus metrics,
So that my system is secure and I can monitor scanner performance.

**Acceptance Criteria:**

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

**References:** FR-8, FR-116, AD-14, AD-17, NFR-AP1

---

### Epic 3: Advanced Hunting Strategies

---

### Story 3.1: Cross-Market Arbitrage Detection

As a quant trader,
I want the arb engine to detect cross-market arbitrage between related markets,
So that I can capture more complex mispricings beyond simple YES+NO arb.

**Acceptance Criteria:**

**Given** the arb engine has market relationship data (e.g., "Will X happen?" vs "Will X by date Y?")
**When** related markets have a price inconsistency that creates a profitable opportunity
**Then** the cross-market arbitrage is detected and scored using the same scoring engine (spread × liquidity × fill_probability)
**And** at least 3 cross-market relationship types are supported
**And** false positive rate is below 10%
**And** the opportunity is logged with relationship context for backtesting

**Given** a market resolution is imminent (within 1 hour)
**When** the arb engine evaluates an opportunity for that market
**Then** the confidence score is reduced by 50% (configurable threshold)
**And** the near-resolution flag is included in the `OpportunityDetected` event

**References:** FR-10, FR-14, AD-2

---

### Story 3.2: Strategy Manager — CRUD & Activation

As a quant trader,
I want to create, read, update, and delete trading strategies and activate/deactivate them without restarting the bot,
So that I can dynamically adjust which strategies are running.

**Acceptance Criteria:**

**Given** the strategy manager API is available
**When** the user creates a new strategy with parameters (thresholds, position sizing, risk limits)
**Then** the strategy is persisted to the PostgreSQL `strategies` table
**And** all parameters are validated before save — invalid values rejected with clear error messages
**And** the `account_id` column is included (nullable) for future multi-account support

**Given** a strategy exists
**When** the user activates or deactivates it
**Then** the change takes effect within 1 second
**And** no other strategies are affected
**And** no service restart is required
**And** a `StrategyUpdated` event is published to NATS

**References:** FR-70, FR-71, FR-72, AD-6, INF-18, NFR-SM1, NFR-SM2

---

### Story 3.3: Strategy Manager — Versioning & Capital Allocation Weights

As a quant trader,
I want strategy parameter changes tracked with version history and each strategy assigned capital allocation weights,
So that I can roll back bad changes and control how capital is distributed.

**Acceptance Criteria:**

**Given** a strategy's parameters are modified
**When** the update is saved
**Then** a new version is created in the version history
**And** the previous version is preserved with timestamp and change summary
**And** rollback to any previous version is supported

**Given** multiple strategies are active
**When** capital allocation weights are assigned
**Then** weights sum to 100% (enforced by Portfolio Manager)
**And** weights are adjustable at runtime
**And** the arb engine uses these weights for per-strategy capital allocation

**References:** FR-73, FR-74, NFR-SM3

---

### Story 3.4: Multi-Strategy — Isolation & Capital Allocation

As a quant trader,
I want multiple strategies to run simultaneously with full isolation,
So that a failure in one strategy doesn't affect the others.

**Acceptance Criteria:**

**Given** multiple strategies are active
**When** one strategy encounters a panic or error
**Then** the failed strategy is logged and deactivated
**And** other strategies continue operating normally without interruption
**And** panic recovery catches the error without crashing the service

**Given** a strategy attempts to place a trade
**When** the Portfolio Manager checks per-strategy capital allocation
**Then** orders are rejected if the strategy would exceed its allocation
**And** capital allocation is tracked accurately per strategy

**Given** multiple strategies have closed positions
**When** portfolio-level metrics are calculated
**Then** metrics are aggregated correctly from all strategies
**And** there is no double-counting of positions or PnL
**And** strategy-level metrics are consistent with portfolio-level metrics

**References:** FR-104, FR-105, FR-106, FR-107, AD-13, NFR-MS1, NFR-MS2, NFR-MS3

---

### Story 3.5: Portfolio — Strategy Allocation, Tier Adjustment & Utilization

As a quant trader,
I want the portfolio manager to auto-adjust position limits based on capital tier and track capital utilization,
So that my risk exposure scales appropriately as my capital grows.

**Acceptance Criteria:**

**Given** the portfolio manager tracks total capital (deposits + realized PnL + unrealized PnL)
**When** capital crosses a tier threshold
**Then** position limits are auto-adjusted based on the capital tier (e.g., $10–$50: 20% max position, $1,000+: 5% max)
**And** tier promotion requires capital above threshold for 7 consecutive days
**And** demotion is immediate on capital drop
**And** tier transitions are logged

**Given** capital is deployed across positions
**When** capital utilization rate is queried
**Then** utilization is calculated as % of capital deployed, accurate within 1%
**And** the rate is available via API and dashboard

**Given** the user wants to rebalance strategy weights
**When** a manual rebalance is initiated
**Then** the rebalance executes within 10 seconds
**And** the rebalance action is logged

**References:** FR-32, FR-35, FR-36, FR-37, AD-16, NFR-PM1, NFR-PM2, NFR-PM3

---

### Story 3.6: Advanced Arb — Correlation Detection & Opportunity Logging

As a quant trader,
I want the arb engine to identify correlated markets and flag cascade risk, and log all opportunities for backtesting,
So that I avoid concentrated exposure and have complete data for strategy analysis.

**Acceptance Criteria:**

**Given** the arb engine is tracking market relationships
**When** correlated markets are identified (shared underlying event, price correlation > threshold)
**Then** the correlation matrix is maintained and updated hourly
**And** potential cascade risk is flagged in the `OpportunityDetected` event

**Given** the arb engine detects any opportunity (including below threshold)
**When** the opportunity is evaluated
**Then** it is logged to TimescaleDB with: timestamp, market IDs, scores, filter reason (if filtered)
**And** the log is queryable by date range for backtesting analysis
**And** the opportunity log includes all opportunities, not just executed ones

**References:** FR-15, FR-16, AD-7

---

### Epic 4: Intelligence & Analytics

---

### Story 4.1: Analytics — PnL & Performance Metrics

As a quant trader,
I want comprehensive PnL and performance metrics calculated from my trade history,
So that I can evaluate my trading performance across time periods and strategies.

**Acceptance Criteria:**

**Given** trade history exists in PostgreSQL and TimescaleDB
**When** the user queries PnL analytics
**Then** PnL is calculated and displayed by: day, week, month, strategy, market
**And** all aggregations are accurate within $0.01
**And** queries support arbitrary date ranges

**Given** performance metrics are requested
**When** the analytics service calculates them
**Then** the following metrics are returned: win rate, average win, average loss, profit factor, Sharpe ratio
**And** calculations match manual verification within 1%
**And** metrics are available for any date range combination

**Given** risk metrics are requested
**When** the analytics service calculates them
**Then** max drawdown, current drawdown, and VaR (95%, parametric method) are returned
**And** drawdown is accurate within 1%
**And** all financial calculations use Decimal precision (never float64)

**References:** FR-56, FR-57, FR-58, AD-7, INF-11, NFR-AN1

---

### Story 4.2: Analytics — Charts & CSV Export

As a quant trader,
I want interactive performance charts and CSV export capability,
So that I can visualize trends and analyze data in external tools.

**Acceptance Criteria:**

**Given** the user opens the analytics page
**When** charts render
**Then** a PnL over time line chart, PnL distribution histogram, and PnL by strategy pie chart are displayed
**And** charts are interactive (hover for details, zoom)
**And** charts render within 2 seconds for 1 year of data

**Given** the user requests a CSV export
**When** the export is generated
**Then** all raw trade data is included with all fields
**And** the CSV is well-formed and downloadable
**And** export completes within 10 seconds for 10,000 trades

**References:** FR-59, FR-60, NFR-AN2

---

### Story 4.3: Analytics — Anomaly Detection

As a quant trader,
I want the analytics service to detect performance anomalies automatically,
So that I'm alerted when something unusual happens with my trading performance.

**Acceptance Criteria:**

**Given** the analytics service is continuously monitoring performance metrics
**When** an anomaly is detected (sudden drop in win rate, unusual drawdown pattern)
**Then** the anomaly is detected within 1 day of occurrence
**And** an alert is sent via the notification center
**And** the anomaly is logged with context (metric, threshold, actual value, timestamp)

**References:** FR-61

---

### Story 4.4: Trade History — Filtering & Export

As a quant trader,
I want to filter and export my trade history,
So that I can analyze specific subsets of trades and share data externally.

**Acceptance Criteria:**

**Given** trade history records exist
**When** the user applies filters (date range, market, strategy, side, PnL sign)
**Then** results are returned within 1 second for 10,000 trades
**And** filters are combinable (e.g., date range + strategy + winning trades only)
**And** results are accurate and match the filter criteria

**Given** the user requests a CSV or JSON export
**When** the export is generated
**Then** all fields are included in the export
**And** the file is well-formed
**And** export completes within 10 seconds for 10,000 trades

**References:** FR-63, FR-64, NFR-TH1

---

### Story 4.5: Orderbook Viewer — Real-time Display & Depth Chart

As a quant trader,
I want to view the real-time orderbook with depth chart and recent trades for any market,
So that I can analyze market microstructure and make informed decisions.

**Acceptance Criteria:**

**Given** the user selects a market in the orderbook viewer
**When** the orderbook renders
**Then** bids, asks, and spread are displayed in real-time
**And** updates arrive within 100ms of Polymarket data
**And** the orderbook matches the Polymarket API state

**Given** the orderbook is displayed
**When** the depth chart renders
**Then** cumulative bid/ask at each price level is visualized
**And** the chart updates in real-time, accurate within 1%

**Given** recent trades are requested
**When** the trade list renders
**Then** the last 100 trades are displayed with price, size, timestamp
**And** the list updates in real-time

**And** up to 5 market tabs can be open simultaneously
**And** each tab is independent (no cross-contamination)
**And** memory per tab stays under 100MB

**References:** FR-66, FR-67, FR-68, FR-69, NFR-OV1, NFR-OV2

---

### Epic 5: Backtesting & Paper Trading

---

### Story 5.1: Backtesting Engine — Replay & Simulation

As a quant trader,
I want to replay historical market data through my strategy engine with realistic execution simulation,
So that I can evaluate how my strategy would have performed in the past.

**Acceptance Criteria:**

**Given** historical market data exists in TimescaleDB
**When** the user runs a backtest with a date range and strategy configuration
**Then** the strategy engine processes the historical data in sequence
**And** results are deterministic — same input always produces the same output
**And** execution is simulated with configurable slippage (default: 1%), partial fills, and latency (default: 100ms)
**And** the simulation matches live behavior within 10% on key metrics (win rate, average PnL)
**And** lookahead bias (using future data) is detected and prevented — warnings are logged if detected

**References:** FR-85, FR-86, FR-90, NFR-BT1, NFR-BT2, NFR-BT3

---

### Story 5.2: Backtesting — Metrics, Reports & Parameter Sweeps

As a quant trader,
I want comprehensive backtest metrics, detailed reports, and the ability to test multiple parameter configurations in batch,
So that I can find the optimal strategy parameters before going live.

**Acceptance Criteria:**

**Given** a backtest completes
**When** the results are calculated
**Then** all performance metrics are computed: PnL (total, daily), win rate, Sharpe ratio, drawdown, profit factor
**And** metrics match the Analytics service calculations

**Given** a backtest report is requested
**When** the report is generated
**Then** it includes all metrics, charts (PnL curve, drawdown chart), and trade-by-trade breakdown
**And** the report is exportable

**Given** the user wants to test multiple parameter configurations
**When** a parameter sweep is initiated (e.g., test 100 different threshold values)
**Then** all configurations are tested in batch
**And** the sweep completes within 1 hour for 100 configurations on 1 year of data
**And** results are ranked by the selected metric (e.g., Sharpe ratio)

**References:** FR-87, FR-88, FR-89

---

### Story 5.3: Paper Trading — Simulation & Separate PnL

As a quant trader,
I want to run my strategies in paper trading mode using real market data with simulated execution,
So that I can test strategies without risking real capital.

**Acceptance Criteria:**

**Given** the system is in PAPER execution mode (stored in Redis)
**When** the arb engine detects an opportunity and the execution engine processes it
**Then** no real orders are placed on Polymarket
**And** fills are simulated based on real orderbook depth
**And** fill simulation is realistic — within 10% of actual fill probability
**And** simulated positions are tracked in a separate `paper_positions` table

**Given** paper trades are executed
**When** PnL is calculated
**Then** simulated PnL is tracked independently from live PnL
**And** paper PnL is clearly labeled and separated in the dashboard
**And** all simulated trades are logged with the same detail as live trades (same format, queryable via Trade History)

**And** paper trading never affects live positions or capital — complete isolation

**References:** FR-91, FR-92, FR-93, FR-95, AD-12, NFR-PT1, NFR-PT2

---

### Story 5.4: Paper Trading — Seamless Live Switch

As a quant trader,
I want to seamlessly switch a strategy from paper trading to live trading with the same configuration,
So that I can go live with a validated strategy without re-entering parameters.

**Acceptance Criteria:**

**Given** a strategy has been running in paper trading mode
**When** the user switches the execution mode from PAPER to LIVE
**Then** the switch takes effect within 1 second
**And** no configuration re-entry is required — the same strategy config is used
**And** the mode change is logged
**And** a restart is required for the mode switch to take effect (per AD-12, to prevent accidental live trades)

**References:** FR-94, AD-12, NFR-PT3

---

### Story 5.5: Replay Mode — Speed Control & Decision Display

As a quant trader,
I want to replay historical market events at configurable speed and see the bot's decision-making process,
So that I can debug specific trades and understand bot behavior.

**Acceptance Criteria:**

**Given** historical market data exists for a selected date range
**When** the user starts a replay
**Then** market events are replayed at the selected speed (1x, 2x, 5x, 10x)
**And** the replay is accurate — matches historical data exactly
**And** playback is smooth at 10x speed

**Given** the replay is running
**When** the bot processes each market event
**Then** bot decisions are displayed in real-time: what was detected, what was decided, and why
**And** the decision log matches the historical execution records

**Given** the user interacts with replay controls
**When** pause, step-forward, or rewind is triggered
**Then** controls are responsive within 100ms
**And** state is consistent after any control action

**Given** risk events occurred during the historical period
**When** those events are replayed
**Then** risk events (circuit breaker triggers, limit breaches) are visually highlighted
**And** details are available on click

**References:** FR-96, FR-97, FR-98, FR-99, NFR-RP1, NFR-RP2, NFR-RP3

---

### Epic 6: AI Strategy Optimization

---

### Story 6.1: AI Optimizer — Pattern Analysis & Suggestions

As a quant trader,
I want the AI optimizer to analyze my trade history and identify patterns in winning vs losing trades,
So that I can improve my strategy parameters based on data-driven insights.

**Acceptance Criteria:**

**Given** at least 100 trades exist in the trade history
**When** the AI optimizer analyzes the data
**Then** it identifies statistically significant patterns (p < 0.05) in winning vs losing trades
**And** patterns include: market characteristics, time-of-day effects, score thresholds, position sizing

**Given** patterns are identified
**When** the optimizer generates suggestions
**Then** each suggestion includes: parameter change, expected impact (quantified), and methodology
**And** suggestions are displayed for manual review — no auto-application
**And** approval (or rejection) is logged

**References:** FR-75, FR-76, FR-77, NFR-AI1, NFR-AI2

---

### Story 6.2: AI Optimizer — A/B Testing & Overfitting Detection

As a quant trader,
I want the AI optimizer to A/B test suggestions in paper trading and detect overfitting,
So that I only deploy changes that generalize beyond historical data.

**Acceptance Criteria:**

**Given** an AI suggestion has been approved for testing
**When** the A/B test runs in paper trading
**Then** the suggestion is tested against the current strategy in parallel
**And** results are compared with statistical significance (p < 0.05 required)
**And** the comparison is available for review before recommending for live deployment

**Given** the optimizer generates a suggestion
**When** overfitting analysis is performed
**Then** the suggestion is tested on out-of-sample data
**And** if out-of-sample performance degrades by more than 20%, an overfitting warning is displayed
**And** the warning includes the performance gap and recommendation

**References:** FR-78, FR-79, NFR-AI3

---

### Story 6.3: AI Assistant — Performance Q&A & Decision Explanation

As a quant trader,
I want an AI assistant that answers questions about my trading performance and explains bot decisions,
So that I can quickly understand what's happening without digging through logs.

**Acceptance Criteria:**

**Given** the AI assistant is configured with an LLM API key
**When** the user asks a performance question (e.g., "What's my PnL this week?")
**Then** the answer is generated within 5 seconds
**And** all numerical values are verified against the database (no hallucinated numbers)
**And** the answer references specific data points

**Given** the user asks about a specific bot decision (e.g., "Why did the bot enter the XYZ market?")
**When** the assistant generates an explanation
**Then** the explanation traces back to specific trade log entries and decision context
**And** the explanation is clear and understandable without technical jargon

**References:** FR-100, FR-101, NFR-AA1, NFR-AA3

---

### Story 6.4: AI Assistant — Read-only Safety & Risk Parameter Suggestions

As a quant trader,
I want the AI assistant to suggest conservative risk parameter adjustments while being strictly read-only,
So that I get helpful suggestions without any risk of the assistant modifying my system.

**Acceptance Criteria:**

**Given** the AI assistant is analyzing current risk state
**When** it generates risk parameter suggestions
**Then** suggestions are conservative — never recommending increasing risk beyond current limits
**And** suggestions include rationale based on current state and historical performance

**Given** the assistant is operational
**When** any action is attempted
**Then** the assistant CANNOT execute trades or modify configurations directly
**And** all suggested actions require explicit human approval
**And** the assistant has read-only access to analytics, positions, and trade history

**References:** FR-102, FR-103, NFR-AA2

---

### Epic 7: Scaling & Enterprise

---

### Story 7.1: Multi-Account — Wallet Configuration & Isolation

As a quant trader,
I want to configure multiple Polymarket wallets with fully isolated state,
So that I can segregate risk between test capital and production capital.

**Acceptance Criteria:**

**Given** the multi-account feature is enabled
**When** the user configures a new Polymarket wallet
**Then** the wallet configuration is persisted with encrypted private key (encrypted at rest, never shared)
**And** each wallet has independent state: positions, PnL, risk limits
**And** no cross-contamination between accounts — positions from account A are invisible to account B

**Given** multiple wallets are configured
**When** the system operates
**Then** risk limits are enforced per account independently — no sharing of risk budget
**And** each account's Pit Boss state is isolated in Redis

**References:** FR-108, FR-109, FR-111, AD-14, NFR-MA1, NFR-MA2

---

### Story 7.2: Multi-Account — Cross-Account View & Per-Account Risk

As a quant trader,
I want a cross-account portfolio view that aggregates all accounts while preserving per-account visibility,
So that I can see my total exposure and drill into individual accounts.

**Acceptance Criteria:**

**Given** multiple accounts are configured and active
**When** the user opens the cross-account portfolio view
**Then** aggregate metrics are displayed: total capital, total PnL, total positions across all accounts
**And** the aggregation is accurate — no double-counting
**And** individual account views are still accessible with per-account metrics

**Given** risk limits are configured per account
**When** a trade is evaluated
**Then** risk limits are applied based on the specific account's budget
**And** cross-account risk exposure (total across all accounts) is also visible in the dashboard

**References:** FR-110, FR-111, NFR-MA3

---

### Story 7.3: Admin Panel — System Configuration & Health Dashboard

As a quant trader,
I want an admin panel to manage system configuration and monitor system health,
So that I can adjust settings and troubleshoot issues without editing config files.

**Acceptance Criteria:**

**Given** the admin panel is accessible (requires authentication)
**When** the user opens the system configuration page
**Then** the following can be configured via the UI: API keys, risk defaults, notification settings
**And** all changes are validated before save
**And** changes are persisted and logged with timestamp and previous value

**Given** the user opens the system health page
**When** health metrics render
**Then** the following are displayed: CPU, memory, disk, network, connection status
**And** metrics are accurate and update every 5 seconds
**And** alerts are displayed when thresholds are breached

**References:** FR-112, FR-113, FR-116, NFR-AP1

---

### Story 7.4: Admin Panel — Log Viewer & Database Management

As a quant trader,
I want a log viewer with filtering and database management tools,
So that I can debug issues and maintain my data without SSH access.

**Acceptance Criteria:**

**Given** the admin panel log viewer is open
**When** the user applies filters (level, component, date range, search text)
**Then** logs are returned within 1 second for 1,000,000 entries
**And** results are accurate and match the filter criteria
**And** log entries include: timestamp, level, service, request_id, message, context

**Given** the user opens the database management page
**When** a backup is initiated
**Then** the backup completes within 10 minutes for a 1GB database
**And** backups are automated daily with 30-day retention
**And** restore functionality is available and tested
**And** database cleanup (remove old data beyond retention) is supported

**References:** FR-114, FR-115, NFR-AP2, NFR-AP3
