---
name: PQAP Dashboard
status: draft
created: 2026-07-03
updated: 2026-07-03
tokens:
  colors:
    - name: bg-primary
      value: "#0a0e17"
    - name: bg-secondary
      value: "#1a1f2e"
    - name: bg-tertiary
      value: "#0f1320"
    - name: bg-glass
      value: "rgba(255,255,255,0.05)"
    - name: bg-glass-hover
      value: "rgba(255,255,255,0.08)"
    - name: bg-glass-active
      value: "rgba(255,255,255,0.12)"
    - name: profit
      value: "#00ff88"
    - name: profit-muted
      value: "rgba(0,255,136,0.15)"
    - name: loss
      value: "#ff4757"
    - name: loss-muted
      value: "rgba(255,71,87,0.15)"
    - name: accent
      value: "#00d4ff"
    - name: accent-muted
      value: "rgba(0,212,255,0.15)"
    - name: warning
      value: "#ffa726"
    - name: warning-muted
      value: "rgba(255,167,38,0.15)"
    - name: text-primary
      value: "#ffffff"
    - name: text-secondary
      value: "#8892a4"
    - name: text-muted
      value: "#4a5568"
    - name: border
      value: "rgba(255,255,255,0.1)"
    - name: border-hover
      value: "rgba(255,255,255,0.2)"
    - name: gradient-primary
      value: "linear-gradient(135deg, #0a0e17 0%, #1a1f2e 100%)"
    - name: gradient-accent
      value: "linear-gradient(135deg, #00d4ff 0%, #00ff88 100%)"
    - name: gradient-profit
      value: "linear-gradient(135deg, #00ff88 0%, #00d4ff 100%)"
    - name: gradient-loss
      value: "linear-gradient(135deg, #ff4757 0%, #ff6b81 100%)"
  typography:
    - name: font-mono
      value: "JetBrains Mono, monospace"
    - name: font-sans
      value: "Inter, sans-serif"
    - name: font-display
      value: "Inter, sans-serif"
  rounded:
    - name: card
      value: "16px"
    - name: card-sm
      value: "12px"
    - name: button
      value: "8px"
    - name: button-lg
      value: "12px"
    - name: badge
      value: "20px"
    - name: badge-sm
      value: "6px"
    - name: input
      value: "8px"
    - name: full
      value: "9999px"
  spacing:
    - name: card-padding
      value: "24px"
    - name: card-padding-sm
      value: "16px"
    - name: gap
      value: "16px"
    - name: gap-sm
      value: "8px"
    - name: gap-lg
      value: "24px"
    - name: gap-xl
      value: "32px"
    - name: sidebar-width
      value: "280px"
    - name: sidebar-collapsed
      value: "72px"
  blur:
    - name: glass
      value: "12px"
    - name: glass-heavy
      value: "20px"
    - name: backdrop
      value: "40px"
  shadow:
    - name: card
      value: "0 4px 24px rgba(0,0,0,0.3)"
    - name: card-hover
      value: "0 8px 32px rgba(0,0,0,0.4)"
    - name: glow-profit
      value: "0 0 20px rgba(0,255,136,0.3)"
    - name: glow-loss
      value: "0 0 20px rgba(255,71,87,0.3)"
    - name: glow-accent
      value: "0 0 20px rgba(0,212,255,0.3)"
---

# DESIGN.md — PQAP Dashboard

## Brand & Style

PQAP is a premium automated trading terminal for Polymarket prediction market arbitrage. The aesthetic is **dark, glassy, and electric** — inspired by Bloomberg Terminal's information density but with modern glassmorphism, gradient backgrounds, and micro-animations.

**Core philosophy:** "Disciplined Predator" — the UI conveys precision, speed, and control. Every pixel serves a purpose. Data is king. The interface feels alive with real-time updates but never chaotic.

