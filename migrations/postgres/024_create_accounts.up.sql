CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    wallet_address VARCHAR(100) NOT NULL,
    private_key_encrypted BYTEA NOT NULL,
    private_key_iv BYTEA NOT NULL,
    private_key_tag BYTEA NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_accounts_wallet ON accounts(wallet_address);
CREATE INDEX idx_accounts_active ON accounts(is_active);
