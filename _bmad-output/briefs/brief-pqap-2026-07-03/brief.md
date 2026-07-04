# Product Brief: Polymarket Quant Arbitrage Platform (PQAP)

## Executive Summary

PQAP is a personal automated trading bot for Polymarket prediction markets. It scans hundreds of binary YES/NO markets in real-time, detects mispricings, and executes trades with disciplined speed and risk management.

The platform is built on the "Disciplined Predator" philosophy: fast but not reckless, aggressive but not greedy, adaptive but not inconsistent. Every decision is data-driven, every trade is scored, every risk is budgeted.

Starting with as little as $10 in USDC on Polygon, PQAP scales with profitability — reinvesting gains to compound returns while maintaining strict risk controls that protect capital during losing streaks.

## The Problem

**Manual trading on Polymarket is inefficient and error-prone:**

- Markets move fast — by the time a human spots a mispricing, it's gone
- Polymarket has hundreds of active markets — impossible to monitor manually
- Emotional trading leads to poor decisions: chasing losses, holding too long, exiting too early
- Small capital ($10–$100) requires extreme discipline — one bad trade can wipe out weeks of gains
- No existing tool provides integrated scanning, risk management, and execution for Polymarket

**The cost of the status quo:** Missed opportunities, emotional losses, and inability to scale.

## The Solution

An autonomous trading system that:

1. **Scans** all Polymarket markets via WebSocket in real-time
2. **Detects** mispricings using opportunity scoring (spread × liquidity × fill probability)
3. **Executes** trades with smart order routing and slippage protection
4. **Manages** risk through daily budgets, position limits, and correlation checks
5. **Learns** from results — adjusting thresholds and strategies based on what works

**Core experience:** The bot runs 24/7. You check the dashboard, see profit accumulating, and adjust risk parameters. No manual trading, no emotional decisions.

## What Makes This Different

| Differentiator | Why It Matters |
|----------------|----------------|
| **Disciplined Predator approach** | Not fastest, but smartest — selective hunting over raw speed |
| **Production-grade from day one** | No MVP shortcuts — redundant sensing, circuit breakers, state reconciliation |
| **Casino-inspired risk management** | House edge thinking: every trade must have positive expected value |
| **Capital-efficient for small accounts** | Designed for $10+ — every dollar must work hard |
| **Cross-market arbitrage focus** | Beyond simple YES+NO — finds mispricings across related markets |
| **Smart Speed** | Fast detection, fast filtering, fast execution, fast rejection — no wasted cycles |

## Who This Serves

**Primary user: Juragan (personal use)**

- Starting capital: $10+, scaling with profitability
- No prior Polymarket experience — relies on the bot's intelligence
- Wants passive income from prediction market inefficiencies
- Values capital preservation over aggressive returns
- Prefers to review and approve, not manually trade

**Growth path:** As capital grows ($10 → $100 → $1,000 → $10,000+), the bot automatically adjusts position sizing, strategy allocation, and risk parameters.

## Success Criteria

| Metric | Target (Month 1) | Target (Month 6) | Target (Year 1) |
|--------|-------------------|-------------------|------------------|
| Monthly return | 2–5% | 5–10% | 10–20% |
| Max drawdown | < 10% | < 15% | < 20% |
| Win rate | > 55% | > 60% | > 65% |
| Uptime | > 95% | > 99% | > 99.5% |
| Trades per day | 1–5 | 5–20 | 20–100 |

**Success = consistent profitability with controlled risk, not home runs.**

## Scope

### In Scope (Phase 1 — Foundation)
- Polymarket CLOB API integration (Python SDK)
- WebSocket market data streaming
- Basic arbitrage detection (YES+NO < $1)
- Simple execution engine (limit orders)
- Daily risk budget enforcement
- Basic PnL tracking
- CLI dashboard

### In Scope (Phase 2 — Production)
- Advanced risk engine (position limits, correlation, drawdown)
- Multiple strategy support (classic arb, cross-market, liquidity capture)
- Web dashboard with real-time data
- Notification system (Telegram)
- Backtesting engine
- Paper trading mode

### In Scope (Phase 3 — Intelligence)
- AI strategy optimization
- Adaptive threshold adjustment
- Market regime detection
- Multi-account support
- Advanced analytics

### Explicitly Out of Scope
- Selling signals or data (personal use only)
- Public SaaS deployment
- Mobile app
- Social trading features
- Fiat on/off ramps (USDC only)

## Vision

**Year 1:** A reliable, profitable trading bot that generates consistent returns on Polymarket. The bot becomes smarter with every trade, learning which markets, times, and conditions produce the best opportunities.

**Year 2–3:** A sophisticated quant platform that can compete with professional market makers. Multiple strategies running simultaneously, AI-optimized parameters, and capital large enough to capture meaningful edges.

**The dream:** A self-sustaining trading operation where the bot hunts 24/7, compounds returns, and you focus on strategy and risk management — not execution.

## Technical Foundation

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| **Execution & Scanner** | Go (Golang) | Concurrency, low latency, WebSocket performance |
| **AI & Analytics** | Python (FastAPI) | ML libraries, backtesting, rapid prototyping |
| **Frontend** | Next.js + React | Real-time dashboard, server-side rendering |
| **Data** | Redis + PostgreSQL + TimescaleDB | Caching, relational data, time-series |
| **Infrastructure** | Docker + Kubernetes | Scalability, reliability |
| **Blockchain** | Polygon + USDC | Low gas fees, fast finality |

## Risk Philosophy

**"The house never goes broke."**

Every decision follows this hierarchy:
1. **Preserve capital** — never risk more than the daily budget allows
2. **Find edges** — only trade when the math is clear
3. **Execute fast** — once decided, act without hesitation
4. **Learn constantly** — every trade teaches something

**Key risk rules:**
- Daily loss limit: 2% of capital
- Max position per market: 10% of capital
- Correlation check: no more than 3 positions in correlated markets
- Win streak breaker: pause after 5 consecutive wins (mean reversion awareness)
- Emergency stop: immediate halt if API errors exceed threshold
