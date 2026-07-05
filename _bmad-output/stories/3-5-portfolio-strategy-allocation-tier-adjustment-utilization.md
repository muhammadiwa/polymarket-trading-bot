# Story 3.5: Portfolio — Strategy Allocation, Tier Adjustment & Utilization

Status: ready-for-dev

## Story

As a quant trader,
I want the portfolio manager to auto-adjust position limits based on capital tier and track capital utilization,
So that my risk exposure scales appropriately as my capital grows.

## Acceptance Criteria

1. **Given** the portfolio manager tracks total capital (deposits + realized PnL + unrealized PnL)
   **When** capital crosses a tier threshold
   **Then** position limits are auto-adjusted based on the capital tier
   **And** tier promotion requires capital above threshold for 7 consecutive days
   **And** demotion is immediate on capital drop
   **And** tier transitions are logged

2. **Given** capital is deployed across positions
   **When** capital utilization rate is queried
   **Then** utilization is calculated as % of capital deployed, accurate within 1%
   **And** the rate is available via API and dashboard

3. **Given** the user wants to rebalance strategy weights
   **When** a manual rebalance is initiated
   **Then** the rebalance executes within 10 seconds
   **And** the rebalance action is logged

## Tasks / Subtasks

- [ ] Task 1: Capital Tier System (AC: #1)
  - [ ] Define tier thresholds and position limits
  - [ ] Track consecutive days above threshold for promotion
  - [ ] Implement immediate demotion on capital drop
  - [ ] Log tier transitions to PostgreSQL
- [ ] Task 2: Capital Utilization Tracking (AC: #2)
  - [ ] Calculate utilization = deployed / total capital
  - [ ] Expose via API endpoint
  - [ ] Update dashboard component
- [ ] Task 3: Manual Rebalance (AC: #3)
  - [ ] Implement POST /api/portfolio/rebalance endpoint
  - [ ] Execute rebalance within 10 seconds
  - [ ] Log rebalance action
- [ ] Task 4: Testing
  - [ ] Unit tests for tier promotion/demotion
  - [ ] Unit tests for utilization calculation
  - [ ] Unit tests for rebalance execution

## Dev Notes

### Architecture Context

- **Service:** New `portfolio-manager` (Python/FastAPI)
- **Database:** PostgreSQL for tier tracking, utilization history
- **Event bus:** NATS for TierChanged, PortfolioRebalanced events
- **Pattern:** Capital tier derived from total capital; promotion requires 7-day sustained threshold

### Key Architecture Rules

- **AD-16:** Capital tier derived from total capital; tier determines active strategies, max position size %, risk budget; promotion requires 7 consecutive days above threshold; demotion immediate
- **FR-32:** Capital allocation across strategies based on configurable weights
- **FR-35:** Auto-adjust position limits based on capital tier
- **FR-36:** Calculate capital utilization rate
- **FR-37:** Support manual capital rebalancing
- **NFR-PM1:** Capital calculation accuracy $0.01
- **NFR-PM2:** Allocation sum consistency always 100%

### Capital Tier Definitions

| Tier | Capital Range | Max Position % | Max Daily Trades | Strategy Limit |
|------|--------------|----------------|------------------|----------------|
| 1 | $0 – $100 | 20% | 10 | 2 |
| 2 | $100 – $1,000 | 15% | 25 | 3 |
| 3 | $1,000 – $10,000 | 10% | 50 | 5 |
| 4 | $10,000+ | 5% | 100 | 10 |

### Database Schema

```sql
-- Migration: migrations/postgres/013_create_portfolio_tiers.up.sql
CREATE TABLE portfolio_tiers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID,
    current_tier INT NOT NULL DEFAULT 1,
    total_capital DECIMAL(20,8) NOT NULL DEFAULT 0,
    deployed_capital DECIMAL(20,8) NOT NULL DEFAULT 0,
    utilization_rate DECIMAL(5,4) NOT NULL DEFAULT 0,
    days_above_threshold INT NOT NULL DEFAULT 0,
    promotion_threshold DECIMAL(20,8),
    promoted_at TIMESTAMPTZ,
    demoted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tier_transitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID,
    from_tier INT NOT NULL,
    to_tier INT NOT NULL,
    capital_at_transition DECIMAL(20,8) NOT NULL,
    reason VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE rebalance_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID,
    old_weights JSONB NOT NULL,
    new_weights JSONB NOT NULL,
    initiated_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Migration DOWN: migrations/postgres/013_create_portfolio_tiers.down.sql
DROP TABLE IF EXISTS rebalance_log;
DROP TABLE IF EXISTS tier_transitions;
DROP TABLE IF EXISTS portfolio_tiers;
```

### Prometheus Metrics

```
pqap_portfolio_tier_current              # Gauge — current tier
pqap_portfolio_utilization_rate          # Gauge — capital utilization %
pqap_portfolio_tier_transitions_total    # Counter — tier transitions
pqap_portfolio_rebalances_total          # Counter — rebalances executed
pqap_portfolio_total_capital             # Gauge — total capital
pqap_portfolio_deployed_capital          # Gauge — deployed capital
```

### References

| Reference | Description |
|-----------|-------------|
| FR-32 | Capital allocation across strategies |
| FR-35 | Auto-adjust position limits based on tier |
| FR-36 | Calculate capital utilization rate |
| FR-37 | Manual capital rebalancing |
| AD-16 | Capital tier system with promotion/demotion rules |
| NFR-PM1 | Capital calculation accuracy $0.01 |

## Dev Agent Record

### Agent Model Used

mimo-v2.5-pro

### Debug Log References

- Created new portfolio-manager service
- Tier system with promotion (7-day) and immediate demotion

### Completion Notes List

- Task 1: Capital tier system with 4 tiers, promotion/demotion logic
- Task 2: Utilization tracking with API endpoint
- Task 3: Rebalance endpoint with logging
- Task 4: Portfolio transitions history endpoint

### File List

**New files:**
- `migrations/postgres/013_create_portfolio_tiers.up/down.sql`
- `services/portfolio-manager/app/main.py`
- `services/portfolio-manager/app/config.py`
- `services/portfolio-manager/app/db.py`
- `services/portfolio-manager/app/middleware/auth.py`
- `services/portfolio-manager/app/models/portfolio.py`
- `services/portfolio-manager/app/repos/portfolio_repo.py`
- `services/portfolio-manager/app/routes/portfolio.py`
- `services/portfolio-manager/requirements.txt`
- `services/portfolio-manager/Dockerfile`
