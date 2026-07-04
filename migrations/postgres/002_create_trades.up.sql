-- migrations/postgres/002_create_trades.up.sql

CREATE TABLE trades (
    -- Primary key
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Trade identification
    client_order_id UUID        NOT NULL UNIQUE,
    strategy_id     VARCHAR(64) NOT NULL,

    -- Market details
    market_id       VARCHAR(128) NOT NULL,
    market_slug     VARCHAR(256) NOT NULL,
    side            VARCHAR(4)  NOT NULL CHECK (side IN ('YES', 'NO')),

    -- Order details
    order_type      VARCHAR(8)  NOT NULL DEFAULT 'GTC' CHECK (order_type IN ('GTC', 'FOK', 'GTD', 'FAK')),
    price           NUMERIC(12, 4) NOT NULL CHECK (price > 0),
    quantity        NUMERIC(20, 8) NOT NULL CHECK (quantity > 0),
    filled_quantity NUMERIC(20, 8) NOT NULL DEFAULT 0 CHECK (filled_quantity >= 0),

    -- Execution details
    fill_status     VARCHAR(16) NOT NULL CHECK (fill_status IN (
                        'PENDING', 'PLACED', 'FILLED', 'PARTIAL_FILL',
                        'CANCELLED', 'FAILED', 'EXPIRED'
                    )),
    latency_ms      INTEGER     NOT NULL CHECK (latency_ms >= 0),

    -- Financial
    pnl             NUMERIC(20, 8) NOT NULL DEFAULT 0,
    fee             NUMERIC(12, 4) NOT NULL DEFAULT 0 CHECK (fee >= 0),
    slippage_pct    NUMERIC(8, 4) NOT NULL DEFAULT 0,

    -- Timestamps
    signal_timestamp   TIMESTAMPTZ NOT NULL,
    order_timestamp    TIMESTAMPTZ NOT NULL,
    fill_timestamp     TIMESTAMPTZ,

    -- Context
    opportunity_id  UUID,
    risk_decision   VARCHAR(16) NOT NULL DEFAULT 'APPROVED',
    failure_reason  TEXT,

    -- Multi-account support
    account_id      UUID,

    -- Immutability guard
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for query performance
CREATE INDEX idx_trades_created_at       ON trades (created_at DESC);
CREATE INDEX idx_trades_strategy_id      ON trades (strategy_id, created_at DESC);
CREATE INDEX idx_trades_market_id        ON trades (market_id, created_at DESC);
CREATE INDEX idx_trades_side             ON trades (side, created_at DESC);
CREATE INDEX idx_trades_fill_status      ON trades (fill_status, created_at DESC);
CREATE INDEX idx_trades_pnl              ON trades (pnl, created_at DESC);
CREATE INDEX idx_trades_signal_timestamp ON trades (signal_timestamp DESC);

-- Composite indexes for common filter combinations
CREATE INDEX idx_trades_strategy_market  ON trades (strategy_id, market_id, created_at DESC);
CREATE INDEX idx_trades_date_pnl        ON trades (created_at DESC, pnl);
CREATE INDEX idx_trades_account         ON trades (account_id, created_at DESC) WHERE account_id IS NOT NULL;

-- Function to enforce immutability: block ALL UPDATE attempts
CREATE OR REPLACE FUNCTION enforce_trade_immutability()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Trade records are immutable: UPDATE is not permitted';
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_trade_immutability
    BEFORE UPDATE ON trades
    FOR EACH ROW
    EXECUTE FUNCTION enforce_trade_immutability();

-- Function to prevent DELETE
CREATE OR REPLACE FUNCTION prevent_trade_delete()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Trade records are immutable: DELETE is not permitted';
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_trade_no_delete
    BEFORE DELETE ON trades
    FOR EACH ROW
    EXECUTE FUNCTION prevent_trade_delete();

-- Revoke UPDATE and DELETE from application role and PUBLIC
REVOKE UPDATE, DELETE ON trades FROM pqap_app;
REVOKE UPDATE, DELETE ON trades FROM PUBLIC;

COMMENT ON TABLE trades IS 'Immutable trade history. Append-only. No UPDATE/DELETE permitted.';
COMMENT ON COLUMN trades.client_order_id IS 'UUID for idempotency — prevents duplicate records.';
COMMENT ON COLUMN trades.pnl IS 'Realized PnL in USDC. 8 decimal places. Zero for unfilled/cancelled orders.';
COMMENT ON COLUMN trades.latency_ms IS 'Time from opportunity signal to order placement in milliseconds.';
COMMENT ON COLUMN trades.account_id IS 'Nullable for future multi-account support. Default null.';
