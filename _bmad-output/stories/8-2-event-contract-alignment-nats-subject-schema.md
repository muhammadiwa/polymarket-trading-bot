# Story 8.2: Event Contract Alignment — Fix NATS Subject & Schema Mismatches

Status: ready-for-dev

## Story

As a developer,
I want all NATS event schemas and subjects aligned between publishers and subscribers,
so that there is no silent data loss and all event flows work correctly end-to-end.

## Acceptance Criteria

- [ ] All NATS subjects have at least one consumer
- [ ] Event schemas match between publishers and subscribers
- [ ] Exit order requests are processed by execution-engine
- [ ] Cancel-all-orders requests are processed by execution-engine
- [ ] CapitalUpdated events published by portfolio-manager

## Tasks / Subtasks

- [ ] Task 1: ExitOrderRequest Handler (AC: 3)
  - [ ] Subtask 1.1: Add ExitOrderRequest type to execution-engine ports
  - [ ] Subtask 1.2: Add subscription handler in execution-engine
  - [ ] Subtask 1.3: Implement exit order logic
- [ ] Task 2: CancelAllOrders Handler (AC: 4)
  - [ ] Subtask 2.1: Add CancelAllOrders type to execution-engine ports
  - [ ] Subtask 2.2: Add subscription handler in execution-engine
  - [ ] Subtask 2.3: Implement cancel-all logic
- [ ] Task 3: CapitalUpdated Publisher (AC: 5)
  - [ ] Subtask 3.1: Add CapitalUpdated type to portfolio-manager
  - [ ] Subtask 3.2: Publish event when capital changes
- [ ] Task 4: PositionUpdated Schema Alignment (AC: 2)
  - [ ] Subtask 4.1: Align schema between position-manager and risk-manager
- [ ] Task 5: OrderFilled Schema Alignment (AC: 2)
  - [ ] Subtask 5.1: Add market_slug to risk-manager OrderFilledPayload

## Issues to Fix

| Issue | Publisher | Subscriber | Impact |
|-------|-----------|------------|--------|
| ExitOrderRequest has no consumer | position-manager | (none) | Position exits never execute |
| CancelAllOrders has no consumer | risk-manager | (none) | Emergency stop can't cancel orders |
| CapitalUpdated has no publisher | (none) | risk-manager | Capital tracking non-functional |
| PositionUpdated schema mismatch | position-manager | risk-manager | Silent data loss |
| OrderFilled schema mismatch | execution-engine | risk-manager | market_slug dropped |

## Dev Notes

### Architecture Context

- **Event Bus:** NATS 2.10+ (JetStream) — INF-8
- **Pattern:** Fire-and-forget with at-least-once delivery — AD-9
- **Idempotency:** All consumers deduplicate by event_id — AD-9

### NATS Subject Hierarchy

```
pqap.opportunity.detected      # Arb Engine → Execution Engine
pqap.order.placed               # Execution Engine → subscribers
pqap.order.filled               # Execution Engine → subscribers
pqap.order.cancelled            # Execution Engine → subscribers
pqap.order.failed               # Execution Engine → subscribers
pqap.order.exit_request         # Position Manager → Execution Engine (NEW)
pqap.order.cancel_all           # Risk Manager → Execution Engine (NEW)
pqap.position.opened            # Position Manager → subscribers
pqap.position.closed            # Position Manager → subscribers
pqap.position.updated           # Position Manager → Risk Manager
pqap.portfolio.capital_updated  # Portfolio Manager → Risk Manager (NEW)
pqap.risk.alert                 # Risk Manager → subscribers
pqap.risk.emergency             # Risk Manager → subscribers
pqap.notification.request       # Any service → Notification
```

### ExitOrderRequest Event Schema

```go
// From position-manager/internal/ports/event.go
type ExitOrderRequest struct {
    EventID   string                `json:"event_id"`
    EventType string                `json:"event_type"`
    Timestamp time.Time             `json:"timestamp"`
    Source    string                `json:"source"`
    Payload   ExitOrderRequestPayload `json:"payload"`
}

type ExitOrderRequestPayload struct {
    PositionID string          `json:"position_id"`
    MarketID   string          `json:"market_id"`
    Side       string          `json:"side"`
    Quantity   decimal.Decimal `json:"quantity"`
    OrderType  string          `json:"order_type"`
    Reason     string          `json:"reason"`
}
```

### CancelAllOrders Event Schema

