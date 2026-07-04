# Addendum: PQAP Technical Details

## Polymarket API Details

### Overview

Polymarket operates a Central Limit Order Book (CLOB) on the Polygon blockchain. The API provides both REST and WebSocket interfaces for market data and order management.

**Base URL:** `https://clob.polymarket.com`
**WebSocket URL:** `wss://ws-subscriptions-clob.polymarket.com/ws/market`

### Authentication Levels

| Level | Requirements | Capabilities |
|-------|-------------|--------------|
| **L0** | None (public) | Read market data, orderbooks, prices |
| **L1** | EOA private key (signature_type=0) | Read account data, place orders |
| **L2** | API key + secret + passphrase | Full access, order management |

### SDK

```bash
# Python SDK (recommended for AI/Analytics layer)
pip install polymarket-client

# Go SDK (recommended for execution layer)
go github.com/Polymarket/go-polymarket
```

### Key Endpoints

#### Market Data (L0 - No Auth)

| Endpoint | Method | Purpose | Rate Limit |
|----------|--------|---------|------------|
| `/markets` | GET | List all markets | 100/min |
| `/simplified-markets` | GET | List markets (simplified) | 100/min |
| `/markets/{id}` | GET | Get market details | 100/min |
| `/book` | GET | Get orderbook for market | 300/min |
| `/books` | GET | Get orderbooks (batch) | 300/min |
| `/midpoint` | GET | Get midpoint price | 300/min |
| `/midpoints` | GET | Get midpoints (batch) | 300/min |
| `/spread` | GET | Get bid-ask spread | 300/min |
| `/spreads` | GET | Get spreads (batch) | 300/min |
| `/price` | GET | Get last traded price | 300/min |
| `/prices` | GET | Get prices (batch) | 300/min |

#### Trading (L1/L2 - Auth Required)

| Endpoint | Method | Purpose | Rate Limit |
|----------|--------|---------|------------|
| `/order` | POST | Place order | 120/min |
| `/orders` | GET | List open orders | 120/min |
| `/order/{id}` | DELETE | Cancel order | 120/min |
| `/cancel-all` | POST | Cancel all orders | 10/min |

#### Account (L1/L2 - Auth Required)

| Endpoint | Method | Purpose | Rate Limit |
|----------|--------|---------|------------|
| `/balance-allowance` | GET | Check USDC balance | 60/min |
| `/fee-rate` | GET | Query fee rate | 60/min |

### Order Types

| Type | Code | Behavior |
|------|------|----------|
| **GTC** | Good-Til-Canceled | Remains active until filled or cancelled |
| **FOK** | Fill-Or-Kill | Must fill completely immediately or cancel |
| **GTD** | Good-Til-Date | Remains active until specified date |
| **FAK** | Fill-And-Kill | Fill what's possible immediately, cancel rest |

### Tick Sizes

Markets have different tick sizes based on their configuration:

| Tick Size | Price Range | Example |
|-----------|-------------|---------|
| 0.1 | $0.00–$1.00 | Rare, high-volume markets |
| 0.01 | $0.00–$1.00 | Standard markets |
| 0.001 | $0.00–$1.00 | Precision markets |
| 0.0001 | $0.00–$1.00 | High-precision markets |

### WebSocket Subscriptions

```json
// Subscribe to market updates
{
  "type": "subscribe",
  "channel": "market",
  "assets_id": "TOKEN_ID"
}

// Subscribe to orderbook updates
{
  "type": "subscribe",
  "channel": "book",
  "assets_id": "TOKEN_ID"
}

// Subscribe to price updates
{
  "type": "subscribe",
  "channel": "price",
  "assets_id": "TOKEN_ID"
}
```

### Error Codes

| Code | Meaning | Action |
|------|---------|--------|
| 400 | Bad request | Check request parameters |
| 401 | Unauthorized | Check API credentials |
| 403 | Forbidden | Check permissions/allowances |
| 404 | Not found | Check market/order ID |
| 429 | Rate limited | Implement backoff |
| 500 | Server error | Retry with backoff |

---

## Fee Structure

### Current Fees (as of 2026-07-03)

| Fee Type | Amount | Notes |
|----------|--------|-------|
| **Maker fee** | 0% | Limit orders that add liquidity |
| **Taker fee** | 0% | Market orders that remove liquidity |
| **Withdrawal fee** | ~$0.001 | Polygon gas only |
| **Deposit fee** | 0% | USDC transfer to Polygon |

