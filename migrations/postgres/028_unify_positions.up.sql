-- Unified positions table schema
-- Combines definitions from risk-manager and position-manager
-- This migration replaces ensureSchema() calls in both services

-- First, check if table exists
DO $$
BEGIN
    -- If table doesn't exist, create it with unified schema
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'positions') THEN
        CREATE TABLE positions (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            market_id TEXT NOT NULL,
            market_slug TEXT NOT NULL DEFAULT '',
            side TEXT NOT NULL,
            entry_price NUMERIC(18,8) NOT NULL,
            current_price NUMERIC(18,8) NOT NULL,
            quantity NUMERIC(18,8) NOT NULL,
            unrealized_pnl NUMERIC(18,8) NOT NULL DEFAULT 0,
            realized_pnl NUMERIC(18,8) NOT NULL DEFAULT 0,
            status TEXT NOT NULL DEFAULT 'open',
            strategy_id TEXT NOT NULL DEFAULT '',
            entry_order_id UUID DEFAULT NULL,
            exit_order_id UUID DEFAULT NULL,
            opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            closed_at TIMESTAMPTZ DEFAULT NULL,
            settled_at TIMESTAMPTZ DEFAULT NULL,
            account_id UUID DEFAULT NULL,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        );
    ELSE
        -- Table exists, add missing columns if they don't exist
        
        -- Add market_slug column (from position-manager)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'positions' AND column_name = 'market_slug') THEN
            ALTER TABLE positions ADD COLUMN market_slug TEXT NOT NULL DEFAULT '';
        END IF;
        
        -- Add unrealized_pnl column (from position-manager)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'positions' AND column_name = 'unrealized_pnl') THEN
            ALTER TABLE positions ADD COLUMN unrealized_pnl NUMERIC(18,8) NOT NULL DEFAULT 0;
        END IF;
        
        -- Add entry_order_id column (from position-manager)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'positions' AND column_name = 'entry_order_id') THEN
            ALTER TABLE positions ADD COLUMN entry_order_id UUID DEFAULT NULL;
        END IF;
        
        -- Add exit_order_id column (from position-manager)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'positions' AND column_name = 'exit_order_id') THEN
            ALTER TABLE positions ADD COLUMN exit_order_id UUID DEFAULT NULL;
        END IF;
        
        -- Add opened_at column (from position-manager)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'positions' AND column_name = 'opened_at') THEN
            ALTER TABLE positions ADD COLUMN opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
        END IF;
        
        -- Add closed_at column (from position-manager)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'positions' AND column_name = 'closed_at') THEN
            ALTER TABLE positions ADD COLUMN closed_at TIMESTAMPTZ DEFAULT NULL;
        END IF;
        
        -- Add settled_at column (from position-manager)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'positions' AND column_name = 'settled_at') THEN
            ALTER TABLE positions ADD COLUMN settled_at TIMESTAMPTZ DEFAULT NULL;
        END IF;
        
        -- Add account_id column (if missing)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'positions' AND column_name = 'account_id') THEN
            ALTER TABLE positions ADD COLUMN account_id UUID DEFAULT NULL;
        END IF;
    END IF;
END $$;

-- Normalize status values to lowercase (position-manager used 'OPEN', risk-manager used 'open')
UPDATE positions SET status = LOWER(status) WHERE status != LOWER(status);

-- Create indexes if they don't exist
CREATE INDEX IF NOT EXISTS idx_positions_market_id ON positions (market_id);
CREATE INDEX IF NOT EXISTS idx_positions_status ON positions (status);
CREATE INDEX IF NOT EXISTS idx_positions_strategy_id ON positions (strategy_id);
CREATE INDEX IF NOT EXISTS idx_positions_account_id ON positions (account_id);
CREATE INDEX IF NOT EXISTS idx_positions_created_at ON positions (created_at DESC);
