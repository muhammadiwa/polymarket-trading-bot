-- migrations/postgres/003_create_risk_parameters.up.sql

CREATE TABLE risk_parameters (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    daily_loss_limit    NUMERIC(12,4),
    max_position_per_market NUMERIC(12,4),
    max_position_per_strategy NUMERIC(12,4),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_risk_parameters_updated_at ON risk_parameters (updated_at DESC);

COMMENT ON TABLE risk_parameters IS 'Audit log of risk parameter changes. Append-only.';
