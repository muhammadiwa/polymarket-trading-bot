-- Unified risk_events table schema
-- Combines definitions from risk-manager and execution-engine
-- This migration replaces ensureSchema() calls in both services

-- First, check if table exists and has the old schema
DO $$
BEGIN
    -- If table doesn't exist, create it with unified schema
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'risk_events') THEN
        CREATE TABLE risk_events (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            decision TEXT NOT NULL,
            reason TEXT NOT NULL,
            market_id TEXT DEFAULT NULL,
            strategy_id TEXT DEFAULT NULL,
            trade_size NUMERIC(18,8) NOT NULL DEFAULT 0,
            order_size NUMERIC(18,8) NOT NULL DEFAULT 0,
            current_exposure NUMERIC(18,8) NOT NULL DEFAULT 0,
            limit_value NUMERIC(18,8) NOT NULL DEFAULT 0,
            daily_budget_remaining NUMERIC(18,8) NOT NULL DEFAULT 0,
            capital NUMERIC(18,8) NOT NULL DEFAULT 0,
            allowed BOOLEAN NOT NULL DEFAULT true,
            latency_ms INTEGER NOT NULL DEFAULT 0,
            context JSONB DEFAULT '{}',
            account_id UUID DEFAULT NULL,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        );
    ELSE
        -- Table exists, add missing columns if they don't exist
        
        -- Add order_size column (from execution-engine)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_events' AND column_name = 'order_size') THEN
            ALTER TABLE risk_events ADD COLUMN order_size NUMERIC(18,8) NOT NULL DEFAULT 0;
        END IF;
        
        -- Add allowed column (from execution-engine)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_events' AND column_name = 'allowed') THEN
            ALTER TABLE risk_events ADD COLUMN allowed BOOLEAN NOT NULL DEFAULT true;
        END IF;
        
        -- Add latency_ms column (from execution-engine)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_events' AND column_name = 'latency_ms') THEN
            ALTER TABLE risk_events ADD COLUMN latency_ms INTEGER NOT NULL DEFAULT 0;
        END IF;
        
        -- Add decision column (from risk-manager, if missing)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_events' AND column_name = 'decision') THEN
            ALTER TABLE risk_events ADD COLUMN decision TEXT NOT NULL DEFAULT 'ALLOW';
        END IF;
        
        -- Add trade_size column (from risk-manager, if missing)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_events' AND column_name = 'trade_size') THEN
            ALTER TABLE risk_events ADD COLUMN trade_size NUMERIC(18,8) NOT NULL DEFAULT 0;
        END IF;
        
        -- Add current_exposure column (from risk-manager, if missing)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_events' AND column_name = 'current_exposure') THEN
            ALTER TABLE risk_events ADD COLUMN current_exposure NUMERIC(18,8) NOT NULL DEFAULT 0;
        END IF;
        
        -- Add limit_value column (from risk-manager, if missing)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_events' AND column_name = 'limit_value') THEN
            ALTER TABLE risk_events ADD COLUMN limit_value NUMERIC(18,8) NOT NULL DEFAULT 0;
        END IF;
        
        -- Add daily_budget_remaining column (from risk-manager, if missing)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_events' AND column_name = 'daily_budget_remaining') THEN
            ALTER TABLE risk_events ADD COLUMN daily_budget_remaining NUMERIC(18,8) NOT NULL DEFAULT 0;
        END IF;
        
        -- Add capital column (from risk-manager, if missing)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_events' AND column_name = 'capital') THEN
            ALTER TABLE risk_events ADD COLUMN capital NUMERIC(18,8) NOT NULL DEFAULT 0;
        END IF;
        
        -- Add context column (from risk-manager, if missing)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_events' AND column_name = 'context') THEN
            ALTER TABLE risk_events ADD COLUMN context JSONB DEFAULT '{}';
        END IF;
        
        -- Add account_id column (from risk-manager, if missing)
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_events' AND column_name = 'account_id') THEN
            ALTER TABLE risk_events ADD COLUMN account_id UUID DEFAULT NULL;
        END IF;
    END IF;
END $$;

-- Create indexes if they don't exist
CREATE INDEX IF NOT EXISTS idx_risk_events_created_at ON risk_events (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_events_market_id ON risk_events (market_id);
CREATE INDEX IF NOT EXISTS idx_risk_events_strategy_id ON risk_events (strategy_id);
CREATE INDEX IF NOT EXISTS idx_risk_events_account_id ON risk_events (account_id);
CREATE INDEX IF NOT EXISTS idx_risk_events_decision ON risk_events (decision);
