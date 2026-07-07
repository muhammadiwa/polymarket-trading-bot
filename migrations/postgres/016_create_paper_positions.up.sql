CREATE TABLE paper_positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    market_id VARCHAR(128) NOT NULL,
    side VARCHAR(4) NOT NULL CHECK (side IN ('YES', 'NO')),
    entry_price NUMERIC(12,4) NOT NULL,
    quantity NUMERIC(20,8) NOT NULL,
    strategy_id VARCHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    pnl NUMERIC(20,8) DEFAULT 0,
    account_id UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_paper_positions_strategy ON paper_positions(strategy_id, status);
CREATE INDEX idx_paper_positions_market ON paper_positions(market_id, status);
