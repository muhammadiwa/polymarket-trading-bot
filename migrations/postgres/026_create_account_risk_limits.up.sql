-- Per-account risk limits table
CREATE TABLE IF NOT EXISTS account_risk_limits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    daily_loss_limit DECIMAL(10, 4) NOT NULL DEFAULT 2.0,
    max_position_per_market DECIMAL(10, 4) NOT NULL DEFAULT 10.0,
    max_position_per_strategy DECIMAL(10, 4) NOT NULL DEFAULT 20.0,
    drawdown_threshold DECIMAL(10, 4) NOT NULL DEFAULT 10.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(account_id)
);

CREATE INDEX IF NOT EXISTS idx_account_risk_limits_account ON account_risk_limits(account_id);

-- Seed default risk limits for existing accounts
INSERT INTO account_risk_limits (account_id)
SELECT id FROM accounts
ON CONFLICT (account_id) DO NOTHING;
