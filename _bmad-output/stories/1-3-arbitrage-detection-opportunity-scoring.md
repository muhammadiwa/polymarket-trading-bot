# Story 1.3: Arbitrage Detection & Opportunity Scoring

## Story

As a quant trader,
I want the bot to detect simple YES+NO arbitrage opportunities and score them,
So that only high-quality opportunities are passed to the execution engine.

## Status

ready-for-dev

## Acceptance Criteria

**Given** the arb engine is subscribed to `MarketPriceUpdated` events via NATS
**When** a price update shows YES_price + NO_price < $1.00 - min_profit_threshold
**Then** an opportunity is detected within 100ms of the price update
**And** the opportunity score is calculated as: spread × liquidity × fill_probability
**And** fill probability is estimated based on orderbook depth and historical fill rates
**And** opportunities with score below the configurable threshold (default: 0.01) are logged but not emitted to execution
**And** opportunities above threshold emit an `OpportunityDetected` event to NATS
**And** all detected opportunities (including filtered) are logged to TimescaleDB for backtesting analysis

## Technical Requirements

### Architecture Context

- **Service:** `arb-engine` (Go)
- **Paradigm:** Event-driven hexagonal architecture (ports & adapters)
- **Event bus:** NATS (JetStream for durable subscriptions)
- **Database:** TimescaleDB for opportunity logging (hypertable, 3-year retention per AD-7)
- **Pattern:** Arb Engine is a **pure function** of market state → scored opportunities (AD-2). It **never** executes. It **never** modifies market state. It logs every opportunity (including filtered ones) to TimescaleDB for backtesting.

### Key Components to Implement

1. **NATS Subscriber** (`internal/ports/event.go` + `adapters/nats_subscriber.go`)
   - Subscribe to `pqap.market.{market_id}.price` (MarketPriceUpdated events)
   - Parse market data from event payload
   - Handle connection lifecycle (reconnect on disconnect)
   - Filter out stale markets (ignore events where `IsStale = true`)

2. **Simple Arbitrage Detector** (`internal/detector/simple_arb.go`)
   - Calculate spread: `$1.00 - (YES_price + NO_price)`
   - If spread > `min_profit_threshold` (configurable, default: 0.01), flag as opportunity
   - Log detection latency (target: <100ms per NFR-A1)
   - Zero false negatives for profitable arb (NFR-A2)

3. **Opportunity Scoring Engine** (`internal/scorer/scorer.go`)
   - Calculate score: `spread × liquidity × fill_probability`
   - `spread` = the detected arbitrage spread (in dollars)
   - `liquidity` = sum of top-5 orderbook depth levels (YES side + NO side), normalized to 0-1
   - `fill_probability` = estimated from orderbook depth + historical fill rates (FR-13)
   - Score must be deterministic and reproducible (NFR-A3)

4. **Fill Probability Estimator** (`internal/scorer/fill_probability.go`)
   - Estimate fill probability based on:
     - Orderbook depth at current price level (deeper = higher probability)
     - Historical fill rates from TimescaleDB (opportunities table, last 30 days)
   - Fill probability estimate target: within 15% of actual fill rate over 100 trades (FR-13)
   - Initial implementation: use orderbook depth as primary signal; historical fill rates as calibration

5. **Threshold Filter** (`internal/filter/filter.go`)
   - Configurable threshold via `ARB_SCORE_THRESHOLD` env var (default: 0.01)
   - Opportunities below threshold: log to TimescaleDB with `filter_reason = "below_threshold"`
   - Opportunities above threshold: emit `OpportunityDetected` to NATS

6. **NATS Publisher** (`adapters/nats_publisher.go`)
   - Publish `OpportunityDetected` to subject `pqap.opportunity.detected`
   - Event schema: `event_id` (UUID), `event_type`, `timestamp` (ISO 8601 UTC), `source` ("arb-engine"), `payload`

7. **TimescaleDB Logger** (`internal/logger/opportunity_logger.go`)
   - Log ALL opportunities to TimescaleDB `opportunities` hypertable
   - Include: market_id, yes_price, no_price, spread, liquidity, fill_probability, score, filter_reason, timestamp
   - Queryable by date range for backtesting analysis (FR-16)

### Data Models