**Visual personality:**
- **Dark foundation** — deep navy/charcoal backgrounds that make data pop
- **Glassmorphism** — frosted glass panels with subtle transparency and blur
- **Electric accents** — neon green for profit, coral red for loss, electric blue for interactive elements
- **Layered depth** — cards float above backgrounds with subtle shadows and borders
- **Micro-animations** — smooth transitions on hover, state changes, and data updates
- **Monospace data** — all numbers and financial data use JetBrains Mono for precision

**What PQAP is NOT:**
- Not flat or boring — gradients and glass effects everywhere
- Not playful or casual — this is a professional tool
- Not cluttered — information density is high but organized
- Not light-themed — dark mode only for v1

## Colors

### Backgrounds

| Token | Value | Usage |
|-------|-------|-------|
| `bg-primary` | `#0a0e17` | Main application background |
| `bg-secondary` | `#1a1f2e` | Sidebar, secondary panels |
| `bg-tertiary` | `#0f1320` | Elevated surfaces, modals |
| `bg-glass` | `rgba(255,255,255,0.05)` | Glass card backgrounds |
| `bg-glass-hover` | `rgba(255,255,255,0.08)` | Glass card hover state |
| `bg-glass-active` | `rgba(255,255,255,0.12)` | Glass card active/selected state |

### Semantic Colors

| Token | Value | Usage |
|-------|-------|-------|
| `profit` | `#00ff88` | Positive PnL, success states, profit values |
| `profit-muted` | `rgba(0,255,136,0.15)` | Profit backgrounds, subtle highlights |
| `loss` | `#ff4757` | Negative PnL, error states, loss values |
| `loss-muted` | `rgba(255,71,87,0.15)` | Loss backgrounds, subtle highlights |
| `accent` | `#00d4ff` | Interactive elements, links, focus states |
| `accent-muted` | `rgba(0,212,255,0.15)` | Accent backgrounds, badges |
| `warning` | `#ffa726` | Warning states, circuit breaker alerts |
| `warning-muted` | `rgba(255,167,38,0.15)` | Warning backgrounds |

### Text

| Token | Value | Usage |
|-------|-------|-------|
| `text-primary` | `#ffffff` | Headings, primary content, labels |
| `text-secondary` | `#8892a4` | Descriptions, secondary info, timestamps |
| `text-muted` | `#4a5568` | Disabled text, placeholders |

### Borders

| Token | Value | Usage |
|-------|-------|-------|
| `border` | `rgba(255,255,255,0.1)` | Card borders, dividers |
| `border-hover` | `rgba(255,255,255,0.2)` | Hover state borders |

### Gradients

| Token | Value | Usage |
|-------|-------|-------|
| `gradient-primary` | `linear-gradient(135deg, #0a0e17 0%, #1a1f2e 100%)` | Main background gradient |
| `gradient-accent` | `linear-gradient(135deg, #00d4ff 0%, #00ff88 100%)` | Accent elements, CTAs |
| `gradient-profit` | `linear-gradient(135deg, #00ff88 0%, #00d4ff 100%)` | Profit indicators |
| `gradient-loss` | `linear-gradient(135deg, #ff4757 0%, #ff6b81 100%)` | Loss indicators |

### Color Usage Rules

- **Profit/loss colors are sacred** — only use `profit`/`loss` for actual financial outcomes. Never for decorative elements.
- **Accent blue is interactive** — buttons, links, focus rings, clickable elements use `accent`.
- **Warning orange is cautionary** — circuit breakers, approaching limits, non-critical alerts.
- **Glass backgrounds layer** — `bg-glass` cards sit on `bg-primary` or gradient backgrounds. The blur creates depth.
- **Muted variants for backgrounds** — use `*-muted` tokens for subtle highlights behind text or badges.

## Typography

### Font Stack

| Token | Value | Usage |
|-------|-------|-------|
| `font-mono` | `JetBrains Mono, monospace` | All numbers, financial data, prices, PnL, code |
| `font-sans` | `Inter, sans-serif` | UI text, labels, descriptions, navigation |
| `font-display` | `Inter, sans-serif` | Headings, page titles (bold weight) |