### Fee Implications

- **Zero-fee trading** makes small-margin arbitrage viable
- **No minimum fee** means even $0.01 profit is real profit
- **Gas costs negligible** — not a factor in trade decisions
- **[ASSUMPTION: Fees remain at 0%]** — monitor Polymarket announcements; fee changes would impact profitability thresholds

### Fee Optimization

Even with 0% fees, optimize for:
1. **Maker orders** (limit) over taker orders (market) — better price control
2. **Batch orders** when possible — reduce API calls
3. **Avoid excessive cancellations** — may trigger rate limits

---

## Wallet Setup Requirements

### EOA Wallet Configuration

1. **Create EOA wallet** (MetaMask or hardware wallet)
   - Export private key for bot configuration
   - [ASSUMPTION: Private key stored securely, never committed to repo]

2. **Fund with USDC on Polygon**
   - Bridge USDC from Ethereum mainnet or other chains
   - Minimum: $10 USDC
   - Recommended: $50+ for strategy flexibility

3. **Set Token Allowances**
   Polymarket requires allowances for three contracts:
   - **Main Exchange** — primary order book
   - **Neg Risk Exchange** — negative risk markets
   - **Neg Risk Adapter** — adapter for neg risk markets

   ```python
   # Python SDK example
   from polymarket.client import PolymarketClient
   
   client = PolymarketClient(private_key="YOUR_PRIVATE_KEY")
   client.set_allowances()  # Sets all required allowances
   ```

4. **API Key Generation**
   - Generate API key via Polymarket UI or API
   - Store API key, secret, and passphrase securely
   - [ASSUMPTION: API keys can be regenerated if compromised]

### Security Best Practices

- **Never commit private keys or API secrets to version control**
- **Use environment variables or secret management** (e.g., HashiCorp Vault, AWS Secrets Manager)
- **Separate wallets for testing and production**
- **Regular key rotation** (quarterly)
- **Monitor wallet for unauthorized transactions**

---

## Arbitrage Mechanics

### Simple YES+NO Arbitrage

**Concept:**
- Every binary market has YES and NO outcomes
- YES + NO should equal $1.00 (theoretical)
- When YES_price + NO_price < $1.00, buy both → guaranteed profit at resolution

**Example:**
```
Market: "Will X happen?"
YES price: $0.45
NO price: $0.50
Combined: $0.95 < $1.00

Buy YES at $0.45, Buy NO at $0.50
Total cost: $0.95
At resolution: One pays $1.00
Profit: $0.05 per share (5.3% return)
```

**Challenges:**
- Heavily competed — professional bots monitor constantly
- Margins razor-thin on liquid markets (0.1–1%)
- Must execute both legs quickly (price can move)
- Minimum order size (~$1) limits small-account profitability

**Detection Algorithm:**
```python
def detect_simple_arb(yes_price, no_price, min_profit_threshold):
    combined = yes_price + no_price
    if combined < 1.0 - min_profit_threshold:
        profit = 1.0 - combined
        return {"profit": profit, "yes_price": yes_price, "no_price": no_price}
    return None
```

### Cross-Market Arbitrage

**Concept:**
Related markets may be mispriced against each other. Example:
- Market A: "Will X happen?" → YES at $0.60
- Market B: "Will X happen by date Y?" → YES at $0.70
- If X happens by Y, then X happens → Market A YES should be >= Market B YES
- Mispricing: Market A YES ($0.60) < Market B YES ($0.70)
- Buy Market A YES, Sell Market B YES → profit when prices converge

**Relationship Types:**
1. **Subset:** "X by date Y" is a subset of "X"
2. **Mutual exclusion:** "X" and "not X" are mutually exclusive
3. **Correlation:** Markets affected by same underlying event
4. **Temporal:** Same event at different time horizons

**Detection Algorithm:**
```python
def detect_cross_market_arb(market_a, market_b, relationship_type):
    if relationship_type == "subset":
        # Market B (subset) YES should be <= Market A (superset) YES
        if market_b.yes_price > market_a.yes_price:
            profit = market_b.yes_price - market_a.yes_price
            return {"profit": profit, "buy": market_a, "sell": market_b}
    return None
```

### Liquidity Capture

**Concept:**
Wide bid-ask spreads create opportunities to place limit orders inside the spread and capture the difference.

