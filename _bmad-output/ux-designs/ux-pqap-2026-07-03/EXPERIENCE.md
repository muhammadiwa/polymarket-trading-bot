---
name: PQAP Dashboard Experience
status: draft
created: 2026-07-03
updated: 2026-07-03
---

# EXPERIENCE.md — PQAP Dashboard

## Foundation

### Form Factor

**Web dashboard, desktop-first, tablet-compatible.**

| Breakpoint | Layout | Behavior |
|------------|--------|----------|
| ≥1440px | Full 12-column grid, expanded sidebar | Optimal experience |
| ≥1024px | 12-column grid, collapsible sidebar | Laptop experience |
| ≥768px | 8-column grid, sidebar overlay | Tablet experience |
| <768px | Not supported for v1 | Show "use desktop" message |

**Screen real estate strategy:** Information density is high by design — this is a trading terminal. Every pixel serves a purpose. The user (Juragan) monitors this dashboard during active trading sessions and for morning reviews. Quick-glance readability is paramount.

### UI System

**shadcn/ui** as the base component library, extended with custom glassmorphism theme. All shadcn components are themed to match the PQAP design tokens:

- **Base:** shadcn/ui components (Button, Card, Table, Input, Badge, etc.)
- **Theme:** Custom dark theme with glass effects applied via CSS overrides
- **Extensions:** Custom components for financial data (PnLCard, PositionRow, OpportunityFeed)
- **Icons:** Lucide icons (consistent with shadcn/ui)
- **Charts:** Recharts with custom PQAP theme
- **Animations:** Framer Motion for micro-interactions

### Technology Stack

| Layer | Technology | Purpose |
|-------|------------|---------|
| Framework | Next.js 16.2.10 (LTS) | React SSR, routing, API routes |
| UI Library | React 19 | Component architecture |
| Styling | TailwindCSS 4 | Utility-first CSS |
| Components | shadcn/ui | Base component system |
| Charts | Recharts | PnL charts, distributions |
| State | Zustand | Client-side state management |
| WebSocket | Native WebSocket API | Real-time data from api-gateway |
| Motion | Framer Motion | Micro-animations |
| Fonts | JetBrains Mono + Inter | Monospace + sans-serif |

## Information Architecture

### Navigation Structure

```
Dashboard (Portfolio Overview)
├── Positions
│   ├── Active Positions
│   └── Position History
├── Trades
│   ├── Trade History
│   └── Trade Detail
├── Analytics
│   ├── Performance
│   ├── PnL Analysis
│   └── Strategy Breakdown
├── Strategies
│   ├── Strategy List
│   ├── Strategy Config
│   └── Backtesting
├── Risk
│   ├── Risk Dashboard
│   ├── Circuit Breakers
│   └── Risk Events
├── Orderbook
│   └── Market Viewer
├── Notifications
│   └── Alert History
└── Admin
    ├── System Health
    ├── Configuration
    └── Logs
```

### Sidebar Navigation

**Structure:**
```
┌─────────────────────────────┐
│  PQAP Logo                  │
│  ─────────────────────────  │
│  ● Dashboard                │  ← Home, portfolio overview
│  ○ Positions                │  ← Active positions table
│  ○ Trades                   │  ← Trade history
│  ○ Analytics                │  ← Performance charts
│  ○ Strategies               │  ← Strategy management
│  ○ Risk                     │  ← Risk monitoring
│  ○ Orderbook                │  ← Market viewer
│  ○ Notifications            │  ← Alert center
│  ─────────────────────────  │
│  ○ Admin                    │  ← System settings
│  ─────────────────────────  │
│  ⚡ Bot Status: RUNNING     │  ← Live indicator
│  📊 Daily PnL: +$12.45     │  ← Quick metric
└─────────────────────────────┘
```

**Navigation behavior:**
- **Active state:** `bg-glass-active` background, `accent` left border (3px), `text-primary` text
- **Hover:** `bg-glass-hover` background, `text-primary` text
- **Default:** Transparent background, `text-secondary` text
- **Badges:** Show count for notifications, risk alerts
- **Quick metrics:** Bottom of sidebar shows bot status and daily PnL

