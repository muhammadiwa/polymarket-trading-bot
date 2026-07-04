package history

// Immutability guarantees for the trade history system:
//
// 1. Database level:
//    - REVOKE UPDATE, DELETE ON trades FROM pqap_app (migration)
//    - trg_trade_immutability trigger prevents modification of created_at/updated_at
//
// 2. Application level:
//    - Repository only exposes Insert() and GetByClientOrderID()
//    - No Update() or Delete() methods exist
//    - Insert uses ON CONFLICT (client_order_id) DO NOTHING for idempotency
//    - Raw SQL prevents accidental bulk updates (no ORM)
//
// 3. Audit:
//    - All inserts are logged with trade_id and client_order_id
//    - TradeRecorded event published to NATS for downstream consumers
//    - Prometheus metrics track all trade record operations
