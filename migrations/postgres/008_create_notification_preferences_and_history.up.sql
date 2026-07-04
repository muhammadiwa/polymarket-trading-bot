-- migrations/postgres/008_create_notification_preferences_and_history.up.sql

CREATE TABLE IF NOT EXISTS notification_preferences (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    critical        BOOLEAN NOT NULL DEFAULT TRUE,
    warning         BOOLEAN NOT NULL DEFAULT TRUE,
    info            BOOLEAN NOT NULL DEFAULT TRUE,
    debug           BOOLEAN NOT NULL DEFAULT FALSE,
    telegram_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    email_enabled   BOOLEAN NOT NULL DEFAULT FALSE,
    telegram_chat_id VARCHAR(255),
    email_address   VARCHAR(255),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO notification_preferences (id, critical, warning, info, debug, telegram_enabled, email_enabled)
VALUES ('00000000-0000-0000-0000-000000000001', TRUE, TRUE, TRUE, FALSE, TRUE, FALSE)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS notification_history (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category    VARCHAR(20) NOT NULL,
    title       VARCHAR(255) NOT NULL,
    message     TEXT NOT NULL,
    channel     VARCHAR(20) NOT NULL,
    status      VARCHAR(20) NOT NULL,
    sent_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notification_history_created ON notification_history(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notification_history_category ON notification_history(category);
CREATE INDEX IF NOT EXISTS idx_notification_history_status ON notification_history(status);

COMMENT ON TABLE notification_history IS 'Stores last 1000 notifications. Auto-cleaned by application.';
COMMENT ON TABLE notification_preferences IS 'Notification preferences per category and channel.';
