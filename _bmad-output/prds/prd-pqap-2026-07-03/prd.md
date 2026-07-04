---
title: Polymarket Quant Arbitrage Platform (PQAP)
status: draft
created: 2026-07-03
updated: 2026-07-03
---

# PRD: Polymarket Quant Arbitrage Platform (PQAP)

## 0. Document Purpose

This PRD defines the complete behavioral and functional requirements for PQAP — a production-grade automated arbitrage trading platform for Polymarket prediction markets. It serves as the authoritative contract between product intent and engineering implementation, covering all features from market scanning to AI-driven strategy optimization.

**Scope:** Full production system, not MVP. Single-user (Juragan), personal use only.

**Reading guide:** Each feature section contains behavioral descriptions, numbered functional requirements (FRs) with testable consequences, and feature-specific NFRs. Assumptions are tagged `[ASSUMPTION: ...]`. User journey references use `UJ-` prefixes defined in Section 2.3.

---

## 1. Vision

**Year 1:** A reliable, profitable trading bot that generates consistent returns on Polymarket. The bot becomes smarter with every trade, learning which markets, times, and conditions produce the best opportunities.

**Year 2–3:** A sophisticated quant platform that can compete with professional market makers. Multiple strategies running simultaneously, AI-optimized parameters, and capital large enough to capture meaningful edges.

**The dream:** A self-sustaining trading operation where the bot hunts 24/7, compounds returns, and Juragan focuses on strategy and risk management — not execution.

**Core philosophy:** "Disciplined Predator" — fast but not reckless, aggressive but not greedy, adaptive but not inconsistent. Every decision is data-driven, every trade is scored, every risk is budgeted.

---

## 2. Target User

### 2.1 Jobs To Be Done

| ID | Job | Current Solution | Pain |
|----|-----|------------------|------|
| JTBD-1 | Generate passive income from prediction market inefficiencies | Manual trading on Polymarket | Time-consuming, emotional, error-prone |
| JTBD-2 | Monitor hundreds of markets simultaneously | Impossible for humans | Missed opportunities |
| JTBD-3 | Execute trades at optimal speed | Manual order placement | Latency kills edges |
| JTBD-4 | Manage risk across positions | Spreadsheet tracking | No automated enforcement |
| JTBD-5 | Learn from trade outcomes | Manual review | No systematic feedback loop |
| JTBD-6 | Scale capital efficiently | Manual position sizing | Inconsistent, emotional |

### 2.2 Non-Users (v1)

- **Public/SaaS customers** — PQAP is personal use only
- **Mobile-first users** — dashboard is web-based
- **Fiat traders** — USDC on Polygon only
- **Social traders** — no signal sharing or copy trading

### 2.3 Key User Journeys

| ID | Journey | Frequency | Critical Path |
|----|---------|-----------|---------------|
| UJ-1 | Start bot → scan markets → detect opportunity → execute trade → track PnL | Continuous | Scanner → Arb Engine → Execution → Position Manager |
| UJ-2 | Morning review: check overnight performance, adjust risk params | Daily | Dashboard → Analytics → Risk Management |
| UJ-3 | Capital increase: deposit USDC → adjust position sizing → resume trading | Monthly | Portfolio Manager → Risk Management |
| UJ-4 | Strategy backtest: define parameters → run on historical data → evaluate | Weekly | Backtesting → Analytics → Strategy Manager |
| UJ-5 | Emergency: circuit breaker triggers → review cause → adjust → resume | As needed | Risk Management → Notification → Dashboard |
| UJ-6 | Market discovery: browse markets → analyze fundamentals → add to watchlist | Daily | Market Scanner → Orderbook Viewer |
| UJ-7 | Strategy optimization: review AI suggestions → approve changes → deploy | Weekly | AI Strategy Optimizer → Strategy Manager |
| UJ-8 | Paper trading: test new strategy in simulation → evaluate → go live | Monthly | Paper Trading → Analytics → Strategy Manager |

---

## 3. Glossary

| Term | Definition |
|------|------------|
| **YES/NO market** | Binary prediction market where outcomes are priced $0.00–$1.00 |
| **CLOB** | Central Limit Order Book — Polymarket's order matching system |
| **Arbitrage** | Risk-free profit from market mispricing (e.g., YES + NO < $1.00) |
| **Cross-market arb** | Arbitrage between related markets (e.g., "Will X happen?" vs "Will X by date Y?") |
| **Liquidity capture** | Profiting from providing liquidity at favorable prices |
| **Kill zone** | Optimal time window for trading (high liquidity, low competition) |
| **Metabolic rate** | System resource consumption rate — must stay below danger threshold |
| **Cascade risk** | Chain reaction of losses across correlated positions |
| **State reconciliation** | Continuous verification that internal state matches exchange state |
| **Batasi Win** | Win-streak breaker — pauses after consecutive wins to prevent overconfidence |
| **Smart Speed** | Fast detection, fast filtering, fast execution, fast rejection |
| **Market Mastery Matrix** | Framework for scoring market knowledge and edge quality |
| **Opportunity Scoring Engine** | Ranks opportunities by spread × liquidity × fill probability |
| **USDC** | USD Coin — stablecoin used as collateral on Polymarket |
| **Polygon** | Ethereum L2 chain where Polymarket operates (Chain ID 137) |
| **UMA oracle** | Oracle system for market resolution with dispute period |
| **GTC/FOK/GTD/FAK** | Order types: Good-Til-Canceled, Fill-Or-Kill, Good-Til-Date, Fill-And-Kill |

---

## 4. Features

### 4.1 Market Scanner

**Behavioral Description:**
The Market Scanner is the system's sensory organ — it continuously monitors all active Polymarket markets via WebSocket connections, maintaining a real-time view of prices, spreads, and liquidity. It feeds downstream components (Arbitrage Engine, Opportunity Scoring) with clean, normalized market data.

The scanner operates in two modes: **streaming** (real-time WebSocket for active markets) and **polling** (periodic REST for market discovery). It maintains an internal market catalog that tracks market metadata, current prices, spread widths, and liquidity depth.

