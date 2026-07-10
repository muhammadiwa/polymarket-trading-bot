-- Add account_id column to tables for multi-account support
-- Uses IF NOT EXISTS to handle cases where columns already exist

-- trades table: account_id may already exist from migration 002
-- Only add FK constraint if column exists but FK doesn't
DO $$
BEGIN
    -- Add column if not exists
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'trades' AND column_name = 'account_id') THEN
        ALTER TABLE trades ADD COLUMN account_id UUID REFERENCES accounts(id);
    END IF;

    -- Add FK constraint if not exists (column might exist without FK)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'trades_account_id_fkey' AND table_name = 'trades'
    ) THEN
        -- Only add FK if accounts table exists
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'accounts') THEN
            ALTER TABLE trades ADD CONSTRAINT trades_account_id_fkey FOREIGN KEY (account_id) REFERENCES accounts(id);
        END IF;
    END IF;
END $$;

-- positions table: created by Go services, add column if table exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'positions') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'positions' AND column_name = 'account_id') THEN
            ALTER TABLE positions ADD COLUMN account_id UUID REFERENCES accounts(id);
        END IF;
    END IF;
END $$;

-- risk_events table: created by Go services, add column if table exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'risk_events') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_events' AND column_name = 'account_id') THEN
            ALTER TABLE risk_events ADD COLUMN account_id UUID REFERENCES accounts(id);
        END IF;
    END IF;
END $$;

-- Create indexes only if they don't exist
CREATE INDEX IF NOT EXISTS idx_trades_account_id ON trades(account_id);

-- Only create indexes if tables exist
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'positions') THEN
        CREATE INDEX IF NOT EXISTS idx_positions_account_id ON positions(account_id);
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'risk_events') THEN
        CREATE INDEX IF NOT EXISTS idx_risk_events_account_id ON risk_events(account_id);
    END IF;
END $$;