### Type Scale

| Name | Size | Weight | Line Height | Usage |
|------|------|--------|-------------|-------|
| `display` | 32px | 700 | 1.2 | Page titles, hero numbers |
| `h1` | 24px | 700 | 1.3 | Section headings |
| `h2` | 20px | 600 | 1.4 | Card titles |
| `h3` | 16px | 600 | 1.4 | Subsection headings |
| `body` | 14px | 400 | 1.5 | Body text, descriptions |
| `body-sm` | 13px | 400 | 1.5 | Compact body text |
| `caption` | 12px | 400 | 1.5 | Captions, timestamps, metadata |
| `mono-lg` | 28px | 600 | 1.2 | Hero PnL numbers |
| `mono-md` | 18px | 500 | 1.3 | Table values, card numbers |
| `mono-sm` | 13px | 400 | 1.4 | Inline numbers, small data |
| `mono-xs` | 11px | 400 | 1.4 | Timestamps, micro data |

### Typography Rules

- **All financial data uses `font-mono`** — prices, PnL, percentages, quantities, scores.
- **UI chrome uses `font-sans`** — navigation, labels, descriptions, buttons.
- **Profit numbers are `profit` color** — always green for positive values.
- **Loss numbers are `loss` color** — always red for negative values.
- **Zero values are `text-secondary`** — neutral gray for zero PnL.
- **Monospace numbers are right-aligned** — in tables and cards for decimal alignment.

## Layout & Spacing

### Grid System

The dashboard uses a 12-column CSS Grid layout with responsive breakpoints.

| Breakpoint | Columns | Gutters | Margin |
|------------|---------|---------|--------|
| Desktop (≥1440px) | 12 | 24px | 32px |
| Laptop (≥1024px) | 12 | 16px | 24px |
| Tablet (≥768px) | 8 | 16px | 16px |

### Page Layout

```
┌─────────────────────────────────────────────────────────────┐
│                        Top Bar (64px)                        │
│  Logo · Page Title · Quick Actions · Notifications · Profile │
├────────┬────────────────────────────────────────────────────┤
│        │                                                    │
│  Side  │              Main Content Area                     │
│  bar   │              (fluid, scrollable)                   │
│ 280px  │                                                    │
│        │                                                    │
│        │                                                    │
│        │                                                    │
└────────┴────────────────────────────────────────────────────┘
```

### Sidebar

- **Expanded:** 280px width, shows icon + label + optional badge
- **Collapsed:** 72px width, shows icon only
- **Background:** `bg-secondary` with glass effect
- **Border:** Right border `border` token
- **Navigation items:** 48px height, 16px icon, 14px label
- **Active state:** `bg-glass-active` background, `accent` left border (3px)

### Content Area

- **Padding:** `card-padding` (24px) on all sides
- **Gap between cards:** `gap` (16px)
- **Max content width:** 1920px (centered on ultra-wide)
- **Scroll:** Custom styled scrollbar (thin, `bg-glass` track, `text-muted` thumb)

### Card Grid Patterns

| Pattern | Columns | Usage |
|---------|---------|-------|
| Stats Row | 4 equal columns | Portfolio overview cards |
| Split | 8/4 or 6/6 | Chart + table combinations |
| Full Width | 12 columns | Tables, feeds, full charts |
| Quad | 2x2 grid | Small metric cards |

## Elevation & Depth

### Glassmorphism Layer System

The UI uses a layered glass effect to create depth:

```
Layer 3: Modals, Popovers (bg-tertiary + blur(40px) + shadow)
Layer 2: Cards, Panels (bg-glass + blur(12px) + border + shadow)
Layer 1: Sidebar, Top Bar (bg-secondary + border)
Layer 0: Background (gradient-primary)
```

### Glass Card Recipe

