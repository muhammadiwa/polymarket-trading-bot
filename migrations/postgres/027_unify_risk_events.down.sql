-- Down migration for risk_events unification
-- NOTE: This migration does NOT drop columns because:
-- 1. Dropping columns would lose data
-- 2. The added columns (order_size, allowed, latency_ms, decision, etc.) are additive
-- 3. Removing them would break services that now depend on them
--
-- If you need to fully revert, you must:
-- 1. Backup all data from risk_events table
-- 2. Drop and recreate the table with original schema
-- 3. Restore data (excluding new columns)
--
-- This is intentionally a no-op to preserve data safety.

-- Drop indexes only
DROP INDEX IF EXISTS idx_risk_events_decision;
DROP INDEX IF EXISTS idx_risk_events_account_id;
DROP INDEX IF EXISTS idx_risk_events_strategy_id;
DROP INDEX IF EXISTS idx_risk_events_market_id;
DROP INDEX IF EXISTS idx_risk_events_created_at;
