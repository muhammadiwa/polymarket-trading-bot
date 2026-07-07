CREATE TABLE paper_trades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    market_id VARCHAR(128) NOT NULL,
    side VARCHAR(4) NOT NULL CHECK (side IN ('YES', 'NO')),
    price NUMERIC(12,4) NOT NULL,
    quantity NUMERIC(20,8) NOT NULL,
    pnl NUMERIC(20,8) NOT NULL DEFAULT 0,
    strategy_id VARCHAR(64) NOT NULL,
    fill_status VARCHAR(16) NOT NULL,
    simulated_latency_ms INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_paper_trades_strategy ON paper_trades(strategy_id, created_at DESC);
CREATE INDEX idx_paper_trades_market ON paper_trades(market_id, created_at DESC);