```css
.glass-card {
  background: rgba(255, 255, 255, 0.05);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 16px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.3);
  transition: all 0.2s ease;
}

.glass-card:hover {
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.2);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}
```

### Shadow Tokens

| Token | Value | Usage |
|-------|-------|-------|
| `shadow-card` | `0 4px 24px rgba(0,0,0,0.3)` | Default card shadow |
| `shadow-card-hover` | `0 8px 32px rgba(0,0,0,0.4)` | Card hover shadow |
| `shadow-glow-profit` | `0 0 20px rgba(0,255,136,0.3)` | Profit highlight glow |
| `shadow-glow-loss` | `0 0 20px rgba(255,71,87,0.3)` | Loss highlight glow |
| `shadow-glow-accent` | `0 0 20px rgba(0,212,255,0.3)` | Accent highlight glow |

### Blur Tokens

| Token | Value | Usage |
|-------|-------|-------|
| `blur-glass` | `12px` | Standard glass cards |
| `blur-glass-heavy` | `20px` | Emphasized glass panels |
| `blur-backdrop` | `40px` | Modal overlays, dropdowns |

### Depth Rules

- **Cards always have borders** — no floating elements without borders in dark theme.
- **Hover elevates** — cards lift on hover with increased shadow and background opacity.
- **Active/selected states** — higher background opacity, brighter border.
- **Modals use maximum blur** — 40px backdrop blur over the entire page.
- **Glow effects are contextual** — profit glow on positive PnL cards, loss glow on negative.

## Shapes

### Border Radius

| Token | Value | Usage |
|-------|-------|-------|
| `rounded-card` | `16px` | Cards, panels, containers |
| `rounded-card-sm` | `12px` | Small cards, inner panels |
| `rounded-button` | `8px` | Buttons, inputs |
| `rounded-button-lg` | `12px` | Large buttons |
| `rounded-badge` | `20px` | Status badges, tags |
| `rounded-badge-sm` | `6px` | Small inline badges |
| `rounded-input` | `8px` | Form inputs, selects |
| `rounded-full` | `9999px` | Avatars, circular buttons |

### Gradient Borders

For premium elements, use gradient borders:

```css
.gradient-border {
  position: relative;
  border: none;
}

.gradient-border::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: 16px;
  padding: 1px;
  background: linear-gradient(135deg, rgba(0,212,255,0.3), rgba(0,255,136,0.3));
  mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
  mask-composite: exclude;
  pointer-events: none;
}
```

### Shape Rules

- **Consistent rounding** — cards use 16px, buttons use 8px, badges use 20px.
- **No sharp corners** — everything has at least 6px border radius.
- **Inner elements slightly smaller** — if card is 16px, inner panels are 12px.
- **Circular for avatars and status dots** — `rounded-full`.

## Components

### Buttons

| Variant | Background | Border | Text | Usage |
|---------|------------|--------|------|-------|
| Primary | `accent` | none | `white` | Primary actions (Start Trading, Save) |
| Secondary | `bg-glass` | `border` | `text-primary` | Secondary actions (Cancel, Close) |
| Danger | `loss` | none | `white` | Destructive actions (Emergency Stop, Delete) |
| Success | `profit` | none | `white` | Positive actions (Resume, Approve) |
| Ghost | transparent | none | `text-secondary` | Tertiary actions (View, More) |
| Icon | `bg-glass` | `border` | `text-secondary` | Icon-only actions (Settings, Refresh) |

**Button sizes:**
- **Large:** 48px height, 20px padding, 16px font, `rounded-button-lg`
- **Default:** 40px height, 16px padding, 14px font, `rounded-button`
- **Small:** 32px height, 12px padding, 13px font, `rounded-button`

**Button states:**
- **Hover:** Lighten background by 10%, add `shadow-glow-accent` for primary
- **Active:** Darken background by 10%
- **Disabled:** 50% opacity, `not-allowed` cursor
- **Loading:** Replace text with spinner, disable interaction

### Cards

