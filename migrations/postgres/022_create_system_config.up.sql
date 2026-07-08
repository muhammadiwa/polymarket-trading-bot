-- System configuration table for admin panel
CREATE TABLE IF NOT EXISTS system_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_key VARCHAR(100) UNIQUE NOT NULL,
    config_value JSONB NOT NULL,
    category VARCHAR(50) NOT NULL CHECK (category IN ('api_keys', 'risk_defaults', 'notification_settings')),
    description TEXT,
    is_sensitive BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_system_config_category ON system_config(category);
CREATE INDEX IF NOT EXISTS idx_system_config_key ON system_config(config_key);

-- Config audit log for tracking all changes
CREATE TABLE IF NOT EXISTS config_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_key VARCHAR(100) NOT NULL,
    old_value JSONB,
    new_value JSONB NOT NULL,
    changed_by UUID REFERENCES users(id),
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason TEXT
);

CREATE INDEX IF NOT EXISTS idx_config_audit_log_key ON config_audit_log(config_key);
CREATE INDEX IF NOT EXISTS idx_config_audit_log_changed_at ON config_audit_log(changed_at DESC);

-- Seed default config values
INSERT INTO system_config (config_key, config_value, category, description, is_sensitive) VALUES
    ('daily_loss_limit_pct', '2.0', 'risk_defaults', 'Daily loss limit as percentage of capital', false),
    ('max_position_per_market_pct', '10.0', 'risk_defaults', 'Max position per market as percentage of capital', false),
    ('max_position_per_strategy_pct', '20.0', 'risk_defaults', 'Max position per strategy as percentage of capital', false),
    ('drawdown_circuit_breaker_pct', '10.0', 'risk_defaults', 'Drawdown circuit breaker threshold', false),
    ('win_streak_threshold', '5', 'risk_defaults', 'Batasi Win streak threshold', false),
    ('throttle_rate_per_min', '10', 'notification_settings', 'Max non-critical notifications per minute', false),
    ('critical_bypass_throttle', 'true', 'notification_settings', 'Critical notifications bypass throttle', false),
    ('enable_telegram', 'true', 'notification_settings', 'Enable Telegram notifications', false),
    ('enable_email', 'false', 'notification_settings', 'Enable email notifications', false),
    ('polymarket_api_key', '""', 'api_keys', 'Polymarket API key', true),
    ('telegram_bot_token', '""', 'api_keys', 'Telegram bot token', true),
    ('polymarket_secret', '""', 'api_keys', 'Polymarket API secret', true)
ON CONFLICT (config_key) DO NOTHING;
