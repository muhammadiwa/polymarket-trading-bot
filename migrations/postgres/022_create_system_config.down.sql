-- Drop indexes
DROP INDEX IF EXISTS idx_config_audit_log_changed_at;
DROP INDEX IF EXISTS idx_config_audit_log_key;
DROP INDEX IF EXISTS idx_system_config_key;
DROP INDEX IF EXISTS idx_system_config_category;

-- Drop tables
DROP TABLE IF EXISTS config_audit_log;
DROP TABLE IF EXISTS system_config;