**Edge cases:**
- WebSocket disconnection → automatic reconnect with state reconciliation
- Stale data detection → mark markets as "stale" if no updates received within threshold
- Market discovery lag → new markets may appear; scanner must detect and subscribe
- Rate limiting → implement backoff and request batching

**User Journeys:** UJ-1 (continuous scanning), UJ-6 (market discovery)

| # | Functional Requirement | Consequences (Testable) |
|---|------------------------|-------------------------|
| FR-1 | Scanner SHALL connect to Polymarket WebSocket API and subscribe to all active binary markets | Connection established within 5s; all active markets subscribed; reconnect within 2s on disconnect |
| FR-2 | Scanner SHALL maintain an internal market catalog with: market ID, slug, current YES/NO prices, spread, volume, liquidity depth | Catalog updated within 100ms of price change; catalog contains all fields for every active market |
| FR-3 | Scanner SHALL detect new markets within 60 seconds of their appearance on Polymarket | New market appears in catalog within 60s; metadata populated correctly |
| FR-4 | Scanner SHALL mark markets as "stale" if no price updates received within configurable threshold (default: 30s) | Stale flag set; downstream consumers notified; stale markets excluded from opportunity scoring |
| FR-5 | Scanner SHALL implement automatic WebSocket reconnection with exponential backoff (initial: 1s, max: 60s) | Reconnect attempted within 1s of disconnect; backoff applied correctly; no data loss during reconnect |
| FR-6 | Scanner SHALL reconcile state after reconnection by fetching current orderbook snapshot | Prices match within 1 tick of REST API snapshot after reconnect |
| FR-7 | Scanner SHALL batch REST API calls when fetching multiple market data (up to 100 markets per request) | API call count reduced by ≥80% vs individual requests |
| FR-8 | Scanner SHALL export metrics: markets tracked, update latency, connection status, stale count | All metrics available via Prometheus endpoint; values accurate within 1s |

**Feature-specific NFRs:**
- **Latency:** Price updates processed within 50ms of receipt
- **Throughput:** Handle 500+ concurrent market subscriptions
- **Reliability:** 99.9% uptime for WebSocket connection
- **Memory:** Market catalog fits within 512MB for 1000 markets

---

### 4.2 Arbitrage Engine

**Behavioral Description:**
The Arbitrage Engine is the brain's pattern recognition — it analyzes market data from the Scanner to identify mispricings. It supports multiple arbitrage strategies:

1. **Simple YES+NO arb:** Detect when YES_price + NO_price < $1.00 (guaranteed profit at resolution)
2. **Cross-market arb:** Detect when related markets are mispriced against each other
3. **Liquidity capture:** Detect wide spreads where placing limit orders captures the spread

The engine scores each opportunity using the **Opportunity Scoring Engine**: `score = spread × liquidity × fill_probability`. Only opportunities above configurable thresholds are passed to the Execution Engine.

**Edge cases:**
- Market resolution imminent → opportunity may not fill in time
- Illiquid market → high spread but low fill probability
- Stale prices → false positive opportunities
- Correlated markets → same underlying event, different market IDs

**User Journeys:** UJ-1 (detect and score opportunities)

| # | Functional Requirement | Consequences (Testable) |
|---|------------------------|-------------------------|
| FR-9 | Engine SHALL detect simple YES+NO arbitrage when YES_price + NO_price < $1.00 - min_profit_threshold | Opportunities detected within 100ms of price update; no false negatives for profitable arb |
| FR-10 | Engine SHALL detect cross-market arbitrage between related markets | At least 3 cross-market relationship types detected; false positive rate < 10% |
| FR-11 | Engine SHALL calculate opportunity score: `spread × liquidity × fill_probability` | Score calculated for every detected opportunity; score matches manual calculation within 1% |
| FR-12 | Engine SHALL filter opportunities below configurable score threshold (default: 0.01) | Low-score opportunities logged but not passed to execution; threshold configurable per strategy |
| FR-13 | Engine SHALL estimate fill probability based on orderbook depth and historical fill rates | Fill probability estimate within 15% of actual fill rate over 100 trades |
| FR-14 | Engine SHALL detect when market resolution is imminent (within 1 hour) and reduce confidence score | Confidence reduced by 50% for near-resolution markets; configurable threshold |
| FR-15 | Engine SHALL identify correlated markets and flag potential cascade risk | Correlated markets flagged; correlation matrix maintained and updated hourly |
| FR-16 | Engine SHALL log all detected opportunities (including filtered ones) for backtesting analysis | Complete opportunity log with timestamps, scores, filter reasons; queryable by date range |

**Feature-specific NFRs:**
- **Latency:** Opportunity detection within 100ms of price update
- **Accuracy:** Zero false negatives for profitable YES+NO arbitrage (spread > threshold)
- **Scoring:** Score calculation deterministic and reproducible

---

### 4.3 Execution Engine

**Behavioral Description:**
The Execution Engine is the muscle — it takes scored opportunities from the Arbitrage Engine and executes trades via Polymarket's CLOB API. It implements **Smart Speed**: fast detection, fast filtering, fast execution, fast rejection.

The engine uses limit orders (GTC) by default for price protection, with FOK for time-critical opportunities. It implements slippage protection: if the price moves beyond the configured slippage tolerance before the order is placed, the trade is rejected.

**Execution flow:**
1. Receive opportunity from Arbitrage Engine
2. Validate against risk budget (via Risk Management)
3. Calculate optimal order size and price
4. Place order via CLOB API
5. Monitor fill status
6. Update Position Manager on fill

**Edge cases:**
- Insufficient balance → reject and log
- Order partially filled → track remaining and decide whether to cancel or wait
- API error → retry with backoff, circuit breaker on repeated failures
- Price moved beyond slippage tolerance → reject
- Network timeout → idempotent retry

**User Journeys:** UJ-1 (execute trade), UJ-5 (circuit breaker)