**Example:**
```
Market: "Will Y happen?"
Best bid: $0.40 (buy)
Best ask: $0.50 (sell)
Spread: $0.10

Place limit buy at $0.42
Place limit sell at $0.48
If both fill: profit = $0.06 per share
```

**Challenges:**
- Requires both sides to fill (uncertain)
- Capital locked while orders pending
- Competition from market makers
- Risk of adverse selection (informed traders)

**Optimal Conditions:**
- Wide spreads (>5%)
- Low competition (few bots)
- High volatility (prices moving)
- Near resolution (outcome becoming clearer)

### Opportunity Scoring

Each opportunity is scored using:
```
score = spread × liquidity × fill_probability
```

Where:
- **spread** = profit margin (e.g., 0.05 for 5% profit)
- **liquidity** = orderbook depth at price level (normalized 0–1)
- **fill_probability** = estimated chance of order filling (0–1)

**Thresholds (configurable):**
- Minimum score: 0.01 (default)
- Minimum spread: 0.02 (2%)
- Minimum liquidity: 0.1 (10% of average)
- Minimum fill probability: 0.5 (50%)

---

## Capital Scaling Strategy

### Tier System

| Tier | Capital Range | Strategy Focus | Max Position | Expected Trades/Day | Risk Level |
|------|---------------|----------------|--------------|---------------------|------------|
| **Tier 1** | $10–$50 | Simple arb only | 20% of capital | 1–3 | Ultra-conservative |
| **Tier 2** | $50–$200 | + Cross-market arb | 15% of capital | 3–10 | Conservative |
| **Tier 3** | $200–$1,000 | + Liquidity capture | 10% of capital | 10–30 | Moderate |
| **Tier 4** | $1,000–$10,000 | Full strategy suite | 5% of capital | 30–100 | Standard |
| **Tier 5** | $10,000+ | + Market making | 3% of capital | 100+ | Aggressive |

### Scaling Rules

1. **Auto-tier promotion:** When capital exceeds tier threshold for 7 consecutive days
2. **Auto-tier demotion:** When capital falls below tier threshold (immediate)
3. **Manual override:** Juragan can manually set tier (with warning)
4. **Position sizing adjustment:** Max position size recalculated on tier change
5. **Strategy enablement:** New strategies enabled only at appropriate tiers

### Reinvestment Strategy

- **Default:** 80% of profits reinvested, 20% withdrawn
- **Configurable:** Reinvestment rate adjustable (0–100%)
- **Compound effect:** Reinvested capital increases position sizes automatically
- **Withdrawal:** Manual withdrawal via dashboard; automatic withdrawal on schedule (configurable)

### Capital Protection

- **Floor:** Never risk more than daily budget allows (2% default)
- **Ceiling:** Max position per market (10% default)
- **Buffer:** Maintain 20% cash reserve for opportunities
- **Emergency:** Immediate halt if capital drops below 50% of starting capital

---

## Competitive Landscape

### Market Participants

| Participant Type | Capital | Speed | Strategy | Threat Level |
|-----------------|---------|-------|----------|--------------|
| **Professional Market Makers** | $100k+ | Sub-ms | Spread capture, arb | High |
| **Retail Bots** | $1k–$10k | ms–s | Simple arb | Medium |
| **Manual Traders** | $100–$10k | s–min | Speculation | Low |
| **Arbitrage Funds** | $1M+ | Sub-ms | Cross-market, statistical | Very High |

### Competitive Advantages

1. **Disciplined Predator approach:** Not fastest, but smartest — selective hunting over raw speed
2. **Production-grade from day one:** Redundant sensing, circuit breakers, state reconciliation
3. **Casino-inspired risk management:** House edge thinking — every trade must have positive expected value
4. **Capital-efficient for small accounts:** Designed for $10+ — every dollar must work hard
5. **Cross-market arbitrage focus:** Beyond simple YES+NO — finds mispricings across related markets
6. **Smart Speed:** Fast detection, fast filtering, fast execution, fast rejection — no wasted cycles

### Competitive Disadvantages

1. **Capital disadvantage:** Professional makers have 1000x+ more capital
2. **Latency disadvantage:** Professional setups have sub-ms latency; PQAP has ms–s latency
3. **Data disadvantage:** Professionals have years of historical data and refined models
4. **Infrastructure disadvantage:** Professionals have dedicated servers, co-location