```go
// From risk-manager/internal/ports/event.go
type CancelAllOrders struct {
    EventID   string                 `json:"event_id"`
    EventType string                 `json:"event_type"`
    Timestamp time.Time              `json:"timestamp"`
    Source    string                 `json:"source"`
    Payload   CancelAllOrdersPayload `json:"payload"`
}

type CancelAllOrdersPayload struct {
    Reason   string `json:"reason"`
    UserID   string `json:"user_id"`
}
```

### CapitalUpdated Event Schema

```go
// New type for portfolio-manager
type CapitalUpdated struct {
    EventID   string                `json:"event_id"`
    EventType string                `json:"event_type"`
    Timestamp time.Time             `json:"timestamp"`
    Source    string                `json:"source"`
    Payload   CapitalUpdatedPayload `json:"payload"`
}

type CapitalUpdatedPayload struct {
    TotalCapital    decimal.Decimal `json:"total_capital"`
    DailyPnL        decimal.Decimal `json:"daily_pnl"`
    UnrealizedPnL   decimal.Decimal `json:"unrealized_pnl"`
    CapitalTier     string          `json:"capital_tier"`
}
```

### PositionUpdated Schema Fix

**Current (position-manager publishes):**
```go
type PositionUpdatedPayload struct {
    PositionID    string          `json:"position_id"`
    MarketID      string          `json:"market_id"`
    CurrentPrice  decimal.Decimal `json:"current_price"`
    UnrealizedPnL decimal.Decimal `json:"unrealized_pnl"`
    UpdatedAt     time.Time       `json:"updated_at"`
}
```

**Current (risk-manager expects):**
```go
type PositionUpdatedPayload struct {
    PositionID   string          `json:"position_id"`
    MarketID     string          `json:"market_id"`
    CurrentPrice decimal.Decimal `json:"current_price"`
    Quantity     decimal.Decimal `json:"quantity"`
    Side         string          `json:"side"`
}
```

**Fix:** Update risk-manager to accept the position-manager's schema.

### OrderFilled Schema Fix

**Current (execution-engine publishes):**
```go
type OrderFilledPayload struct {
    OrderID       string          `json:"order_id"`
    ClientOrderID string          `json:"client_order_id"`
    OpportunityID string          `json:"opportunity_id"`
    MarketID      string          `json:"market_id"`
    MarketSlug    string          `json:"market_slug"`
    Side          string          `json:"side"`
    Price         decimal.Decimal `json:"price"`
    FilledQty     decimal.Decimal `json:"filled_qty"`
    LatencyMs     int64           `json:"latency_ms"`
    StrategyID    string          `json:"strategy_id"`
}
```

**Current (risk-manager expects):**
```go
type OrderFilledPayload struct {
    OrderID       string          `json:"order_id"`
    ClientOrderID string          `json:"client_order_id"`
    MarketID      string          `json:"market_id"`
    Side          string          `json:"side"`
    Price         decimal.Decimal `json:"price"`
    FilledQty     decimal.Decimal `json:"filled_qty"`
    StrategyID    string          `json:"strategy_id"`
}
```

**Fix:** Add missing fields to risk-manager's OrderFilledPayload.

## Implementation Guide

### Step 1: Add ExitOrderRequest to execution-engine

- Add event type to `services/execution-engine/internal/ports/event.go`
- Add subscription in `services/execution-engine/adapters/nats_subscriber.go`
- Implement exit order handler

### Step 2: Add CancelAllOrders to execution-engine

- Add event type to `services/execution-engine/internal/ports/event.go`
- Add subscription in `services/execution-engine/adapters/nats_subscriber.go`
- Implement cancel-all handler

### Step 3: Add CapitalUpdated to portfolio-manager

- Add event type to `services/portfolio-manager/app/models/events.py`
- Publish event when capital changes

### Step 4: Align PositionUpdated schema

- Update risk-manager to accept position-manager's schema

### Step 5: Align OrderFilled schema

- Add missing fields to risk-manager's OrderFilledPayload

## Files to Modify

| File | Changes |
|------|---------|
| `services/execution-engine/internal/ports/event.go` | Add ExitOrderRequest, CancelAllOrders types |
| `services/execution-engine/adapters/nats_subscriber.go` | Add subscriptions |
| `services/execution-engine/adapters/nats_publisher.go` | Add publish methods |
| `services/risk-manager/internal/ports/event.go` | Fix PositionUpdated, OrderFilled schemas |
| `services/portfolio-manager/app/models/events.py` | Add CapitalUpdated type |
| `services/portfolio-manager/app/main.py` | Publish CapitalUpdated |

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