| # | Functional Requirement | Consequences (Testable) |
|---|------------------------|-------------------------|
| FR-17 | Engine SHALL place limit orders (GTC) by default with configurable time-in-force override | Orders placed as GTC unless strategy specifies FOK/GTD/FAK; order type logged |
| FR-18 | Engine SHALL validate order against risk budget before placement | Order rejected if daily budget exceeded; rejection logged with reason |
| FR-19 | Engine SHALL implement slippage protection: reject if price moves beyond tolerance (default: 1%) | Order cancelled if price moves >1% from signal; tolerance configurable per strategy |
| FR-20 | Engine SHALL handle partial fills: track filled quantity and decide cancel/wait based on strategy | Partial fill logged; decision (cancel/wait) logged with reason; position updated correctly |
| FR-21 | Engine SHALL implement circuit breaker: halt trading after N consecutive API errors (default: 5) | Trading halted after 5 errors; alert sent; manual resume required |
| FR-22 | Engine SHALL implement idempotent order placement (client order ID prevents duplicates) | Duplicate orders detected and rejected; no double-spending on retry |
| FR-23 | Engine SHALL execute both legs of YES+NO arbitrage atomically (both succeed or both cancel) | Both legs placed within 500ms; if one fails, other cancelled within 1s |
| FR-24 | Engine SHALL log every order attempt with: timestamp, market, side, price, size, result, latency | Complete audit trail; queryable; latency accurate within 10ms |

**Feature-specific NFRs:**
- **Latency:** Order placement within 200ms of decision
- **Reliability:** 99.9% order placement success rate (excluding API outages)
- **Atomicity:** YES+NO legs executed within 500ms window
- **Idempotency:** Zero duplicate orders under any failure scenario

---

### 4.4 Position Manager

**Behavioral Description:**
The Position Manager tracks all open positions in real-time. It maintains the canonical state of what the bot owns, reconciles with Polymarket's API, and provides position data to Risk Management and Portfolio Manager.

**Position lifecycle:**
1. Opened (order filled)
2. Monitored (price tracking, PnL calculation)
3. Closed (market resolution or manual exit)
4. Settled (profit/loss realized)

**Edge cases:**
- Market resolution while position open → automatic settlement
- API state disagrees with internal state → reconciliation alert
- Position exceeds risk limits → alert and potential forced exit

**User Journeys:** UJ-1 (track position), UJ-2 (review positions)

| # | Functional Requirement | Consequences (Testable) |
|---|------------------------|-------------------------|
| FR-25 | Manager SHALL track all open positions with: market, side, entry price, current price, quantity, unrealized PnL | Position data accurate within 1% of actual; updated within 1s of price change |
| FR-26 | Manager SHALL reconcile position state with Polymarket API every 60 seconds | Discrepancies detected and alerted within 60s; reconciliation logged |
| FR-27 | Manager SHALL detect market resolution and automatically settle positions | Settlement calculated correctly; PnL updated; position moved to history |
| FR-28 | Manager SHALL calculate unrealized PnL using current market prices | PnL accurate within 1% of manual calculation; updated in real-time |
| FR-29 | Manager SHALL alert when position exceeds configured limits | Alert sent within 5s of limit breach; alert includes position details |
| FR-30 | Manager SHALL support manual position exit (close at market) | Exit order placed within 1s of command; exit price logged |

**Feature-specific NFRs:**
- **Accuracy:** Position state matches Polymarket API within 1%
- **Latency:** PnL update within 1s of price change
- **Reconciliation:** State mismatch detected within 60s

---

### 4.5 Portfolio Manager

**Behavioral Description:**
The Portfolio Manager oversees the total capital allocation across strategies and positions. It enforces capital efficiency rules: every dollar must work hard, but no dollar works too hard.

**Key responsibilities:**
- Track total capital (deposits + realized PnL + unrealized PnL)
- Allocate capital across strategies based on configured weights
- Enforce per-strategy and per-market position limits
- Calculate capital utilization rate
- Support capital scaling rules (auto-adjust as capital grows)

**Capital scaling tiers:**
| Capital Level | Strategy Focus | Max Position Size |
|---------------|----------------|-------------------|
| $10–$50 | Simple arb only | 20% of capital |
| $50–$200 | Add cross-market arb | 15% of capital |
| $200–$1,000 | Add liquidity capture | 10% of capital |
| $1,000–$10,000 | Full strategy suite | 5% of capital |
| $10,000+ | Market making component | 3% of capital |

**User Journeys:** UJ-3 (capital increase), UJ-2 (portfolio review)

| # | Functional Requirement | Consequences (Testable) |
|---|------------------------|-------------------------|
| FR-31 | Manager SHALL track total capital: deposits + realized PnL + unrealized PnL | Capital accurate within $0.01; updated within 1s of any change |
| FR-32 | Manager SHALL allocate capital across strategies based on configurable weights | Allocation sums to 100%; weights adjustable at runtime |
| FR-33 | Manager SHALL enforce per-strategy position limits (configurable, default: 20% of capital) | Orders rejected if strategy limit exceeded; rejection logged |
| FR-34 | Manager SHALL enforce per-market position limits (configurable, default: 10% of capital) | Orders rejected if market limit exceeded; rejection logged |
| FR-35 | Manager SHALL auto-adjust position limits based on capital tier | Limits recalculated within 1s of tier change; tier transitions logged |
| FR-36 | Manager SHALL calculate capital utilization rate (% of capital deployed) | Utilization accurate within 1%; available via API and dashboard |
| FR-37 | Manager SHALL support manual capital rebalancing (adjust strategy weights) | Rebalance executed within 10s; rebalance logged |

**Feature-specific NFRs:**
- **Accuracy:** Capital calculations accurate to $0.01
- **Consistency:** Allocation always sums to 100%
- **Latency:** Limit enforcement within 100ms of order attempt

---

### 4.6 Risk Management

**Behavioral Description:**
Risk Management is the system's survival instinct — inspired by casino operations, it ensures the bot never bets more than it can afford to lose. It implements multiple layers of protection:

