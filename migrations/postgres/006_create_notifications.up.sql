-- migrations/postgres/006_create_notifications.up.sql

CREATE TYPE notification_severity AS ENUM ('critical', 'warning', 'info', 'debug');
CREATE TYPE notification_status AS ENUM ('sent', 'failed', 'throttled', 'suppressed');
CREATE TYPE notification_channel AS ENUM ('telegram', 'email');

CREATE TABLE notifications (
    id          UUID                     PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type  VARCHAR(100)             NOT NULL,
    severity    notification_severity    NOT NULL,
    title       VARCHAR(255)             NOT NULL,
    message     TEXT                     NOT NULL,
    channel     notification_channel     NOT NULL DEFAULT 'telegram',
    status      notification_status      NOT NULL,
    delivered_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ              NOT NULL DEFAULT NOW(),
    metadata    JSONB                    DEFAULT '{}'
);

CREATE INDEX idx_notifications_created_at ON notifications (created_at DESC);
CREATE INDEX idx_notifications_severity   ON notifications (severity);

COMMENT ON TABLE notifications IS 'Notification delivery history. Retention limited to last 1000 rows via application logic.';
