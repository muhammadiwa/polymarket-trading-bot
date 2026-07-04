# Addendum: PQAP Technical Details

## Polymarket Platform Details

### How It Works
- Binary YES/NO markets with outcomes priced $0.00–$1.00
- YES + NO always sums to $1.00 (in theory)
- Built on Polygon (Chain ID 137) using USDC as collateral
- Resolution via UMA oracle system with dispute period (2 hours to 2 days)

### API Details
- **Base URL:** `https://clob.polymarket.com`
- **Auth levels:** L0 (no auth, read-only), L1 (private key), L2 (API key + secret + passphrase)
- **SDK:** `pip install polymarket-client` (new unified Python SDK)
- **Order types:** GTC, FOK, GTD, FAK
- **Tick sizes:** 0.1, 0.01, 0.001, 0.0001 (market-dependent)

### Key Endpoints
| Category | Endpoint | Purpose |
|----------|----------|---------|
| Market Data | `/book`, `/books` | Orderbook data |
| Market Data | `/midpoint`, `/midpoints` | Midpoint prices |
| Market Data | `/spread`, `/spreads` | Bid-ask spreads |
| Market Data | `/markets`, `/simplified-markets` | Market discovery |
| Trading | `/order`, `/orders` | Place/cancel orders |
| Account | `/balance-allowance` | Check balance |
| Account | `/fee-rate` | Query fee rate |

### Fees
- **Trading fees:** Currently 0% (maker and taker)
- **Withdrawal:** Polygon gas only (fractions of a cent)
- **No deposit fees**

### Minimum Orders
- ~$1.00 per order for most markets
- Minimum price: 0.001
- Size = shares (not dollars)

### Wallet Setup
- EOA wallet (MetaMask or hardware) with signature_type=0
- Fund with USDC on Polygon
- Set token allowances for: Main exchange, Neg risk exchange, Neg risk adapter

## Arbitrage Mechanics

### Simple YES+NO Arbitrage
- Buy YES at $X, buy NO at $Y, where X + Y < $1.00
- At resolution, one pays $1.00
- Profit = $1.00 - X - Y
- **Challenge:** Heavily competed, margins razor-thin on liquid markets

### Cross-Market Arbitrage
- Related markets may be mispriced against each other
- Example: "Will X happen?" and "Will X happen by date Y?" should be correlated
- More opportunity but requires sophisticated detection

### Best Opportunities
- Illiquid/new markets with wide spreads
- High volatility periods (major news events)
- Near-resolution markets where outcomes become clearer
- Markets with low bot competition

## Capital Scaling Strategy

| Capital Level | Strategy Focus | Expected Trades/Day |
|---------------|----------------|---------------------|
| $10–$50 | Simple arb only, ultra-selective | 1–3 |
| $50–$200 | Add cross-market arb | 3–10 |
| $200–$1,000 | Add liquidity capture | 10–30 |
| $1,000–$10,000 | Full strategy suite | 30–100 |
| $10,000+ | Market making component | 100+ |

## Competitive Landscape

- Professional market makers actively operate on Polymarket
- `py-clob-client` has 1.2k stars and 392 forks — heavy developer interest
- `@polymarket/clob-client` has 21k+ weekly npm downloads
- Pure YES+NO arbitrage is heavily competed
- Cross-market and illiquid market opportunities remain viable
- Polymarket has builder attribution system supporting bot ecosystem

## Key Risks

1. **Capital lockup** — funds locked until market resolution (hours to days)
2. **Oracle delays** — UMA resolution can take 2 hours to 2 days
3. **Competition** — professional bots with more capital and speed
4. **Liquidity risk** — illiquid markets may not fill orders
5. **Smart contract risk** — Polygon/Polymarket contract vulnerabilities
6. **Regulatory risk** — prediction market regulation changes