### Page Hierarchy

**Primary pages (always in sidebar):**
1. **Dashboard** — Portfolio overview, quick actions, live feed
2. **Positions** — Active positions with real-time PnL
3. **Trades** — Complete trade history with filtering
4. **Analytics** — Performance charts and metrics

**Secondary pages (sidebar):**
5. **Strategies** — Strategy configuration and management
6. **Risk** — Risk monitoring and circuit breakers
7. **Orderbook** — Market depth viewer

**Utility pages:**
8. **Notifications** — Alert history and preferences
9. **Admin** — System health, config, logs

### Top Bar

```
┌─────────────────────────────────────────────────────────────────┐
│  [Sidebar Toggle]  Page Title                    [Quick Actions] │
│                                                   ● Connected    │
│                                                   [Emergency]    │
│                                                   [Notifications]│
│                                                   [Profile]      │
└─────────────────────────────────────────────────────────────────┘
```

**Top bar elements:**
- **Left:** Sidebar toggle button, current page title
- **Right:**
  - Connection status indicator (green dot + "Connected")
  - Emergency Stop button (always visible, red, requires confirmation)
  - Notification bell with unread count badge
  - Profile/settings dropdown

## Voice and Tone

### Personality

PQAP's interface communicates like a **senior trading desk operator** — precise, confident, data-driven, and occasionally sharp. The UI never wastes words. Every label, tooltip, and message serves a purpose.

### Microcopy Guidelines

**Principles:**
1. **Be precise** — "PnL: +$12.45" not "You made some money"
2. **Be concise** — "5 positions open" not "You currently have 5 active positions"
3. **Be actionable** — "Resume trading" not "Trading is paused"
4. **Be consistent** — same terms everywhere (PnL, not P&L or profit/loss)

**Terminology (Bahasa Indonesia for UI, English for technical terms):**

| Technical Term | UI Display | Notes |
|----------------|------------|-------|
| PnL | PnL | Always abbreviated, always monospace |
| Position | Posisi | Indonesian for UI labels |
| Trade | Trade | Keep English — industry standard |
| Strategy | Strategi | Indonesian |
| Risk | Risiko | Indonesian |
| Capital | Modal | Indonesian |
| Drawdown | Drawdown | Keep English — industry standard |
| Circuit Breaker | Circuit Breaker | Keep English |
| Win Rate | Win Rate | Keep English |
| Sharpe Ratio | Sharpe Ratio | Keep English |

**Status messages:**
| State | Message |
|-------|---------|
| Bot running | "Bot aktif — memindai pasar" |
| Bot paused | "Bot dijeda — klik untuk melanjutkan" |
| Bot stopped | "Bot berhenti — periksa log" |
| Circuit breaker | "Circuit breaker aktif — trading dihentikan" |
| Emergency stop | "STOP DARURAT — aksi manual diperlukan" |
| No positions | "Tidak ada posisi terbuka" |
| No trades | "Belum ada trade" |
| Loading | "Memuat..." |
| Error | "Gagal memuat — coba lagi" |

**Tooltips:**
- Keep under 80 characters
- Explain what, not how
- Use monospace for values: "PnL hari ini: +$12.45"

### Number Formatting

| Type | Format | Example |
|------|--------|---------|
| USD currency | `$X.XX` | `$1,234.56` |
| Percentage | `+X.XX%` / `-X.XX%` | `+12.34%` |
| Price | `$X.XXXX` | `$0.6250` |
| Quantity | `X.XXXXXXXX` | `150.00000000` |
| Timestamp | `HH:MM:SS` | `14:32:05` |
| Date | `DD MMM YYYY` | `03 Jul 2026` |
| Duration | `Xh Xm` | `2h 15m` |

## Component Patterns

### Card Patterns