1. **Daily budget:** Maximum loss per day (default: 2% of capital)
2. **Position limits:** Maximum exposure per market and per strategy
3. **Correlation limits:** Maximum positions in correlated markets
4. **Win streak breaker (Batasi Win):** Pause after N consecutive wins (default: 5)
5. **Drawdown circuit breaker:** Halt trading if drawdown exceeds threshold
6. **Metabolic rate monitor:** Ensure system resource consumption stays safe
7. **Emergency stop:** Immediate halt on critical failures

The **Pit Boss** concept: a centralized risk monitor (backed by Redis) that all components consult before taking action. No trade happens without Pit Boss approval.

**Risk hierarchy:**
1. Preserve capital — never risk more than daily budget allows
2. Find edges — only trade when math is clear
3. Execute fast — once decided, act without hesitation
4. Learn constantly — every trade teaches something

**User Journeys:** UJ-5 (circuit breaker), UJ-2 (review risk), UJ-1 (risk check before trade)

| # | Functional Requirement | Consequences (Testable) |
|---|------------------------|-------------------------|
| FR-38 | System SHALL enforce daily loss limit (default: 2% of capital, configurable) | Trading halted when daily loss reaches limit; halt logged; alert sent |
| FR-39 | System SHALL enforce max position per market (default: 10% of capital, configurable) | Orders rejected if market position exceeds limit; rejection logged |
| FR-40 | System SHALL enforce max correlated positions (default: 3, configurable) | Orders rejected if correlated position count exceeds limit; rejection logged |
| FR-41 | System SHALL implement Batasi Win (win streak breaker): pause after N consecutive wins (default: 5) | Trading paused after 5 wins; resume requires manual approval or configurable cooldown |
| FR-42 | System SHALL implement drawdown circuit breaker: halt if drawdown exceeds threshold (default: 10%) | Trading halted on drawdown breach; alert sent; manual resume required |
| FR-43 | System SHALL implement metabolic rate monitor: track CPU, memory, goroutine counts | Metrics exported; alerts when consumption exceeds thresholds (CPU >80%, memory >1GB) |
| FR-44 | System SHALL implement emergency stop: immediate halt on critical failures (API death spiral, data corruption) | All trading halted within 1s; all open orders cancelled; alert sent |
| FR-45 | Pit Boss SHALL be consulted before every trade; trades rejected if Pit Boss returns deny | Zero trades bypass Pit Boss; denial logged with reason |
| FR-46 | System SHALL maintain risk state in Redis for cross-component access | Risk state consistent across all components; state updates within 10ms |
| FR-47 | System SHALL log all risk decisions (approve/deny) with full context | Complete audit trail; queryable by date, component, reason |

**Feature-specific NFRs:**
- **Latency:** Risk check within 10ms of trade request
- **Consistency:** Risk state consistent across all components via Redis
- **Reliability:** Pit Boss 99.99% availability (no single point of failure)
- **Auditability:** Complete risk decision log with full context

---

### 4.7 Dashboard

**Behavioral Description:**
The Dashboard is the command center — a real-time web interface (Next.js) that provides Juragan with complete visibility into the bot's operation. It displays:

- **Portfolio overview:** Total capital, PnL, utilization rate
- **Active positions:** Current holdings with real-time PnL
- **Opportunity feed:** Live stream of detected and executed opportunities
- **Risk status:** Daily budget remaining, drawdown, win streak
- **System health:** Connection status, metabolic rate, error counts
- **Quick actions:** Emergency stop, pause trading, adjust risk params

The dashboard updates in real-time via WebSocket (server-sent events or Socket.io). It's designed for quick glances (morning review) and deep dives (strategy analysis).

**User Journeys:** UJ-2 (morning review), UJ-5 (emergency review), UJ-6 (market discovery)

| # | Functional Requirement | Consequences (Testable) |
|---|------------------------|-------------------------|
| FR-48 | Dashboard SHALL display portfolio overview: total capital, daily PnL, total PnL, utilization rate | Data accurate within 1% of backend; updates within 2s of change |
| FR-49 | Dashboard SHALL display all active positions with real-time PnL | Position list matches backend; PnL updates within 2s |
| FR-50 | Dashboard SHALL display live opportunity feed (detected and executed) | Feed updates within 1s of opportunity detection; scrollable history |
| FR-51 | Dashboard SHALL display risk status: daily budget, drawdown, win streak, circuit breaker status | Risk data matches backend; updates within 2s |
| FR-52 | Dashboard SHALL display system health: connection status, CPU, memory, error rate | Health metrics update every 5s; accurate within 10% |
| FR-53 | Dashboard SHALL provide quick actions: emergency stop, pause/resume, risk param adjustment | Actions execute within 1s; confirmation required for critical actions |
| FR-54 | Dashboard SHALL be responsive (desktop-first, tablet-compatible) | Usable on 1024px+ screens; no horizontal scrolling |
| FR-55 | Dashboard SHALL support dark mode | Toggle persists across sessions; all elements styled correctly |

**Feature-specific NFRs:**
- **Latency:** Real-time updates within 2s
- **Performance:** Page load < 3s on 3G connection
- **Accessibility:** WCAG 2.1 AA compliance for critical elements

---

### 4.8 Analytics

**Behavioral Description:**
Analytics provides deep insight into trading performance, strategy effectiveness, and market patterns. It answers questions like: "Which strategy is most profitable?" "What time of day produces the best opportunities?" "Am I improving over time?"

**Key metrics:**
- **PnL metrics:** Total, daily, weekly, monthly, by strategy, by market
- **Performance metrics:** Win rate, average win, average loss, profit factor, Sharpe ratio
- **Risk metrics:** Max drawdown, current drawdown, VaR, correlation exposure
- **Operational metrics:** Trades per day, average hold time, fill rate, slippage

**User Journeys:** UJ-2 (performance review), UJ-4 (strategy evaluation)

