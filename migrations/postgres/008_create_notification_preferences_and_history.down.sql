-- migrations/postgres/008_create_notification_preferences_and_history.down.sql

DROP TABLE IF EXISTS notification_history;
DROP TABLE IF EXISTS notification_preferences;