**Stat Card (Portfolio Overview):**
```
┌──────────────────────────────────────┐
│  Total Modal              [icon]     │
│  $1,234.56                          │  ← JetBrains Mono, 28px, text-primary
│  +$12.45 hari ini                    │  ← profit color, 13px mono
└──────────────────────────────────────┘
```
- Large number is the hero
- Trend indicator below (profit/loss color)
- Icon in top-right corner (contextual)
- Hover: subtle lift effect

**PnL Card:**
```
┌──────────────────────────────────────┐
│  PnL Hari Ini                        │
│  ─────────────────────────────────── │
│  +$12.45                             │  ← JetBrains Mono, 32px, profit color
│  ████████████████░░░░ +1.2%          │  ← Mini progress bar
│  Target: $20.00                      │  ← text-secondary
└──────────────────────────────────────┘
```

**Position Card:**
```
┌──────────────────────────────────────┐
│  BTC > $100k by Dec    YES  0.6250   │  ← Market + current price
│  ─────────────────────────────────── │
│  Entry: $0.5800    Current: $0.6250  │
│  Qty: 150.00       PnL: +$6.75      │  ← profit color
│  ─────────────────────────────────── │
│  [Close Position]           [Detail] │
└──────────────────────────────────────┘
```

### Table Patterns

**Position Table:**
```
┌─────────────────────────────────────────────────────────────────┐
│  Market                    Side  Entry   Current  Qty    PnL    │
│  ─────────────────────────────────────────────────────────────  │
│  BTC > $100k by Dec       YES   0.5800  0.6250   150   +$6.75  │
│  ETH > $5k by Jun         NO    0.3200  0.2800   200   +$8.00  │
│  Fed rate cut Jul         YES   0.4500  0.5200   100   +$7.00  │
│  ─────────────────────────────────────────────────────────────  │
│  Total: 3 positions                              PnL: +$21.75  │
└─────────────────────────────────────────────────────────────────┘
```
- Numbers right-aligned and monospace
- PnL colored (profit/loss)
- Side column: YES = accent, NO = text-secondary
- Hover row: bg-glass-hover
- Click row: Navigate to detail

**Trade History Table:**
```
┌──────────────────────────────────────────────────────────────────┐
│  Time        Market              Side  Price   Qty    PnL  Status│
│  ──────────────────────────────────────────────────────────────  │
│  14:32:05    BTC > $100k        YES   0.5800  150   +$6.75  ✓   │
│  14:31:58    ETH > $5k          NO    0.3200  200   +$8.00  ✓   │
│  14:31:45    Fed rate cut       YES   0.4500  100   -$2.00  ✓   │
│  ──────────────────────────────────────────────────────────────  │
│  Showing 1-50 of 1,234 trades              [Export CSV]          │
└──────────────────────────────────────────────────────────────────┘
```
- Timestamps in mono-xs
- Status: checkmark for filled, X for failed, clock for pending
- Sortable columns
- Pagination or virtual scroll

### Feed Patterns

**Opportunity Feed:**
```
┌──────────────────────────────────────────────────────────────┐
│  Live Opportunities                          [Filter ▾]       │
│  ─────────────────────────────────────────────────────────── │
│  ● 14:32:05  BTC > $100k     Score: 0.85  Spread: 2.3%  ▸   │
│  ● 14:31:58  ETH > $5k       Score: 0.72  Spread: 1.8%  ▸   │
│  ○ 14:31:45  Fed rate cut    Score: 0.45  Spread: 1.2%  ✗   │  ← Filtered
│  ○ 14:31:30  CPI data        Score: 0.38  Spread: 0.9%  ✗   │  ← Filtered
│  ─────────────────────────────────────────────────────────── │
│  Auto-scroll: ON                              [Clear]         │
└──────────────────────────────────────────────────────────────┘
```
- Live indicator (green dot) for new items
- Score color-coded (high = profit, low = text-muted)
- Filtered items shown but dimmed
- Click to expand details
- Auto-scroll with pause on hover

### Chart Patterns

