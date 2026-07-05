CREATE TABLE strategies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'inactive' CHECK (status IN ('active', 'inactive', 'paused')),
    
    -- Strategy parameters
    min_profit_threshold DECIMAL(10,6) NOT NULL DEFAULT 0.01,
    score_threshold DECIMAL(10,6) NOT NULL DEFAULT 0.01,
    max_position_size DECIMAL(20,8) NOT NULL DEFAULT 1000.0,
    max_daily_trades INT NOT NULL DEFAULT 50,
    risk_limit_pct DECIMAL(5,2) NOT NULL DEFAULT 5.0,
    
    -- Capital allocation
    capital_weight DECIMAL(5,2) NOT NULL DEFAULT 100.0,
    
    -- Multi-account support (nullable) per INF-18
    account_id UUID,
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    activated_at TIMESTAMPTZ,
    
    CONSTRAINT valid_capital_weight CHECK (capital_weight >= 0 AND capital_weight <= 100),
    CONSTRAINT valid_risk_limit CHECK (risk_limit_pct > 0 AND risk_limit_pct <= 100),
    CONSTRAINT unique_strategy_name UNIQUE (name)
);

CREATE INDEX idx_strategies_status ON strategies(status);
CREATE INDEX idx_strategies_account ON strategies(account_id);