**Opportunity (internal domain model):**
```go
type Opportunity struct {
    ID             string          `json:"id"`              // UUID
    MarketID       string          `json:"market_id"`
    YESPrice       decimal.Decimal `json:"yes_price"`       // 4dp
    NOPrice        decimal.Decimal `json:"no_price"`        // 4dp
    Spread         decimal.Decimal `json:"spread"`          // $1.00 - (YES + NO)
    Liquidity      decimal.Decimal `json:"liquidity"`       // normalized 0-1
    FillProbability decimal.Decimal `json:"fill_probability"` // 0-1
    Score          decimal.Decimal `json:"score"`           // spread × liquidity × fill_probability
    FilterReason   string          `json:"filter_reason"`   // "" if emitted, "below_threshold" if filtered
    DetectedAt     time.Time       `json:"detected_at"`     // UTC TIMESTAMPTZ
    LatencyMs      int64           `json:"latency_ms"`      // detection latency in ms
}
```

**OpportunityDetected Event (from `shared/proto/events.go`):**
```go
type OpportunityDetected struct {
    EventID   string             `json:"event_id"`   // UUID
    EventType string             `json:"event_type"` // "OpportunityDetected"
    Timestamp time.Time          `json:"timestamp"`   // ISO 8601 UTC
    Source    string             `json:"source"`      // "arb-engine"
    Payload   OpportunityPayload `json:"payload"`
}

type OpportunityPayload struct {
    OpportunityID  string          `json:"opportunity_id"`
    MarketID       string          `json:"market_id"`
    YESPrice       decimal.Decimal `json:"yes_price"`
    NOPrice        decimal.Decimal `json:"no_price"`
    Spread         decimal.Decimal `json:"spread"`
    Score          decimal.Decimal `json:"score"`
    FillProbability decimal.Decimal `json:"fill_probability"`
    Liquidity      decimal.Decimal `json:"liquidity"`
}
```

### API Endpoints

None (event-driven service). No REST API endpoints.

### NATS Subject Hierarchy (from AD-9)

```
pqap.market.{market_id}.price     # Consumed: MarketPriceUpdated
pqap.market.stale                 # Consumed: MarketStaleDetected (to filter stale markets)
pqap.opportunity.detected         # Produced: OpportunityDetected
```

### Prometheus Metrics (AD-17)

```
pqap_arb_opportunities_detected_total    # Counter — total opportunities detected (before filter)
pqap_arb_opportunities_emitted_total     # Counter — opportunities emitted to NATS (above threshold)
pqap_arb_opportunities_filtered_total    # Counter — opportunities filtered (below threshold)
pqap_arb_detection_latency_ms            # Histogram — detection latency in ms (target: <100ms)
pqap_arb_score_distribution              # Histogram — opportunity score distribution
pqap_arb_fill_probability_estimate       # Histogram — fill probability estimates
pqap_arb_nats_connection_status          # Gauge — 1=connected, 0=disconnected
pqap_arb_stale_market_ignored_total      # Counter — stale market events ignored
```

## Implementation Guide

### Step 1: NATS Subscriber

- Subscribe to `pqap.market.>` (wildcard for all market price events)
- Parse `MarketPriceUpdated` event payload: market_id, yes_price, no_price, spread, volume, liquidity_depth, is_stale
- If `is_stale == true`, increment `pqap_arb_stale_market_ignored_total` and skip
- Handle NATS connection lifecycle: reconnect on disconnect, update `pqap_arb_nats_connection_status` gauge
- Use JetStream for durable subscription (at-least-once delivery, consumer deduplicates by event_id)

### Step 2: Arbitrage Detection

- On each `MarketPriceUpdated` event:
  1. Record `start_time = time.Now()`
  2. Calculate spread: `spread = 1.00 - (yes_price + no_price)`
  3. If `spread > min_profit_threshold` (configurable via `ARB_MIN_PROFIT_THRESHOLD`, default: 0.01):
     - Create `Opportunity` struct
     - Proceed to scoring
  4. If `spread <= min_profit_threshold`: no opportunity, skip
- After detection: `latency_ms = time.Since(start_time).Milliseconds()`
- Log if latency > 100ms (NFR-A1 warning)

### Step 3: Opportunity Scoring