| # | Functional Requirement | Consequences (Testable) |
|---|------------------------|-------------------------|
| FR-56 | Analytics SHALL calculate and display PnL by: day, week, month, strategy, market | All aggregations accurate within $0.01; queryable by date range |
| FR-57 | Analytics SHALL calculate win rate, average win/loss, profit factor, Sharpe ratio | Calculations match manual verification within 1%; available for any date range |
| FR-58 | Analytics SHALL calculate max drawdown, current drawdown, VaR (95%) | Drawdown accurate within 1%; VaR calculated using parametric method |
| FR-59 | Analytics SHALL visualize PnL over time (line chart), distribution (histogram), by strategy (pie) | Charts render within 2s; data accurate; interactive (hover, zoom) |
| FR-60 | Analytics SHALL export data to CSV for external analysis | Export includes all raw trade data; CSV well-formed; export within 10s for 10k trades |
| FR-61 | Analytics SHALL detect performance anomalies (sudden drop in win rate, unusual drawdown) | Anomaly detected within 1 day; alert sent; anomaly logged |

**Feature-specific NFRs:**
- **Accuracy:** All financial calculations accurate to $0.01
- **Performance:** Charts render within 2s for 1 year of data
- **Storage:** TimescaleDB for efficient time-series queries

---

### 4.9 Trade History

**Behavioral Description:**
Trade History is the permanent record of every trade the bot has executed. It stores complete details for every order attempt (successful or not) and provides query, filter, and export capabilities.

**User Journeys:** UJ-2 (review trades), UJ-4 (backtesting validation)

| # | Functional Requirement | Consequences (Testable) |
|---|------------------------|-------------------------|
| FR-62 | History SHALL record every trade with: timestamp, market, side, price, quantity, fill status, PnL, strategy, latency | All fields populated; no NULL values for required fields |
| FR-63 | History SHALL support filtering by: date range, market, strategy, side, PnL sign | Filters combinable; results accurate; query < 1s for 10k trades |
| FR-64 | History SHALL support export to CSV and JSON | Export includes all fields; well-formed; < 10s for 10k trades |
| FR-65 | History SHALL be immutable (no edits, no deletes) | Attempts to modify rejected; audit log of any access |

**Feature-specific NFRs:**
- **Storage:** PostgreSQL with proper indexing for query performance
- **Retention:** Minimum 3 years of history
- **Query:** < 1s response for typical queries on 10k trades

---

### 4.10 Orderbook Viewer

**Behavioral Description:**
The Orderbook Viewer provides a real-time view of the Polymarket orderbook for any selected market. It displays bids, asks, spread, depth, and recent trades. Used for manual market analysis and debugging.

**User Journeys:** UJ-6 (market analysis)

| # | Functional Requirement | Consequences (Testable) |
|---|------------------------|-------------------------|
| FR-66 | Viewer SHALL display real-time orderbook for selected market (bids, asks, spread) | Orderbook updates within 100ms; matches Polymarket API |
| FR-67 | Viewer SHALL display orderbook depth chart (cumulative bid/ask at each price level) | Chart updates in real-time; accurate within 1% |
| FR-68 | Viewer SHALL display recent trades (last 100) with price, size, timestamp | Trade list updates in real-time; matches Polymarket data |
| FR-69 | Viewer SHALL support multiple market tabs (up to 5 simultaneous) | Each tab independent; no cross-contamination; memory < 100MB per tab |

**Feature-specific NFRs:**
- **Latency:** Orderbook updates within 100ms
- **Memory:** < 100MB per market tab

---

### 4.11 Strategy Manager

**Behavioral Description:**
The Strategy Manager defines, configures, and activates trading strategies. Each strategy has its own parameters (thresholds, position sizing, risk limits) and can be enabled/disabled independently.

**Strategy types:**
1. **Simple YES+NO arbitrage** — buy YES + NO when combined < $1.00
2. **Cross-market arbitrage** — exploit mispricings between related markets
3. **Liquidity capture** — profit from wide spreads by placing limit orders
4. **Market making** — provide liquidity and capture the spread (Phase 3)

**User Journeys:** UJ-7 (strategy optimization), UJ-8 (paper trading)

| # | Functional Requirement | Consequences (Testable) |
|---|------------------------|-------------------------|
| FR-70 | Manager SHALL support CRUD operations for strategies | Strategy created/updated/deleted; changes persisted; validation on all fields |
| FR-71 | Manager SHALL support strategy activation/deactivation without restart | Strategy starts/stops within 1s; no impact on other strategies |
| FR-72 | Manager SHALL validate strategy parameters (thresholds, limits, weights) | Invalid parameters rejected with clear error message; no partial saves |
| FR-73 | Manager SHALL support strategy versioning (track parameter changes) | Version history maintained; rollback to any version supported |
| FR-74 | Manager SHALL assign capital allocation weights to each active strategy | Weights sum to 100%; enforcement via Portfolio Manager |

**Feature-specific NFRs:**
- **Persistence:** Strategy configs stored in PostgreSQL
- **Validation:** All parameters validated before save
- **Versioning:** Complete version history with rollback

---

### 4.12 AI Strategy Optimizer

**Behavioral Description:**
The AI Strategy Optimizer uses machine learning to analyze trade history and suggest parameter improvements. It identifies patterns in winning/losing trades and recommends adjustments to thresholds, position sizing, and strategy allocation.

**Optimization targets:**
- Score thresholds (what minimum score produces profitable trades?)
- Position sizing (what size maximizes risk-adjusted returns?)
- Strategy allocation (what mix of strategies performs best?)
- Kill zones (what time windows produce the best opportunities?)
- Market selection (which market categories are most profitable?)

[ASSUMPTION: Sufficient trade history (>100 trades) is required for meaningful optimization]

**User Journeys:** UJ-7 (review suggestions, approve changes)

