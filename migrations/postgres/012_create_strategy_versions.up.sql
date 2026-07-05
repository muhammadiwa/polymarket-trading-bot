CREATE TABLE strategy_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id UUID NOT NULL REFERENCES strategies(id) ON DELETE RESTRICT,
    version_number INT NOT NULL,
    
    -- Full parameter snapshot
    parameters JSONB NOT NULL,
    
    -- Change summary
    change_summary TEXT NOT NULL DEFAULT '',
    changed_by UUID,
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_strategy_version UNIQUE (strategy_id, version_number)
);

CREATE INDEX idx_strategy_versions_strategy ON strategy_versions(strategy_id);
CREATE INDEX idx_strategy_versions_number ON strategy_versions(strategy_id, version_number DESC);
