-- Remove account_id column from tables
-- Drop constraints first, then columns

-- Drop FK constraints if they exist
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'trades_account_id_fkey') THEN
        ALTER TABLE trades DROP CONSTRAINT trades_account_id_fkey;
    END IF;
END $$;

-- Drop indexes
DROP INDEX IF EXISTS idx_trades_account_id;
DROP INDEX IF EXISTS idx_positions_account_id;
DROP INDEX IF EXISTS idx_risk_events_account_id;

-- Drop columns
ALTER TABLE trades DROP COLUMN IF EXISTS account_id;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'positions') THEN
        ALTER TABLE positions DROP COLUMN IF EXISTS account_id;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'risk_events') THEN
        ALTER TABLE risk_events DROP COLUMN IF EXISTS account_id;
    END IF;
END $$;
