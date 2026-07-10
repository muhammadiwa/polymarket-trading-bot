-- Drop indexes
DROP INDEX IF EXISTS idx_positions_created_at;
DROP INDEX IF EXISTS idx_positions_account_id;
DROP INDEX IF EXISTS idx_positions_strategy_id;
DROP INDEX IF EXISTS idx_positions_status;
DROP INDEX IF EXISTS idx_positions_market_id;

-- Note: We don't drop columns in down migration as this would lose data.
-- The columns added by this migration are additive and safe to leave in place.
