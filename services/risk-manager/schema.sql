-- Risk Manager Database Schema
-- risk_events table — sole writer: risk-manager service (AD-6)
-- Append-only, immutable (no UPDATE/DELETE)

CREATE TABLE IF NOT EXISTS risk_events (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision                TEXT NOT NULL,            -- "ALLOW" or "DENY"
    reason                  TEXT NOT NULL,            -- "daily_limit", "market_limit", "strategy_limit", "emergency_stop", "correlation_limit_exceeded", "batasi_win_paused", "approved"
    market_id               TEXT DEFAULT NULL,
    strategy_id             TEXT DEFAULT NULL,
    trade_size              NUMERIC(18,8) NOT NULL DEFAULT 0,
    current_exposure        NUMERIC(18,8) NOT NULL DEFAULT 0,
    limit_value             NUMERIC(18,8) NOT NULL DEFAULT 0,
    daily_budget_remaining  NUMERIC(18,8) NOT NULL,
    capital                 NUMERIC(18,8) NOT NULL,
    context                 JSONB DEFAULT '{}',
    account_id              UUID DEFAULT NULL,        -- nullable, for future multi-account
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_risk_events_created_at ON risk_events(created_at);
CREATE INDEX IF NOT EXISTS idx_risk_events_decision ON risk_events(decision);
CREATE INDEX IF NOT EXISTS idx_risk_events_reason ON risk_events(reason);
CREATE INDEX IF NOT EXISTS idx_risk_events_market_id ON risk_events(market_id);
CREATE INDEX IF NOT EXISTS idx_risk_events_strategy_id ON risk_events(strategy_id);
CREATE INDEX IF NOT EXISTS idx_risk_events_account_id ON risk_events(account_id);

-- Correlation groups — auto-detected by Correlation Engine
CREATE TABLE IF NOT EXISTS correlation_groups (
    id                  TEXT PRIMARY KEY,
    name                TEXT NOT NULL,
    detection_method    TEXT NOT NULL,           -- "category", "correlation", "keyword"
    market_ids          TEXT[] NOT NULL,
    max_positions       INT NOT NULL DEFAULT 3,
    confidence          NUMERIC(3,2) NOT NULL DEFAULT 0.0,
    last_updated        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_correlation_groups_detection_method ON correlation_groups(detection_method);
CREATE INDEX IF NOT EXISTS idx_correlation_groups_last_updated ON correlation_groups(last_updated);