**PnL Over Time:**
```
┌──────────────────────────────────────────────────────────────┐
│  PnL Trend                   [7D] [30D] [90D] [1Y] [All]    │
│  ─────────────────────────────────────────────────────────── │
│  $50 ┤                                    ╭────              │
│  $40 ┤                              ╭────╯                   │
│  $30 ┤                        ╭────╯                         │
│  $20 ┤                  ╭────╯                               │
│  $10 ┤            ╭────╯                                     │
│   $0 ┼──────╮────╯                                           │
│ -$10 ┤      ╰──╯                                             │
│      └──────────────────────────────────────────────────     │
│       Jun 26    Jun 28    Jun 30    Jul 02    Jul 03         │
│  ─────────────────────────────────────────────────────────── │
│  Total PnL: +$45.23    Win Rate: 68%    Sharpe: 1.82        │
└──────────────────────────────────────────────────────────────┘
```
- Area chart with gradient fill
- Profit area: gradient from profit (20%) to transparent
- Loss area: gradient from loss (20%) to transparent
- Zero line: dashed border
- Tooltip: glass card with detailed values
- Time range selector: pill buttons

## State Patterns

### Loading States

**Skeleton loaders** for all data-fetching components:

```
┌──────────────────────────────────────┐
│  ████████████              ████      │  ← Title skeleton
│  ─────────────────────────────────── │
│  ████████████████████████████████    │  ← Number skeleton
│  ████████████████                    │  ← Subtext skeleton
└──────────────────────────────────────┘
```

- Skeleton color: `bg-glass` with subtle pulse animation
- Duration: 2s pulse cycle
- Show after 300ms delay (avoid flash for fast loads)

### Empty States

