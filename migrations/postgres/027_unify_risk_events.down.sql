-- Drop indexes
DROP INDEX IF EXISTS idx_risk_events_decision;
DROP INDEX IF EXISTS idx_risk_events_account_id;
DROP INDEX IF EXISTS idx_risk_events_strategy_id;
DROP INDEX IF EXISTS idx_risk_events_market_id;
DROP INDEX IF EXISTS idx_risk_events_created_at;

-- Note: We don't drop the table or columns in down migration
-- as this would lose data. The columns added by this migration
-- (order_size, allowed, latency_ms, decision, trade_size, etc.)
-- are additive and safe to leave in place.
