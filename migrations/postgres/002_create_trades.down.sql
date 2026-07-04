DROP TRIGGER IF EXISTS trg_trade_immutability ON trades;
DROP FUNCTION IF EXISTS enforce_trade_immutability();
DROP TABLE IF EXISTS trades;
