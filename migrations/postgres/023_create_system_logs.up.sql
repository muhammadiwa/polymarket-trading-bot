-- System logs table (TimescaleDB hypertable for time-series data)
CREATE TABLE IF NOT EXISTS system_logs (
    id UUID DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    level VARCHAR(10) NOT NULL CHECK (level IN ('debug', 'info', 'warn', 'error', 'fatal')),
    service VARCHAR(50) NOT NULL,
    request_id VARCHAR(100),
    message TEXT NOT NULL,
    context JSONB
);

-- Convert to hypertable for time-series optimization (only if TimescaleDB is available)
-- If TimescaleDB is not installed, this will be a regular table
DO $$
BEGIN
    PERFORM create_hypertable('system_logs', 'timestamp', chunk_time_interval => INTERVAL '1 day');
EXCEPTION WHEN OTHERS THEN
    -- TimescaleDB not available, table will remain as regular PostgreSQL table
    NULL;
END $$;

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_system_logs_level ON system_logs (level, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_system_logs_service ON system_logs (service, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_system_logs_request_id ON system_logs (request_id) WHERE request_id IS NOT NULL;

-- Full-text search index
CREATE INDEX IF NOT EXISTS idx_system_logs_message_fts ON system_logs USING gin(to_tsvector('english', message));

-- Database backups tracking table
CREATE TABLE IF NOT EXISTS database_backups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    filename VARCHAR(255) NOT NULL,
    file_path TEXT NOT NULL,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'in_progress' CHECK (status IN ('completed', 'failed', 'in_progress')),
    duration_ms INTEGER,
    triggered_by VARCHAR(20) NOT NULL DEFAULT 'manual' CHECK (triggered_by IN ('manual', 'scheduled')),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_database_backups_status ON database_backups (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_database_backups_created ON database_backups (created_at DESC);