| # | Functional Requirement | Consequences (Testable) |
|---|------------------------|-------------------------|
| FR-75 | Optimizer SHALL analyze trade history and identify patterns in winning vs losing trades | Pattern identification with statistical significance (p < 0.05) |
| FR-76 | Optimizer SHALL suggest parameter adjustments with expected impact (e.g., "increasing threshold by 0.005 would have improved Sharpe by 15%") | Suggestions include quantified expected impact; impact calculation methodology documented |
| FR-77 | Optimizer SHALL require manual approval before applying suggestions | Suggestions displayed for review; no auto-apply; approval logged |
| FR-78 | Optimizer SHALL A/B test suggestions in paper trading before recommending for live | Paper trading comparison available; statistical significance required |
| FR-79 | Optimizer SHALL detect overfitting and warn when suggestions are too specific to historical data | Overfitting warning when out-of-sample performance degrades >20% |

**Feature-specific NFRs:**
- **Accuracy:** Suggestions based on statistically significant patterns
- **Safety:** No auto-application; manual approval required
- **Validation:** A/B testing in paper trading before live deployment

---

### 4.13 Notification Center

**Behavioral Description:**
The Notification Center sends alerts to Juragan via configured channels (Telegram primary, email secondary). It categorizes notifications by severity and ensures critical alerts are never missed.

**Notification categories:**
- **Critical:** Emergency stop, circuit breaker, API failure
- **Warning:** Daily budget 80% consumed, drawdown approaching limit
- **Info:** Trade executed, strategy optimization suggestion
- **Debug:** System health, reconnection events

[ASSUMPTION: Telegram bot token is configured for notifications]

**User Journeys:** UJ-5 (emergency alert), UJ-7 (optimization suggestion)

| # | Functional Requirement | Consequences (Testable) |
|---|------------------------|-------------------------|
| FR-80 | Center SHALL send notifications via Telegram (primary) and email (secondary) | Notification delivered within 5s of event; delivery confirmed |
| FR-81 | Center SHALL categorize notifications: critical, warning, info, debug | Categories correctly assigned; critical never suppressed |
| FR-82 | Center SHALL support notification throttling (max 10 per minute for non-critical) | Throttling applied correctly; critical alerts bypass throttling |
| FR-83 | Center SHALL support configurable notification preferences (enable/disable per category) | Preferences respected; changes take effect immediately |
| FR-84 | Center SHALL maintain notification history (last 1000) | History queryable; includes delivery status |

**Feature-specific NFRs:**
- **Latency:** Critical notifications within 5s
- **Reliability:** 99.9% delivery rate for critical notifications
- **Throttling:** Non-critical throttled to prevent spam

---

### 4.14 Backtesting

**Behavioral Description:**
The Backtesting Engine allows Juragan to test strategies against historical market data before deploying them live. It simulates the complete trading flow: scanning, opportunity detection, execution (with simulated slippage and fills), and PnL calculation.

**Key capabilities:**
- Replay historical market data through the strategy engine
- Simulate realistic execution (slippage, partial fills, latency)
- Calculate comprehensive performance metrics
- Compare multiple strategy configurations
- Identify optimal parameters

**User Journeys:** UJ-4 (backtest strategy)

| # | Functional Requirement | Consequences (Testable) |
|---|------------------------|-------------------------|
| FR-85 | Backtesting SHALL replay historical market data through strategy engine | Results deterministic (same input → same output) |
| FR-86 | Backtesting SHALL simulate realistic execution: slippage (configurable), partial fills, latency (configurable) | Simulation matches live behavior within 10% on key metrics |
| FR-87 | Backtesting SHALL calculate all performance metrics (PnL, win rate, Sharpe, drawdown, etc.) | Metrics match Analytics calculations; queryable by date range |
| FR-88 | Backtesting SHALL support parameter sweeps (test multiple configurations in batch) | Sweep completes within reasonable time (100 configs in < 1 hour) |
| FR-89 | Backtesting SHALL generate detailed reports with charts and trade-by-trade breakdown | Reports include all metrics, charts, and trade list; exportable |
| FR-90 | Backtesting SHALL detect lookahead bias (using future data) and prevent it | Lookahead bias detected and prevented; warnings logged |

**Feature-specific NFRs:**
- **Performance:** 1 year of data backtestable in < 10 minutes
- **Determinism:** Same input always produces same output
- **Accuracy:** Simulation within 10% of live behavior

---

### 4.15 Paper Trading

**Behavioral Description:**
Paper Trading runs the complete trading system in simulation mode — no real orders, but real market data. It's used for:
- Testing new strategies before going live
- Validating system changes
- Training the AI optimizer
- Confidence building for new strategies

Paper trading uses the same code path as live trading, with a "paper" flag that prevents real order placement. Simulated orders are filled based on real orderbook data.

**User Journeys:** UJ-8 (test strategy in simulation)

| # | Functional Requirement | Consequences (Testable) |
|---|------------------------|-------------------------|
| FR-91 | Paper trading SHALL use real market data but simulated execution | Market data identical to live; no real orders placed |
| FR-92 | Paper trading SHALL simulate fills based on real orderbook depth | Fill simulation realistic (within 10% of actual fill probability) |
| FR-93 | Paper trading SHALL track simulated PnL independently from live PnL | Paper PnL accurate; clearly separated from live in dashboard |
| FR-94 | Paper trading SHALL support seamless switch to live trading (same strategy config) | Switch takes effect within 1s; no configuration re-entry required |
| FR-95 | Paper trading SHALL log all simulated trades with same detail as live trades | Log format identical to live; queryable via Trade History |

**Feature-specific NFRs:**
- **Isolation:** Paper trading never affects live positions or capital
- **Realism:** Fill simulation within 10% of actual behavior
- **Switching:** Live/paper switch within 1s

---

### 4.16 Replay Mode

**Behavioral Description:**
Replay Mode allows Juragan to replay historical market events at configurable speed, watching how the bot would have reacted. It's a learning and debugging tool — not for backtesting (that's Backtesting feature), but for understanding bot behavior.

**Use cases:**
- Debug a specific trade: "Why did the bot enter this position?"
- Understand market dynamics: "How did prices move during the event?"
- Validate risk management: "Did the circuit breaker trigger correctly?"

**User Journeys:** UJ-2 (understand past behavior)

