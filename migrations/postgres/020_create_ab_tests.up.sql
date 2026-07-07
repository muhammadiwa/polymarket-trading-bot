CREATE TABLE optimizer_ab_tests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    suggestion_id UUID NOT NULL REFERENCES optimizer_suggestions(id),
    strategy_id VARCHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'completed', 'failed')),
    min_sample_size INTEGER NOT NULL DEFAULT 50,
    current_sample_size INTEGER NOT NULL DEFAULT 0,
    p_value DECIMAL(10,8),
    mean_difference DECIMAL(18,8),
    recommendation VARCHAR(20) CHECK (recommendation IN ('recommend', 'inconclusive', 'reject')),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    failed_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE optimizer_ab_test_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ab_test_id UUID NOT NULL REFERENCES optimizer_ab_tests(id),
    variant VARCHAR(10) NOT NULL CHECK (variant IN ('control', 'treatment')),
    market_id VARCHAR(100) NOT NULL,
    side VARCHAR(10) NOT NULL,
    entry_price DECIMAL(18,8) NOT NULL,
    exit_price DECIMAL(18,8),
    quantity DECIMAL(18,8) NOT NULL,
    pnl DECIMAL(18,8),
    simulated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ab_tests_suggestion ON optimizer_ab_tests(suggestion_id);
CREATE INDEX idx_ab_tests_status ON optimizer_ab_tests(status);
CREATE UNIQUE INDEX idx_ab_tests_unique_running ON optimizer_ab_tests(suggestion_id) WHERE status = 'running';
CREATE INDEX idx_ab_results_test ON optimizer_ab_test_results(ab_test_id, variant);
