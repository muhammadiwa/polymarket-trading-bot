CREATE TABLE optimizer_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id VARCHAR(64) NOT NULL,
    pattern_type VARCHAR(50) NOT NULL,
    parameter_name VARCHAR(100) NOT NULL,
    current_value TEXT NOT NULL,
    suggested_value TEXT NOT NULL,
    expected_impact TEXT NOT NULL,
    methodology TEXT NOT NULL,
    confidence DECIMAL(5,4) NOT NULL,
    p_value DECIMAL(10,8),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    reviewed_by UUID,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_optimizer_suggestions_strategy ON optimizer_suggestions(strategy_id, status);
CREATE INDEX idx_optimizer_suggestions_created ON optimizer_suggestions(created_at DESC);