- Calculate score: `spread × liquidity × fill_probability`
- `spread`: the detected spread value
- `liquidity`: sum of top-5 orderbook depth levels (YES + NO), normalized to 0-1 range
  - If orderbook depth unavailable, use volume as proxy (normalized)
  - Default: 0.5 if no liquidity data
- `fill_probability`: output of Fill Probability Estimator (Step 3a)
- Verify determinism: same inputs must produce same output (NFR-A3)

### Step 3a: Fill Probability Estimation

- Primary signal: orderbook depth at current price level
  - Deeper orderbook → higher fill probability
  - Formula: `min(depth_ratio, 1.0)` where `depth_ratio = available_depth / required_depth`
- Calibration: query TimescaleDB for historical fill rates on similar opportunities (last 30 days)
  - If insufficient history (<100 trades), use orderbook-only estimate
  - If sufficient history, blend: `0.7 × orderbook_estimate + 0.3 × historical_rate`
- Target: within 15% of actual fill rate over 100 trades (FR-13)

### Step 4: Threshold Filter

- Read threshold from `ARB_SCORE_THRESHOLD` env var (default: 0.01)
- If `score >= threshold`:
  1. Set `filter_reason = ""`
  2. Proceed to Step 5 (publish event)
  3. Increment `pqap_arb_opportunities_emitted_total`
- If `score < threshold`:
  1. Set `filter_reason = "below_threshold"`
  2. Skip event publishing
  3. Increment `pqap_arb_opportunities_filtered_total`
- Both cases: log to TimescaleDB (Step 6)

### Step 5: Event Publishing

- Create `OpportunityDetected` event with:
  - `event_id`: UUID v4
  - `event_type`: "OpportunityDetected"
  - `timestamp`: `time.Now().UTC()`
  - `source`: "arb-engine"
  - `payload`: OpportunityPayload (market_id, prices, spread, score, fill_probability, liquidity)
- Publish to NATS subject `pqap.opportunity.detected`
- Fire-and-forget (at-least-once delivery, consumers idempotent by event_id per AD-9)
- Increment `pqap_arb_opportunities_detected_total` (total, before filter)
- Record `pqap_arb_detection_latency_ms` histogram
- Record `pqap_arb_score_distribution` histogram

### Step 6: Opportunity Logging

- Log ALL opportunities (emitted and filtered) to TimescaleDB `opportunities` hypertable
- Table schema:
  ```sql
  CREATE TABLE opportunities (
      time         TIMESTAMPTZ NOT NULL,
      opportunity_id UUID NOT NULL,
      market_id    TEXT NOT NULL,
      yes_price    NUMERIC(10,4) NOT NULL,
      no_price     NUMERIC(10,4) NOT NULL,
      spread       NUMERIC(10,4) NOT NULL,
      liquidity    NUMERIC(10,8) NOT NULL,
      fill_probability NUMERIC(10,8) NOT NULL,
      score        NUMERIC(10,8) NOT NULL,
      filter_reason TEXT DEFAULT '',
      latency_ms   INTEGER NOT NULL,
      account_id   UUID DEFAULT NULL  -- for future multi-account (INF-18)
  );
  SELECT create_hypertable('opportunities', 'time');
  ```
- Retention: 3 years (AD-7)
- Queryable by date range for backtesting (FR-16)

## Testing

### Unit Tests

- **Spread calculation (`simple_arb_test.go`):**
  - YES=0.55, NO=0.40 → spread=0.05 (opportunity)
  - YES=0.55, NO=0.45 → spread=0.00 (no opportunity)
  - YES=0.60, NO=0.50 → spread=-0.10 (no opportunity, negative)
  - Edge case: YES=0.00, NO=0.00 → spread=1.00 (max)
  - Edge case: YES=1.00, NO=1.00 → spread=-1.00 (impossible but handle)

- **Score calculation (`scorer_test.go`):**
  - Deterministic: same inputs → same output (NFR-A3)
  - Score = spread × liquidity × fill_probability
  - Zero spread → zero score
  - Zero liquidity → zero score
  - Zero fill_probability → zero score
  - All positive → positive score

- **Fill probability estimation (`fill_probability_test.go`):**
  - Deep orderbook → high probability (near 1.0)
  - Shallow orderbook → low probability (near 0.0)
  - No orderbook data → default 0.5
  - Historical calibration blends correctly

