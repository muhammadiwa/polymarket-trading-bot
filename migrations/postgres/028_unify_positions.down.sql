-- Down migration for positions unification
-- NOTE: This migration does NOT drop columns because:
-- 1. Dropping columns would lose data
-- 2. The added columns (market_slug, unrealized_pnl, entry_order_id, etc.) are additive
-- 3. Removing them would break services that now depend on them
--
-- If you need to fully revert, you must:
-- 1. Backup all data from positions table
-- 2. Drop and recreate the table with original schema
-- 3. Restore data (excluding new columns)
--
-- This is intentionally a no-op to preserve data safety.

-- Drop indexes only
DROP INDEX IF EXISTS idx_positions_created_at;
DROP INDEX IF EXISTS idx_positions_account_id;
DROP INDEX IF EXISTS idx_positions_strategy_id;
DROP INDEX IF EXISTS idx_positions_status;
DROP INDEX IF EXISTS idx_positions_market_id;
