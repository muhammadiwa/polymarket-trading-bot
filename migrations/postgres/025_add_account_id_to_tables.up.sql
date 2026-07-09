ALTER TABLE trades ADD COLUMN account_id UUID REFERENCES accounts(id);
ALTER TABLE positions ADD COLUMN account_id UUID REFERENCES accounts(id);
ALTER TABLE risk_events ADD COLUMN account_id UUID REFERENCES accounts(id);

CREATE INDEX idx_trades_account ON trades(account_id);
CREATE INDEX idx_positions_account ON positions(account_id);
CREATE INDEX idx_risk_events_account ON risk_events(account_id);