**Standard Card:**
```
┌──────────────────────────────────────┐
│  Title                    [Action]   │
│  ─────────────────────────────────── │
│  Content area                        │
│  Numbers in JetBrains Mono           │
│                                      │
└──────────────────────────────────────┘
```

**Card anatomy:**
- **Header:** Title (h2) + optional action button/icon
- **Divider:** 1px `border` line
- **Content:** Variable height, scrollable if needed
- **Padding:** `card-padding` (24px) all sides
- **Background:** `bg-glass` + `blur-glass` + `shadow-card`

**Card variants:**
- **Stat Card:** Single large number with label and trend indicator
- **Table Card:** Wrapped table with header
- **Chart Card:** Chart with controls and legend
- **Feed Card:** Scrollable list with live updates

### Tables

**Table styling:**
- **Header:** `bg-glass` background, `text-secondary` text, 12px caption font, uppercase
- **Rows:** Transparent background, 1px `border` bottom divider
- **Hover:** `bg-glass-hover` background
- **Cells:** 14px body font, 16px padding
- **Numeric cells:** `font-mono`, right-aligned
- **PnL cells:** `profit` or `loss` color based on value

**Table features:**
- **Sortable columns:** Click header to sort, arrow indicator
- **Sticky header:** Stays visible on scroll
- **Virtual scrolling:** For large datasets (1000+ rows)
- **Row selection:** Checkbox for bulk actions

### Badges

| Variant | Background | Text | Usage |
|---------|------------|------|-------|
| Profit | `profit-muted` | `profit` | Positive PnL badge |
| Loss | `loss-muted` | `loss` | Negative PnL badge |
| Accent | `accent-muted` | `accent` | Status, info badges |
| Warning | `warning-muted` | `warning` | Warning badges |
| Neutral | `bg-glass` | `text-secondary` | Default badges |

**Badge sizes:**
- **Default:** 28px height, 12px padding, 12px font, `rounded-badge`
- **Small:** 22px height, 8px padding, 11px font, `rounded-badge-sm`

### Charts

**Chart styling (Recharts/Chart.js theme):**
- **Background:** Transparent (sits in glass card)
- **Grid lines:** `rgba(255,255,255,0.05)` — barely visible
- **Axis labels:** `text-secondary`, 11px mono
- **Tooltip:** Glass card style with `bg-tertiary` background
- **Line color:** `accent` for primary, `profit` for positive, `loss` for negative
- **Area fill:** Gradient from color (20% opacity) to transparent

**Chart types:**
- **PnL Line:** Area chart with gradient fill, accent color
- **Distribution:** Bar chart with profit/loss colors
- **Allocation:** Donut chart with accent/profit/loss palette
- **Depth:** Area chart showing bid/ask depth

### Inputs

**Input styling:**
- **Background:** `bg-glass`
- **Border:** `border` token, 1px
- **Border radius:** `rounded-input` (8px)
- **Text:** `text-primary`, 14px
- **Placeholder:** `text-muted`
- **Focus:** `accent` border, `shadow-glow-accent` outline
- **Error:** `loss` border, `loss` error text below
- **Height:** 40px default, 48px large, 32px small

### Status Indicators

| Status | Color | Shape | Usage |
|--------|-------|-------|-------|
| Online | `profit` | Circle (8px) | WebSocket connected, service healthy |
| Warning | `warning` | Circle (8px) | Approaching limits, degraded |
| Error | `loss` | Circle (8px) | Disconnected, circuit breaker open |
| Offline | `text-muted` | Circle (8px) | Service stopped, disabled |
| Loading | `accent` | Spinning circle | Fetching data, processing |

### Animations

| Animation | Duration | Easing | Usage |
|-----------|----------|--------|-------|
| Hover | 200ms | ease | Card hover, button hover |
| State change | 300ms | ease-in-out | PnL color change, status update |
| Data update | 150ms | ease-out | Number increment, value flash |
| Enter | 300ms | ease-out | Card mount, modal open |
| Exit | 200ms | ease-in | Card unmount, modal close |
| Pulse | 2s | ease-in-out | Live indicator, loading state |
| Slide | 250ms | ease-out | Sidebar toggle, panel slide |

