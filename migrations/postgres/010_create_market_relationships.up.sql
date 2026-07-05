CREATE TABLE market_relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    market_a_id VARCHAR(255) NOT NULL,
    market_b_id VARCHAR(255) NOT NULL,
    relationship_type VARCHAR(50) NOT NULL CHECK (relationship_type IN ('same_event', 'date_variant', 'correlated_outcome')),
    confidence DECIMAL(5,4) NOT NULL DEFAULT 0.8000,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(market_a_id, market_b_id, relationship_type)
);

CREATE INDEX idx_market_relationships_a ON market_relationships(market_a_id);
CREATE INDEX idx_market_relationships_b ON market_relationships(market_b_id);
CREATE INDEX idx_market_relationships_type ON market_relationships(relationship_type);
