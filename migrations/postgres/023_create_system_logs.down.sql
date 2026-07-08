-- Drop indexes for database_backups
DROP INDEX IF EXISTS idx_database_backups_created;
DROP INDEX IF EXISTS idx_database_backups_status;

-- Drop indexes for system_logs
DROP INDEX IF EXISTS idx_system_logs_message_fts;
DROP INDEX IF EXISTS idx_system_logs_request_id;
DROP INDEX IF EXISTS idx_system_logs_service;
DROP INDEX IF EXISTS idx_system_logs_level;

-- Drop tables
DROP TABLE IF EXISTS database_backups;
DROP TABLE IF EXISTS system_logs;