- **Threshold filtering (`filter_test.go`):**
  - Score above threshold → emitted
  - Score below threshold → filtered
  - Score equal to threshold → emitted (>= not >)
  - Configurable threshold override via env var

- **Event publishing (`publisher_test.go`):**
  - Event schema correct (event_id, event_type, timestamp, source, payload)
  - Event_id is UUID
  - Timestamp is UTC
  - Source is "arb-engine"

### Integration Tests

- **MarketPriceUpdated → OpportunityDetected flow:**
  - Mock NATS server
  - Publish MarketPriceUpdated with arb opportunity
  - Verify OpportunityDetected received on `pqap.opportunity.detected`
  - Verify opportunity logged to TimescaleDB

- **Filtered opportunity flow:**
  - Publish MarketPriceUpdated with low-score opportunity
  - Verify NO OpportunityDetected received
  - Verify opportunity logged to TimescaleDB with `filter_reason = "below_threshold"`

- **Stale market ignored:**
  - Publish MarketPriceUpdated with `is_stale = true`
  - Verify NO opportunity detected
  - Verify `pqap_arb_stale_market_ignored_total` incremented

- **Detection latency:**
  - Publish 100 MarketPriceUpdated events
  - Verify p99 latency < 100ms (NFR-A1)

### Test Files

```
tests/unit/arb-engine/
├── simple_arb_test.go          # Spread calculation, edge cases
├── scorer_test.go              # Score calculation, determinism
├── fill_probability_test.go    # Fill probability estimation
├── filter_test.go              # Threshold filtering, configurable threshold
├── publisher_test.go           # Event schema, UUID, timestamp
└── opportunity_logger_test.go  # TimescaleDB logging

tests/integration/
├── arb_detection_flow_test.go     # MarketPriceUpdated → OpportunityDetected end-to-end
├── arb_filtered_flow_test.go      # Low-score opportunity → logged but not emitted
├── arb_stale_market_test.go       # Stale market → ignored
└── arb_latency_test.go            # Detection latency benchmark
```

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| Story 1.1 | — | Scanner WebSocket Connection & Market Catalog (prerequisite) |
| Story 1.2 | — | Market Scanner — Stale Detection, Reconnect & Batching (prerequisite) |
| `github.com/nats-io/nats.go` | latest | NATS event bus |
| `github.com/shopspring/decimal` | latest | Decimal precision (prices, scores) |
| `github.com/prometheus/client_golang` | latest | Metrics export |
| `go.uber.org/zap` | latest | Structured logging |
| `github.com/google/uuid` | latest | Event ID generation |
| `github.com/jackc/pgx/v5` | latest | PostgreSQL/TimescaleDB driver |

## External Dependencies (Infrastructure)

| Service | Required | Purpose |
|---------|----------|---------|
| NATS server | Yes | Event bus (consume MarketPriceUpdated, produce OpportunityDetected) |
| TimescaleDB | Yes | opportunities hypertable for opportunity logging |
| Polymarket API | No | Arb engine does not call Polymarket directly (AD-2) |

## Definition of Done

- [ ] YES+NO arbitrage detected when spread > threshold (FR-9)
- [ ] Detection latency < 100ms (NFR-A1)
- [ ] Zero false negatives for profitable arb (NFR-A2)
- [ ] Score calculation deterministic and reproducible (NFR-A3)
- [ ] Fill probability estimated from orderbook depth (FR-13)
- [ ] Opportunities filtered by configurable threshold (FR-12)
- [ ] `OpportunityDetected` event published to NATS (AD-9)
- [ ] All opportunities logged to TimescaleDB (FR-16)
- [ ] Prometheus metrics exported (AD-17)
- [ ] Structured JSON logging (INF-14)
- [ ] Unit tests pass with ≥80% coverage
- [ ] Integration tests pass
- [ ] Detection latency p99 < 100ms verified (NFR-A1)

## References