| # | Functional Requirement | Consequences (Testable) |
|---|------------------------|-------------------------|
| FR-96 | Replay SHALL replay historical market events at configurable speed (1x, 2x, 5x, 10x) | Replay accurate to historical data; speed controllable |
| FR-97 | Replay SHALL display bot decisions in real-time (what it detected, what it decided, why) | Decision log matches historical execution; reasons displayed |
| FR-98 | Replay SHALL support pause, step-forward, and rewind | Controls responsive; state consistent after any control action |
| FR-99 | Replay SHALL highlight risk events (circuit breaker triggers, limit breaches) | Risk events visually highlighted; details available on click |

**Feature-specific NFRs:**
- **Accuracy:** Replay matches historical data exactly
- **Performance:** Smooth playback at 10x speed
- **Interactivity:** Controls responsive within 100ms

---

### 4.17 AI Assistant

**Behavioral Description:**
The AI Assistant provides natural language interaction with the trading system. Juragan can ask questions like "What's my PnL this week?" or "Why did the bot enter the XYZ market?" and receive clear, data-backed answers.

**Capabilities:**
- Answer questions about trading performance
- Explain bot decisions in plain language
- Suggest risk parameter adjustments
- Provide market analysis summaries

[ASSUMPTION: LLM integration (OpenAI/Anthropic) is available and configured]

**User Journeys:** UJ-2 (ask questions), UJ-7 (get suggestions)

| # | Functional Requirement | Consequences (Testable) |
|---|------------------------|-------------------------|
| FR-100 | Assistant SHALL answer questions about trading performance using actual data | Answers accurate and data-backed; no hallucinated numbers |
| FR-101 | Assistant SHALL explain bot decisions by referencing trade logs and decision context | Explanations traceable to specific log entries; clear and understandable |
| FR-102 | Assistant SHALL suggest risk parameter adjustments based on current state | Suggestions reasonable and conservative; never suggest increasing risk beyond limits |
| FR-103 | Assistant SHALL NOT execute trades or modify configurations directly | All actions require human approval; assistant is read-only |

**Feature-specific NFRs:**
- **Accuracy:** All numerical answers verified against database
- **Safety:** Read-only access; no configuration changes
- **Latency:** Response within 5s for simple queries

---

### 4.18 Multi-Strategy

**Behavioral Description:**
Multi-Strategy support allows multiple trading strategies to run simultaneously, each with its own parameters, risk limits, and capital allocation. Strategies are isolated — a failure in one doesn't affect others.

**Strategy isolation:**
- Separate capital allocation
- Separate risk limits
- Separate position tracking
- Shared market data (Scanner)
- Shared risk oversight (Pit Boss)

**User Journeys:** UJ-7 (manage strategies), UJ-1 (multiple strategies executing)

