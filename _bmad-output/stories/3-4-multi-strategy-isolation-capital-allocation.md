# Story 3.4: Multi-Strategy — Isolation & Capital Allocation

Status: ready-for-dev

## Story

As a quant trader,
I want multiple strategies to run simultaneously with full isolation,
So that a failure in one strategy doesn't affect the others.

## Acceptance Criteria

1. **Given** multiple strategies are active
   **When** one strategy encounters a panic or error
   **Then** the failed strategy is logged and deactivated
   **And** other strategies continue operating normally without interruption
   **And** panic recovery catches the error without crashing the service

2. **Given** a strategy attempts to place a trade
   **When** the Portfolio Manager checks per-strategy capital allocation
   **Then** orders are rejected if the strategy would exceed its allocation
   **And** capital allocation is tracked accurately per strategy

3. **Given** multiple strategies have closed positions
   **When** portfolio-level metrics are calculated
   **Then** metrics are aggregated correctly from all strategies
   **And** there is no double-counting of positions or PnL
   **And** strategy-level metrics are consistent with portfolio-level metrics

## Tasks / Subtasks

- [ ] Task 1: Strategy Isolation in Execution Engine (AC: #1)
  - [ ] Add panic recovery per strategy goroutine
  - [ ] Auto-deactivate failed strategy via NATS event
  - [ ] Log failure with full context
  - [ ] Other strategies continue unaffected
- [ ] Task 2: Per-Strategy Capital Enforcement (AC: #2)
  - [ ] Extend Pit Boss to check per-strategy capital allocation
  - [ ] Reject orders that would exceed strategy's weight-based allocation
  - [ ] Track capital usage per strategy in Redis
- [ ] Task 3: Portfolio Metric Aggregation (AC: #3)
  - [ ] Aggregate PnL across all strategies (no double-counting)
  - [ ] Aggregate positions across all strategies
  - [ ] Strategy-level metrics consistent with portfolio-level
- [ ] Task 4: Testing
  - [ ] Unit tests for panic recovery and auto-deactivation
  - [ ] Unit tests for capital allocation enforcement
  - [ ] Unit tests for metric aggregation

## Dev Notes

### Architecture Context

- **Services:** execution-engine (Go), portfolio-manager (Python), risk-manager (Go)
- **Pattern:** Each strategy runs as isolated goroutine group (AD-13)
- **State:** Redis for per-strategy capital tracking, PostgreSQL for persistence
- **Event bus:** NATS for StrategyDeactivated events

### Key Architecture Rules

- **AD-13:** Each strategy as logical goroutine group; separate capital allocation, risk limits, position tracking, performance metrics; panic recovery without service crash
- **FR-104:** Support running multiple strategies simultaneously
- **FR-105:** Isolate strategy failures (one strategy crash doesn't affect others)
- **FR-106:** Enforce per-strategy capital allocation
- **FR-107:** Aggregate strategy performance for portfolio-level metrics
- **NFR-MS1:** Failure isolation — no cascade to other strategies
- **NFR-MS2:** Resource budget — configurable CPU/memory per strategy
- **NFR-MS3:** Portfolio metric consistency

### Files to MODIFY

**`services/execution-engine/internal/strategy/runner.go`** (NEW)
- Strategy runner with panic recovery
- Per-strategy goroutine isolation
- Auto-deactivate on failure

**`services/risk-manager/internal/pitboss/pitboss.go`**
- Extend `evaluate()` to check per-strategy capital allocation
- Read strategy weights from Redis
- Reject if trade would exceed allocation

**`services/risk-manager/internal/pitboss/redis_writer.go`**
- Add per-strategy capital tracking to Pit Boss state
- Write strategy allocation info to Redis

**`services/portfolio-manager/`** (if exists)
- Aggregate PnL across strategies
- Aggregate positions across strategies

### Strategy Runner Pattern

```go
type StrategyRunner struct {
    strategyID string
    config     StrategyConfig
    cancel     context.CancelFunc
    mu         sync.Mutex
    status     string // "running", "failed", "stopped"
}

func (sr *StrategyRunner) Run(ctx context.Context) {
    defer func() {
        if r := recover(); r != nil {
            sr.status = "failed"
            log.Error("strategy panic recovered", zap.String("strategy_id", sr.strategyID), zap.Any("panic", r))
            // Publish StrategyDeactivated event
            // Auto-deactivate in DB
        }
    }()
    // Strategy execution loop
}
```

### Capital Allocation Check

```go
func (pb *PitBoss) checkStrategyAllocation(strategyID string, tradeSize decimal.Decimal) bool {
    // Get strategy weight from Redis
    weight := pb.getStrategyWeight(strategyID)
    // Get total capital
    totalCapital := pb.getCapital()
    // Calculate max allocation
    maxAllocation := totalCapital.Mul(weight).Div(decimal.NewFromInt(100))
    // Get current usage
    currentUsage := pb.getStrategyUsage(strategyID)
    // Check if trade would exceed
    return currentUsage.Add(tradeSize).LessThanOrEqual(maxAllocation)
}
```

### Prometheus Metrics

```
pqap_strategy_isolation_failures_total    # Counter — strategy failures caught by panic recovery
pqap_strategy_capital_rejections_total    # Counter — trades rejected due to capital allocation
pqap_portfolio_strategy_pnl               # Gauge — PnL per strategy (label: strategy_id)
pqap_portfolio_total_pnl                  # Gauge — total PnL across all strategies
```

### Testing Standards

- Unit tests for panic recovery (simulate panic, verify recovery)
- Unit tests for capital allocation enforcement (exceed limit → reject)
- Unit tests for metric aggregation (verify no double-counting)
- Integration tests with multiple active strategies

### References

| Reference | Description |
|-----------|-------------|
| FR-104 | Support running multiple strategies simultaneously |
| FR-105 | Isolate strategy failures (one strategy crash doesn't affect others) |
| FR-106 | Enforce per-strategy capital allocation |
| FR-107 | Aggregate strategy performance for portfolio-level metrics |
| AD-13 | Each strategy as logical goroutine group with panic recovery |
| NFR-MS1 | Failure isolation — no cascade |
| NFR-MS2 | Resource budget per strategy |
| NFR-MS3 | Portfolio metric consistency |

## Dev Agent Record

### Agent Model Used

mimo-v2.5-pro

### Debug Log References

- Strategy runner uses defer/recover pattern for panic isolation
- PitBoss capital enforcement integrated into existing evaluate flow

### Completion Notes List

- Task 1: Strategy runner with panic recovery, retry logic, and Manager for multi-strategy
- Task 2: PitBoss extended with per-strategy capital allocation check
- Task 3: StrategyWeights added to PitBossState for portfolio visibility

### File List

**New files:**
- `services/execution-engine/internal/strategy/runner.go`

**Modified files:**
- `services/execution-engine/metrics/metrics.go` — added strategy isolation metrics
- `services/risk-manager/internal/ports/risk_state.go` — added StrategyWeights to PitBossState
- `services/risk-manager/internal/pitboss/pitboss.go` — added per-strategy capital check
- `services/risk-manager/internal/pitboss/redis_writer.go` — added strategy weights to state