| Reference | Description |
|-----------|-------------|
| FR-9 | Engine SHALL detect simple YES+NO arbitrage when YES_price + NO_price < $1.00 - min_profit_threshold |
| FR-11 | Engine SHALL calculate opportunity score: spread × liquidity × fill_probability |
| FR-12 | Engine SHALL filter opportunities below configurable score threshold (default: 0.01) |
| FR-13 | Engine SHALL estimate fill probability based on orderbook depth and historical fill rates |
| FR-16 | Engine SHALL log all detected opportunities (including filtered ones) for backtesting analysis |
| AD-2 | Arb Engine is a pure function of market state → scored opportunities; it never executes and never modifies market state |
| AD-7 | TimescaleDB for time-series only: opportunities (3yr retention) |
| AD-9 | NATS event bus with defined subject hierarchy; fire-and-forget with at-least-once delivery |
| AD-17 | Prometheus metrics on /metrics for all services |
| NFR-A1 | Opportunity detection latency: within 100ms of price update |
| NFR-A2 | Zero false negatives for profitable YES+NO arbitrage |
| NFR-A3 | Score calculation determinism: reproducible |
| INF-11 | Decimal precision: all monetary values use Decimal (never float64); prices 4dp, quantities 8dp |
| INF-14 | Structured JSON logs with timestamp, level, service, request_id, message, context |
| INF-15 | Prometheus metric naming: pqap_{service}_{metric_name}_{unit} |
| INF-16 | Event naming: past tense verb + noun (OpportunityDetected) |
| INF-17 | All events include: event_id (UUID), event_type, timestamp (ISO 8601 UTC), source, payload |
| INF-18 | Include account_id as nullable column in all relevant tables from day one |

## Architecture Decisions Impacting This Story

| Decision | Impact |
|----------|--------|
| AD-2 (Service Boundary) | Arb Engine is pure function: market state → scored opportunities. Never executes, never modifies market state. |
| AD-7 (TimescaleDB) | Opportunities stored in hypertable with 3-year retention. Analytics queries run against TimescaleDB only. |
| AD-9 (Event Bus) | Consumes `pqap.market.*.price`, produces `pqap.opportunity.detected`. Fire-and-forget, at-least-once, idempotent consumers. |
| AD-10 (Sync vs Async) | Arb Engine → NATS is async (publish). No sync calls required. |
| AD-17 (Observability) | Prometheus metrics for detection count, latency, score distribution, fill probability. |
| INF-11 (Decimal) | All prices, spreads, scores use `decimal.Decimal` — never `float64`. |
| INF-18 (Multi-Account) | `account_id` nullable column in opportunities table from day one. |

## Directory Structure

```
services/arb-engine/
├── cmd/
│   └── main.go                    # Entry point — starts subscriber, detector, scorer
├── internal/
│   ├── detector/
│   │   └── simple_arb.go          # YES+NO arbitrage detection algorithm
│   ├── scorer/
│   │   ├── scorer.go              # Opportunity scoring: spread × liquidity × fill_probability
│   │   └── fill_probability.go    # Fill probability estimation (orderbook + historical)
│   ├── filter/
│   │   └── filter.go              # Threshold filtering (configurable, default: 0.01)
│   ├── logger/
│   │   └── opportunity_logger.go  # TimescaleDB opportunity logging (all opportunities)
│   └── ports/
│       ├── market_data.go         # MarketDataPort interface
│       └── event.go               # EventPort interface
├── adapters/
│   ├── nats_subscriber.go         # NATS subscriber (MarketPriceUpdated)
│   ├── nats_publisher.go          # NATS publisher (OpportunityDetected)
│   └── timescale_repo.go          # TimescaleDB adapter for opportunity logging
├── config/
│   └── config.go                  # Configuration (ARB_MIN_PROFIT_THRESHOLD, ARB_SCORE_THRESHOLD)
├── metrics/
│   └── metrics.go                 # Prometheus metrics (8 metrics)
└── go.mod
```

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `ARB_MIN_PROFIT_THRESHOLD` | `0.01` | Minimum spread to consider as opportunity (in dollars) |
| `ARB_SCORE_THRESHOLD` | `0.01` | Minimum score to emit to execution (below this → log only) |
| `ARB_FILL_PROB_WEIGHT_ORDERBOOK` | `0.7` | Weight for orderbook-based fill probability estimate |
| `ARB_FILL_PROB_WEIGHT_HISTORICAL` | `0.3` | Weight for historical fill rate calibration |
| `NATS_URL` | `nats://localhost:4222` | NATS server URL |
| `TIMESCALE_URL` | `postgres://localhost:5432/pqap` | TimescaleDB connection string |