| # | Functional Requirement | Consequences (Testestable) |
|---|------------------------|---------------------------|
| FR-104 | System SHALL support running multiple strategies simultaneously | All strategies active; no interference between strategies |
| FR-105 | System SHALL isolate strategy failures (one strategy crash doesn't affect others) | Failed strategy logged and deactivated; others continue normally |
| FR-106 | System SHALL enforce per-strategy capital allocation | Orders rejected if strategy exceeds allocation; allocation tracked accurately |
| FR-107 | System SHALL aggregate strategy performance for portfolio-level metrics | Portfolio metrics correct sum of strategy metrics; no double-counting |

**Feature-specific NFRs:**
- **Isolation:** Strategy failures contained; no cascade to other strategies
- **Resource:** Each strategy limited to configurable CPU/memory budget
- **Consistency:** Portfolio metrics consistent with strategy-level metrics

---

### 4.19 Multi-Account

**Behavioral Description:**
Multi-Account support allows managing multiple Polymarket wallets from a single PQAP instance. This enables:
- Risk segregation (test capital vs production capital)
- Strategy segregation (different strategies on different wallets)
- Performance comparison

[ASSUMPTION: User has or can create multiple Polymarket wallets]

**User Journeys:** UJ-3 (manage accounts)

| # | Functional Requirement | Consequences (Testable) |
|---|------------------------|-------------------------|
| FR-108 | System SHALL support multiple Polymarket wallet configurations | Each wallet independently configured; no shared secrets |
| FR-109 | System SHALL isolate account state (positions, PnL, risk) per wallet | Account data clearly separated; no cross-contamination |
| FR-110 | System SHALL support cross-account portfolio view (aggregate all accounts) | Aggregate view accurate; individual accounts still viewable |
| FR-111 | System SHALL enforce risk limits per account independently | Risk limits applied per account; no sharing of risk budget |

**Feature-specific NFRs:**
- **Isolation:** Account state completely independent
- **Security:** Wallet keys encrypted at rest; never shared between accounts
- **Aggregation:** Cross-account view accurate and performant

---

### 4.20 Admin Panel

**Behavioral Description:**
The Admin Panel provides system-level configuration and monitoring. It's used for:
- System configuration (API keys, risk defaults, notification settings)
- User management (single user, but role-based for future)
- System health monitoring
- Log viewing and debugging
- Database management

**User Journeys:** UJ-5 (system configuration), UJ-2 (health monitoring)

| # | Functional Requirement | Consequences (Testable) |
|---|------------------------|-------------------------|
| FR-112 | Panel SHALL provide system configuration interface (API keys, risk defaults, notification settings) | Configuration changes persisted; validation on all fields; changes logged |
| FR-113 | Panel SHALL provide system health dashboard (CPU, memory, disk, network, connections) | Health metrics accurate; updates every 5s; alerts on threshold breach |
| FR-114 | Panel SHALL provide log viewer with filtering and search | Logs queryable by level, component, date; response < 1s |
| FR-115 | Panel SHALL provide database management (backup, restore, cleanup) | Backup completes within 10 minutes for 1GB database; restore tested |
| FR-116 | Panel SHALL require authentication (even for single user) | Unauthorized access prevented; session timeout configurable |

**Feature-specific NFRs:**
- **Security:** Authentication required; session timeout; CSRF protection
- **Performance:** Log search < 1s for 1M entries
- **Backup:** Automated daily backups with 30-day retention

---

## 5. Non-Goals (Explicit)

The following are explicitly **out of scope** for PQAP:

| Category | Non-Goal | Reason |
|----------|----------|--------|
| **SaaS** | Public deployment, multi-tenant, billing | Personal use only |
| **Mobile** | Native mobile app | Web dashboard sufficient |
| **Social** | Signal sharing, copy trading, leaderboards | Not the business model |
| **Fiat** | Fiat on/off ramps | USDC on Polygon only |
| **Other chains** | Support for non-Polygon chains | Polymarket is Polygon-only |
| **Other markets** | Non-Prediction market trading | Focus on Polymarket |
| **Data sales** | Selling market data or signals | Personal use only |
| **HFT** | Sub-millisecond latency optimization | Disciplined Predator, not fastest |
| **Manual trading** | Order entry UI for manual trades | Bot is the trader |
| **KYC/Compliance** | Regulatory compliance framework | Personal use, not commercial |

---

## 6. MVP Scope

### 6.1 In Scope (MVP — Phase 1: Foundation)

| Feature | MVP Scope |
|---------|-----------|
| **Market Scanner** | WebSocket streaming, REST polling, basic market catalog |
| **Arbitrage Engine** | Simple YES+NO arbitrage only |
| **Execution Engine** | Limit orders (GTC), basic slippage protection |
| **Position Manager** | Position tracking, basic PnL, manual exit |
| **Portfolio Manager** | Capital tracking, basic position limits |
| **Risk Management** | Daily budget, max position per market, emergency stop |
| **Dashboard** | Basic portfolio view, position list, trade history |
| **Analytics** | Basic PnL tracking (daily, total) |
| **Trade History** | Complete trade logging with basic filtering |
| **Notification Center** | Telegram alerts for critical events |

### 6.2 Out of Scope for MVP

| Feature | Deferred To |
|---------|-------------|
| Cross-market arbitrage | Phase 2 |
| Liquidity capture | Phase 2 |
| AI Strategy Optimizer | Phase 3 |
| Backtesting | Phase 2 |
| Paper Trading | Phase 2 |
| Replay Mode | Phase 2 |
| AI Assistant | Phase 3 |
| Multi-Strategy | Phase 2 |
| Multi-Account | Phase 3 |
| Admin Panel | Phase 2 |
| Orderbook Viewer | Phase 2 |
| Strategy Manager | Phase 2 |
| Correlation limits | Phase 2 |
| Batasi Win | Phase 2 |
| Metabolic rate monitor | Phase 2 |

---

## 7. Success Metrics

| Metric | Target (Month 1) | Target (Month 6) | Target (Year 1) |
|--------|-------------------|-------------------|------------------|
| Monthly return | 2–5% | 5–10% | 10–20% |
| Max drawdown | < 10% | < 15% | < 20% |
| Win rate | > 55% | > 60% | > 65% |
| Uptime | > 95% | > 99% | > 99.5% |
| Trades per day | 1–5 | 5–20 | 20–100 |
| Sharpe ratio | > 1.0 | > 1.5 | > 2.0 |
| Profit factor | > 1.2 | > 1.5 | > 2.0 |
| Fill rate | > 80% | > 85% | > 90% |
| Avg slippage | < 1% | < 0.5% | < 0.3% |
| Capital utilization | > 30% | > 50% | > 70% |

**Success = consistent profitability with controlled risk, not home runs.**

---

## 8. Open Questions

| # | Question | Impact | Status |
|---|----------|--------|--------|
| OQ-1 | What is the minimum viable capital for each strategy type? | Strategy enablement | Needs research |
| OQ-2 | How does Polymarket handle market resolution disputes? Risk to open positions? | Risk management | Needs research |
| OQ-3 | What is the actual fill rate for limit orders on illiquid markets? | Opportunity scoring accuracy | Needs empirical data |
| OQ-4 | Are there rate limits on Polymarket CLOB API? What are they? | Scanner and execution design | Needs research |
| OQ-5 | How does builder attribution work? Can it reduce fees? | Cost optimization | Needs research |
| OQ-6 | What is the typical latency from order placement to fill confirmation? | Execution engine design | Needs empirical data |
| OQ-7 | How does Polymarket handle partial fills? Do they expire? | Execution engine design | Needs research |
| OQ-8 | What is the minimum order size across all markets? | Position sizing constraints | Needs research |

---

## 9. Assumptions Index

| # | Assumption | Impact if Wrong | Mitigation |
|---|-----------|-----------------|------------|
| A-1 | Polymarket CLOB API is reliable (>99% uptime) | Core functionality depends on API | Circuit breaker, fallback to polling |
| A-2 | 0% fees persist | Profitability calculations depend on this | Monitor fee announcements, adjust thresholds |
| A-3 | USDC on Polygon is stable and liquid | Capital base stability | Monitor USDC peg, alert on depeg |
| A-4 | WebSocket API provides real-time updates (<100ms latency) | Opportunity detection speed | Fallback to polling with higher frequency |
| A-5 | Sufficient liquidity exists for profitable arbitrage | Core value proposition | Focus on illiquid markets, cross-market arb |
| A-6 | $1 minimum order is sufficient for capital-efficient trading | Small account viability | Batch small orders, focus on high-score opportunities |
| A-7 | Polymarket resolution via UMA oracle is reliable | Position settlement | Monitor oracle status, manual resolution tracking |
| A-8 | Telegram bot API is available for notifications | Alert delivery | Fallback to email notifications |
| A-9 | Historical market data is available for backtesting | Strategy validation | Build data collection from day one |
| A-10 | Single-user deployment doesn't need multi-tenancy | Architecture simplification | Refactor if multi-user needed later |
| A-11 | Sufficient trade history (>100 trades) for AI optimization | AI Strategy Optimizer effectiveness | Phase 3 feature; build data collection early |
| A-12 | LLM integration (OpenAI/Anthropic) available for AI Assistant | AI Assistant functionality | Phase 3 feature; API key configuration required |