### Differentiation Strategy

1. **Focus on illiquid markets:** Less competition, wider spreads
2. **Focus on new markets:** Before professional bots discover them
3. **Focus on cross-market arb:** More complex, harder to automate
4. **Focus on risk management:** Survive longer, compound more
5. **Focus on learning:** AI optimization improves over time

### Market Data Sources

| Source | Data Type | Cost | Latency |
|--------|-----------|------|---------|
| Polymarket CLOB API | Orderbook, prices | Free | 100ms+ |
| Polymarket WebSocket | Real-time updates | Free | 50ms+ |
| Polymarket REST | Market discovery | Free | 1s+ |
| UMA Oracle | Resolution data | Free | Hours–days |

---

## Technical Architecture Details

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Next.js Frontend                        │
│                  (Dashboard, Analytics, Admin)                │
└─────────────────────┬───────────────────────────────────────┘
                      │ REST/WebSocket
┌─────────────────────┴───────────────────────────────────────┐
│                    API Gateway (FastAPI)                      │
│              (Auth, Rate Limiting, Routing)                   │
└──────┬──────────────┬──────────────┬────────────────────────┘
       │              │              │
┌──────┴──────┐ ┌─────┴─────┐ ┌─────┴─────┐
│  Go Scanner │ │ Go Engine │ │ Python AI │
│  (Market    │ │ (Execution│ │ (Analytics│
│   Data)     │ │  Arb)     │ │  ML)      │
└──────┬──────┘ └─────┬─────┘ └─────┬─────┘
       │              │              │
┌──────┴──────────────┴──────────────┴────────────────────────┐
│                        NATS/Kafka                            │
│                   (Event Streaming)                          │
└──────┬──────────────┬──────────────┬────────────────────────┘
       │              │              │