**No positions:**
```
┌──────────────────────────────────────────────────────────────┐
│                                                              │
│                    📊                                        │
│            Tidak ada posisi terbuka                          │
│                                                              │
│      Bot sedang memindai pasar untuk peluang arbitrase       │
│                                                              │
│              [Lihat Opportunity Feed →]                      │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

- Gradient background
- Centered content
- Icon + title + description + CTA
- Subtle fade-in animation

### Error States

**Connection error:**
```
┌──────────────────────────────────────────────────────────────┐
│  ⚠️  Koneksi terputus                                        │
│  ─────────────────────────────────────────────────────────── │
│  WebSocket connection to api-gateway lost.                   │
│  Last connected: 14:32:05                                    │
│                                                              │
│  [Retry Connection]                          Status: ● Error │
└──────────────────────────────────────────────────────────────┘
```

- Warning icon + clear message
- Last known state timestamp
- Retry action button
- Status indicator shows error state

**Data fetch error:**
```
┌──────────────────────────────────────────────────────────────┐
│  Gagal memuat data                                           │
│  ─────────────────────────────────────────────────────────── │
│  Unable to fetch position data. Please try again.            │
│                                                              │
│  [Coba Lagi]                                                 │
└──────────────────────────────────────────────────────────────┘
```

### Success States

**Trade executed:**
```
┌──────────────────────────────────────────────────────────────┐
│  ✓  Trade dieksekusi                                         │
│  ─────────────────────────────────────────────────────────── │
│  BTC > $100k    YES    150 @ $0.5800                         │
│  PnL: +$6.75                                                 │
│                                                              │
│  [Lihat Posisi]                                              │
└──────────────────────────────────────────────────────────────┘
```

- Green checkmark icon
- Trade details summary
- Action button to view result
- Auto-dismiss after 5 seconds (toast pattern)

### Real-Time Data Updates

**Live indicators:**
- Green dot with pulse animation for connected/live status
- Number flash on value change (150ms glow)
- Row highlight on new data (fade from accent-muted to transparent)
- Feed items slide in from top

**Update cadence:**
| Data | Update Frequency | Animation |
|------|------------------|-----------|
| Prices | Real-time (WebSocket) | Number roll |
| PnL | Real-time | Color flash |
| Positions | On change | Row highlight |
| Opportunities | Real-time | Slide in |
| System health | Every 5s | Subtle transition |
| Risk status | On change | Color change |

## Interaction Primitives

### Hover Effects

| Element | Hover Effect |
|---------|--------------|
| Card | Scale 1.01, shadow increase, border brighten |
| Table row | `bg-glass-hover` background |
| Button | Background lighten, cursor pointer |
| Link | `accent` color, underline |
| Badge | Subtle brightness increase |
| Sidebar item | `bg-glass-hover` background |

### Click Feedback

| Element | Click Effect |
|---------|--------------|
| Button | Scale 0.98, background darken (150ms) |
| Card (clickable) | Scale 0.99, then navigate |
| Table row | Navigate to detail page |
| Checkbox | Toggle with scale animation |
| Toggle | Slide with color transition |

### Transitions

| Interaction | Duration | Easing | Property |
|-------------|----------|--------|----------|
| Hover | 200ms | ease | background, border, shadow, transform |
| Active | 150ms | ease-in | transform, background |
| State change | 300ms | ease-in-out | color, opacity |
| Page transition | 300ms | ease-out | opacity, transform |
| Modal open | 300ms | ease-out | opacity, scale |
| Modal close | 200ms | ease-in | opacity, scale |

### Keyboard Navigation

| Key | Action |
|-----|--------|
| `Tab` | Move focus between interactive elements |
| `Enter` / `Space` | Activate focused element |
| `Escape` | Close modal, cancel action, blur input |
| `↑` / `↓` | Navigate table rows, feed items |
| `←` / `→` | Navigate sidebar, tab panels |
| `Ctrl+K` | Open command palette (future) |

### Focus States

- **Focus ring:** 2px `accent` outline, 2px offset
- **Focus visible:** Only show focus ring on keyboard navigation, not mouse clicks
- **Skip link:** "Skip to main content" for screen readers

## Accessibility Floor

### WCAG 2.1 AA Compliance

**Color contrast:**
- `text-primary` (#ffffff) on `bg-primary` (#0a0e17) = 17.4:1 ✓
- `text-secondary` (#8892a4) on `bg-primary` (#0a0e17) = 5.9:1 ✓
- `profit` (#00ff88) on `bg-primary` (#0a0e17) = 11.3:1 ✓
- `loss` (#ff4757) on `bg-primary` (#0a0e17) = 4.8:1 ✓

**Required implementations:**
- All images have alt text
- All interactive elements have focus states
- All form inputs have labels
- All data tables have headers
- All status changes announced to screen readers (aria-live)
- All modals trap focus
- All pages have skip-to-content link

**Screen reader support:**
- Use semantic HTML (nav, main, aside, header, footer)
- ARIA labels for icon-only buttons
- ARIA live regions for real-time data updates
- Role attributes for custom components (feed, chart)

**Keyboard navigation:**
- All interactive elements focusable
- Logical tab order
- Focus visible on keyboard navigation
- Escape closes modals and menus

### Reduced Motion

```css
@media (prefers-reduced-motion: reduce) {
  * {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
```

## Key Flows

### Flow 1: Juragan Checks Morning Dashboard

**Protagonist:** Juragan (quant trader)
**Goal:** Review overnight performance and bot status before starting the day
**Frequency:** Daily, first thing in the morning

**Journey:**

```
1. Open dashboard URL
   → Page loads with skeleton loaders
   → Portfolio overview renders (300ms)
   → Shows: total capital, daily PnL, overnight PnL, active positions

2. Scan portfolio overview cards
   → "Total Modal: $1,234.56"
   → "PnL Hari Ini: +$12.45" (green)
   → "PnL Kemarin: +$8.30" (green)
   → "Posisi Aktif: 5"
   → "Modal Terpakai: 68%"

3. Review active positions
   → Table shows 5 positions with real-time PnL
   → Sort by PnL descending
   → Identify best performer: "BTC > $100k: +$6.75"
   → Identify worst performer: "Fed rate cut: -$2.00"

4. Check risk status
   → "Budget Harian: $24.69 tersisa dari $25.00"
   → "Drawdown: 2.3%"
   → "Win Streak: 3"
   → Status: ● Normal

5. Scan opportunity feed
   → 12 opportunities detected overnight
   → 8 executed, 4 filtered
   → Best opportunity: "CPI data, Score: 0.92"

6. Check system health
   → "WebSocket: ● Connected"
   → "CPU: 12%"
   → "Memory: 245MB"
   → "Error Rate: 0.1%"

7. Satisfied — bot running normally
   → Continue with daily routine
```

**Emotional arc:** Uncertainty → Reassurance → Confidence

### Flow 2: Juragan Handles Emergency (Circuit Breaker)

**Protagonist:** Juragan (quant trader)
**Goal:** Respond to circuit breaker trip, diagnose issue, resume trading
**Frequency:** Rare (as needed)

**Journey:**

```
1. Receive Telegram notification
   → "⚠️ Circuit Breaker aktif — 5 consecutive API errors"
   → "Trading dihentikan. Aksi manual diperlukan."

2. Open dashboard urgently
   → Top bar shows: ● Circuit Breaker (red, pulsing)
   → Emergency status banner at top of page

3. Review risk dashboard
   → "Circuit Breaker: OPEN"
   → "Trigger: 5 consecutive API errors"
   → "Last error: timeout on CLOB API"
   → "Time triggered: 14:32:05"
   → "Positions: 5 open (not affected)"

4. Check system health
   → "Polymarket API: ● Error"
   → "WebSocket: ● Connected"
   → "Redis: ● Connected"
   → Diagnosis: Polymarket API temporary outage

5. Wait for API recovery
   → Monitor system health page
   → API status changes to: ● Connected

6. Reset circuit breaker
   → Click "Reset Circuit Breaker" button
   → Confirmation modal: "Reset circuit breaker and resume trading?"
   → Confirm with button

7. Resume trading
   → Bot resumes scanning and executing
   → Status changes to: ● Running
   → Telegram notification: "Trading resumed"

8. Review missed opportunities
   → Check opportunity feed for downtime period
   → 3 opportunities missed during 5-minute outage
```

**Emotional arc:** Alert → Investigation → Understanding → Action → Resolution

### Flow 3: Juragan Backtests Strategy

**Protagonist:** Juragan (quant trader)
**Goal:** Test new strategy parameters against historical data before deploying
**Frequency:** Weekly

**Journey:**

```
1. Navigate to Strategies page
   → List of strategies with status indicators
   → Select "Simple YES+NO Arb" strategy

2. Open backtesting panel
   → Configure parameters:
     - Min profit threshold: 0.02
     - Score threshold: 0.015
     - Position size: 10% of capital
   → Select date range: "Last 30 days"
   → Click "Run Backtest"

3. Wait for backtest execution
   → Progress bar: "Processing 1,234 market snapshots..."
   → Estimated time: 2 minutes
   → Continue browsing other pages (non-blocking)

4. Review backtest results
   → "Total PnL: +$45.23"
   → "Win Rate: 68%"
   → "Sharpe Ratio: 1.82"
   → "Max Drawdown: 4.2%"
   → "Trades: 89"
   → PnL chart shows equity curve

5. Compare with current parameters
   → Side-by-side comparison:
     - Current: +$32.10 PnL, 62% win rate
     - New: +$45.23 PnL, 68% win rate
   → Improvement: +40% PnL, +6% win rate

6. Deploy new parameters
   → Click "Deploy to Paper Trading"
   → Confirmation: "Test in paper trading before going live?"
   → Confirm

7. Monitor paper trading performance
   → Switch to paper mode in sidebar
   → Track simulated PnL with new parameters
   → After 1 week: +$12.45 paper PnL

8. Go live
   → Click "Deploy to Live"
   → Confirmation: "Switch to live trading with new parameters?"
   → Confirm with password
   → Bot starts using new parameters
```

**Emotional arc:** Curiosity → Configuration → Anticipation → Analysis → Confidence → Action

### Flow 4: Juragan Reviews Trade History

**Protagonist:** Juragan (quant trader)
**Goal:** Analyze past trades to identify patterns and improve strategy
**Frequency:** Weekly

**Journey:**

```
1. Navigate to Trades page
   → Trade history table loads
   → 1,234 trades in last 30 days
   → Default sort: newest first

2. Apply filters
   → Date range: "Last 7 days"
   → Strategy: "Simple YES+NO"
   → PnL: "Profitable only"
   → Filtered results: 45 trades

3. Analyze patterns
   → Sort by PnL descending
   → Top trade: "BTC > $100k, +$12.50"
   → Notice: Most profitable trades on BTC markets

4. Export data
   → Click "Export CSV"
   → Download 45 trades with all details

5. Review in spreadsheet
   → Calculate additional metrics
   → Identify: BTC markets have 78% win rate
   → Identify: ETH markets have 52% win rate

6. Adjust strategy
   → Navigate to Strategies
   → Increase allocation to BTC markets
   → Decrease allocation to ETH markets
   → Save changes

7. Monitor impact
   → Track performance over next week
   → Win rate improves from 62% to 68%
```

**Emotional arc:** Investigation → Discovery → Insight → Action → Validation

### Flow 5: Juragan Monitors Risk During Volatility

**Protagonist:** Juragan (quant trader)
**Goal:** Ensure risk limits are respected during high-volatility market events
**Frequency:** As needed during market events

**Journey:**

```
1. Receive warning notification
   → "⚠️ Budget harian 80% terpakai ($20.00 dari $25.00)"

2. Open risk dashboard
   → "Daily Budget: $5.00 remaining"
   → "Drawdown: 4.2%"
   → "Open Positions: 8"
   → "Correlation: 3 BTC positions (warning)"

3. Review position exposure
   → Positions page shows 3 BTC-correlated markets
   → Total BTC exposure: $450 (36% of capital)
   → Risk indicator: ● Warning

4. Decide to reduce exposure
   → Select smallest BTC position
   → Click "Close Position"
   → Confirmation: "Close BTC > $150k at market?"
   → Confirm

5. Monitor risk improvement
   → Budget remaining: $5.00 (unchanged)
   → BTC exposure: $300 (24% of capital)
   → Correlation: 2 BTC positions (normal)
   → Risk indicator: ● Normal

6. Adjust daily budget
   → Increase daily budget from $25 to $30
   → Reasoning: Higher volatility = more opportunities
   → Save changes

7. Continue monitoring
   → Bot continues trading within new limits
   → Track performance for rest of day
```

**Emotional arc:** Alert → Assessment → Decision → Action → Relief → Adjustment

## Responsive & Platform

### Desktop (≥1440px)

- Full 12-column grid
- Sidebar always visible (280px)
- All panels and cards visible
- Optimal information density
- Multi-panel layouts (chart + table side by side)

### Laptop (≥1024px)

- 12-column grid with tighter spacing
- Sidebar collapsible (280px → 72px)
- Cards stack slightly more
- Charts maintain readability
- All features accessible

### Tablet (≥768px)

- 8-column grid
- Sidebar as overlay (hamburger menu)
- Cards stack vertically
- Tables become scrollable
- Charts resize proportionally
- Touch-friendly tap targets (48px minimum)

### Mobile (<768px)

- **Not supported for v1**
- Show message: "Gunakan desktop atau tablet untuk pengalaman terbaik"
- Link to Telegram for critical notifications

### Responsive Breakpoint Behavior

| Component | Desktop | Laptop | Tablet |
|-----------|---------|--------|--------|
| Sidebar | Expanded (280px) | Collapsible | Overlay |
| Stat cards | 4 columns | 4 columns | 2 columns |
| Tables | Full width | Full width | Horizontal scroll |
| Charts | Full width | Full width | Full width |
| Feed | Side panel | Side panel | Full width |
| Modals | Centered (600px) | Centered (500px) | Full width |
| Buttons | Default size | Default size | Large (48px) |

### Touch Considerations (Tablet)

- Minimum tap target: 48px × 48px
- Swipe gestures: swipe left to close position detail
- Long press: context menu for additional actions
- Pull to refresh: refresh data feed
- Pinch to zoom: chart zoom

### Print Styles

- Not a priority for v1
- If needed: white background, hide sidebar, show all data
- Print-specific: hide live indicators, animations, interactive elements
