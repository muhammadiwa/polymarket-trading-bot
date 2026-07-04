CREATE TABLE IF NOT EXISTS opportunities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    market TEXT NOT NULL,
    market_slug TEXT NOT NULL,
    score NUMERIC(20, 8) NOT NULL,
    spread NUMERIC(20, 8) NOT NULL,
    fill_probability NUMERIC(20, 8) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT NOT NULL CHECK (status IN ('detected', 'executed', 'filtered')),
    filter_reason TEXT,
    execution_latency_ms INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_opportunities_timestamp ON opportunities (timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_opportunities_status ON opportunities (status);
CREATE INDEX IF NOT EXISTS idx_opportunities_market_slug ON opportunities (market_slug);
-- #4: Composite index for cursor pagination (status, timestamp DESC)
CREATE INDEX IF NOT EXISTS idx_opportunities_status_timestamp ON opportunities (status, timestamp DESC);

SELECT create_hypertable('opportunities', 'timestamp', if_not_exists => TRUE);