┌──────┴──────┐ ┌─────┴─────┐ ┌─────┴─────┐
│   Redis     │ │ PostgreSQL│ │TimescaleDB│
│  (Cache,    │ │ (Config,  │ │ (Time-    │
│   Risk)     │ │  History) │ │  Series)  │
└─────────────┘ └───────────┘ └───────────┘
```

### Component Responsibilities

| Component | Language | Responsibility |
|-----------|----------|----------------|
| **Market Scanner** | Go | WebSocket streaming, market catalog, REST polling |
| **Arbitrage Engine** | Go | Opportunity detection, scoring, filtering |
| **Execution Engine** | Go | Order placement, fill monitoring, circuit breakers |
| **Position Manager** | Go | Position tracking, PnL calculation, reconciliation |
| **Risk Management** | Go | Pit Boss, risk checks, budget enforcement |
| **Portfolio Manager** | Python | Capital allocation, strategy weights |
| **Analytics** | Python | Performance metrics, charts, reporting |
| **AI Strategy Optimizer** | Python | ML analysis, parameter suggestions |
| **Backtesting** | Python | Historical replay, simulation |
| **Dashboard** | Next.js | Real-time UI, charts, controls |
| **Notification Center** | Python | Telegram/email alerts |
| **Admin Panel** | Next.js | Configuration, monitoring |

### Data Flow

1. **Market Data Flow:**
   - Scanner → NATS → Arbitrage Engine
   - Scanner → Redis (cache) → Dashboard
   - Scanner → TimescaleDB (history)

2. **Opportunity Flow:**
   - Arbitrage Engine → NATS → Execution Engine
   - Arbitrage Engine → TimescaleDB (logging)

3. **Execution Flow:**
   - Execution Engine → Polymarket API
   - Execution Engine → NATS → Position Manager
   - Execution Engine → PostgreSQL (history)

4. **Risk Flow:**
   - All components → Redis (risk state)
   - Pit Boss → Redis (risk decisions)
   - Risk Management → NATS → Notification Center

### Technology Stack

| Layer | Technology | Version | Purpose |
|-------|-----------|---------|---------|
| **Execution** | Go | 1.21+ | Concurrency, low latency |
| **AI/Analytics** | Python | 3.11+ | ML libraries, prototyping |
| **API** | FastAPI | 0.100+ | Async API framework |
| **Frontend** | Next.js | 14+ | React framework, SSR |
| **Cache** | Redis | 7+ | Risk state, session cache |
| **Database** | PostgreSQL | 15+ | Relational data |
| **Time-Series** | TimescaleDB | 2.10+ | Time-series analytics |
| **Messaging** | NATS | 2.10+ | Event streaming |
| **Container** | Docker | 24+ | Containerization |
| **Orchestration** | Kubernetes | 1.28+ | Container orchestration |
| **Monitoring** | Prometheus | 2.45+ | Metrics collection |
| **Visualization** | Grafana | 10+ | Metrics dashboards |

### Database Schema (Key Tables)

```sql
-- Markets
CREATE TABLE markets (
    id UUID PRIMARY KEY,
    slug VARCHAR(255) UNIQUE NOT NULL,
    question TEXT NOT NULL,
    yes_token_id VARCHAR(255),
    no_token_id VARCHAR(255),
    end_date TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Trades
CREATE TABLE trades (
    id UUID PRIMARY KEY,
    market_id UUID REFERENCES markets(id),
    strategy VARCHAR(50) NOT NULL,
    side VARCHAR(3) NOT NULL, -- YES or NO
    price DECIMAL(10,4) NOT NULL,
    quantity DECIMAL(20,8) NOT NULL,
    fill_status VARCHAR(20) NOT NULL,
    pnl DECIMAL(20,8),
    latency_ms INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Positions
CREATE TABLE positions (
    id UUID PRIMARY KEY,
    market_id UUID REFERENCES markets(id),
    side VARCHAR(3) NOT NULL,
    entry_price DECIMAL(10,4) NOT NULL,
    current_price DECIMAL(10,4),
    quantity DECIMAL(20,8) NOT NULL,
    unrealized_pnl DECIMAL(20,8),
    status VARCHAR(20) DEFAULT 'open',
    opened_at TIMESTAMPTZ DEFAULT NOW(),
    closed_at TIMESTAMPTZ
);

-- Risk Events
CREATE TABLE risk_events (
    id UUID PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    component VARCHAR(50) NOT NULL,
    details JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Strategy Configs
CREATE TABLE strategies (
    id UUID PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    type VARCHAR(50) NOT NULL,
    config JSONB NOT NULL,
    active BOOLEAN DEFAULT false,
    version INTEGER DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

### Deployment Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                         │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ Scanner Pod │  │ Engine Pod  │  │  AI Pod     │          │
│  │ (Go)        │  │ (Go)        │  │ (Python)    │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ Dashboard   │  │ API Gateway │  │ Notification│          │
│  │ (Next.js)   │  │ (FastAPI)   │  │ (Python)    │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ Redis       │  │ PostgreSQL  │  │ TimescaleDB │          │
│  │ (StatefulSet│  │ (StatefulSet│  │ (StatefulSet│          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐                           │
│  │ NATS        │  │ Prometheus  │                           │
│  │ (StatefulSet│  │ + Grafana   │                           │
│  └─────────────┘  └─────────────┘                           │
└─────────────────────────────────────────────────────────────┘
```

### Security Considerations

1. **Secret Management:**
   - Private keys stored in Kubernetes Secrets (encrypted at rest)
   - API keys stored in environment variables or secret management
   - Never committed to version control

2. **Network Security:**
   - All internal communication via NATS (encrypted)
   - External API calls via HTTPS only
   - Rate limiting on all public endpoints

3. **Access Control:**
   - Single-user system (Juragan)
   - Authentication required for dashboard and admin
   - API keys rotated quarterly

4. **Audit Logging:**
   - All trade actions logged with timestamps
   - Risk decisions logged with full context
   - Configuration changes logged

### Monitoring & Alerting

| Metric | Threshold | Alert |
|--------|-----------|-------|
| API latency | > 500ms | Warning |
| API errors | > 5/min | Critical |
| WebSocket disconnect | Any | Warning |
| Drawdown | > 5% | Warning |
| Drawdown | > 10% | Critical |
| Daily loss | > 1.5% | Warning |
| Daily loss | > 2% | Critical |
| CPU usage | > 80% | Warning |
| Memory usage | > 1GB | Warning |
| Disk usage | > 80% | Warning |

### Disaster Recovery

1. **Database backup:** Daily automated backups, 30-day retention
2. **State recovery:** Redis state reconstructed from PostgreSQL on restart
3. **Position reconciliation:** Full reconciliation with Polymarket API on startup
4. **Emergency procedures:**
   - Circuit breaker triggers → halt trading, alert Juragan
   - API death spiral → halt trading, switch to polling
   - Data corruption → halt trading, restore from backup