**Micro-animation details:**
- **PnL flash:** When PnL updates, briefly flash the number with a 150ms glow effect
- **Card hover:** Scale 1.01, increase shadow, brighten border — 200ms
- **Status change:** Cross-fade between status colors — 300ms
- **Live dot:** Subtle pulse animation on connected indicators
- **Number roll:** Animate number changes with a brief roll effect

## Do's and Don'ts

### Do's

- **Do** use `font-mono` for ALL numbers, prices, percentages, and financial data
- **Do** use `profit`/`loss` colors consistently — green for positive, red for negative
- **Do** use glass effects on all cards and panels
- **Do** keep information density high but organized — this is a trading terminal
- **Do** use subtle animations to indicate live data and state changes
- **Do** align numbers right in tables for decimal alignment
- **Do** use gradient backgrounds for hero sections and empty states
- **Do** provide quick access to emergency stop — always visible in top bar
- **Do** use color-coded badges for status indicators
- **Do** keep the sidebar persistent across all pages

### Don'ts

- **Don't** use light backgrounds or light mode — dark theme only for v1
- **Don't** use generic green/red — use the specific `profit` (#00ff88) and `loss` (#ff4757) tokens
- **Don't** use `font-sans` for numbers — always `font-mono`
- **Don't** remove borders from glass cards — they need borders in dark theme
- **Don't** use heavy animations — keep transitions under 300ms
- **Don't** make the sidebar collapsible to zero — always show icons
- **Don't** use solid backgrounds for cards — always use glass effect
- **Don't** mix border radius values inconsistently — follow the token system
- **Don't** hide critical actions behind menus — emergency stop, pause, resume must be immediate
- **Don't** use placeholder data without indication — show skeleton loaders or "—" for missing data

## Design Token Reference (TailwindCSS)

```javascript
// tailwind.config.js extension
module.exports = {
  theme: {
    extend: {
      colors: {
        'bg-primary': '#0a0e17',
        'bg-secondary': '#1a1f2e',
        'bg-tertiary': '#0f1320',
        'glass': {
          DEFAULT: 'rgba(255,255,255,0.05)',
          hover: 'rgba(255,255,255,0.08)',
          active: 'rgba(255,255,255,0.12)',
        },
        'profit': {
          DEFAULT: '#00ff88',
          muted: 'rgba(0,255,136,0.15)',
        },
        'loss': {
          DEFAULT: '#ff4757',
          muted: 'rgba(255,71,87,0.15)',
        },
        'accent': {
          DEFAULT: '#00d4ff',
          muted: 'rgba(0,212,255,0.15)',
        },
        'warning': {
          DEFAULT: '#ffa726',
          muted: 'rgba(255,167,38,0.15)',
        },
        'text-primary': '#ffffff',
        'text-secondary': '#8892a4',
        'text-muted': '#4a5568',
      },
      fontFamily: {
        mono: ['JetBrains Mono', 'monospace'],
        sans: ['Inter', 'sans-serif'],
      },
      borderRadius: {
        card: '16px',
        'card-sm': '12px',
        button: '8px',
        'button-lg': '12px',
        badge: '20px',
        'badge-sm': '6px',
      },
      backdropBlur: {
        glass: '12px',
        'glass-heavy': '20px',
        backdrop: '40px',
      },
      boxShadow: {
        card: '0 4px 24px rgba(0,0,0,0.3)',
        'card-hover': '0 8px 32px rgba(0,0,0,0.4)',
        'glow-profit': '0 0 20px rgba(0,255,136,0.3)',
        'glow-loss': '0 0 20px rgba(255,71,87,0.3)',
        'glow-accent': '0 0 20px rgba(0,212,255,0.3)',
      },
    },
  },
}
```
