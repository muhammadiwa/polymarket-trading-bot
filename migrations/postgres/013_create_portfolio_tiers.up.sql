CREATE TABLE portfolio_tiers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID,
    current_tier INT NOT NULL DEFAULT 1,
    total_capital DECIMAL(20,8) NOT NULL DEFAULT 0,
    deployed_capital DECIMAL(20,8) NOT NULL DEFAULT 0,
    utilization_rate DECIMAL(5,4) NOT NULL DEFAULT 0,
    days_above_threshold INT NOT NULL DEFAULT 0,
    promotion_threshold DECIMAL(20,8),
    promoted_at TIMESTAMPTZ,
    demoted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tier_transitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID,
    from_tier INT NOT NULL,
    to_tier INT NOT NULL,
    capital_at_transition DECIMAL(20,8) NOT NULL,
    reason VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE rebalance_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID,
    old_weights JSONB NOT NULL,
    new_weights JSONB NOT NULL,
    initiated_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_portfolio_tiers_account ON portfolio_tiers(account_id);
CREATE INDEX idx_tier_transitions_account ON tier_transitions(account_id);
CREATE INDEX idx_rebalance_log_account ON rebalance_log(account_id);
