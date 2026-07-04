from datetime import datetime, timezone
from decimal import Decimal
from typing import AsyncIterator

import orjson

from app.models.trade import TradeResponse


def _trade_to_dict(trade: TradeResponse) -> dict:
    return {
        "id": trade.id,
        "client_order_id": trade.client_order_id,
        "strategy_id": trade.strategy_id,
        "market_id": trade.market_id,
        "market_slug": trade.market_slug,
        "side": trade.side,
        "order_type": trade.order_type,
        "price": str(trade.price),
        "quantity": str(trade.quantity),
        "filled_quantity": str(trade.filled_quantity),
        "fill_status": trade.fill_status.value,
        "latency_ms": trade.latency_ms,
        "pnl": str(trade.pnl),
        "fee": str(trade.fee),
        "slippage_pct": str(trade.slippage_pct),
        "signal_timestamp": trade.signal_timestamp.isoformat(),
        "order_timestamp": trade.order_timestamp.isoformat(),
        "fill_timestamp": trade.fill_timestamp.isoformat() if trade.fill_timestamp else None,
        "opportunity_id": trade.opportunity_id,
        "risk_decision": trade.risk_decision,
        "failure_reason": trade.failure_reason,
        "account_id": trade.account_id,
        "created_at": trade.created_at.isoformat(),
    }


def trade_to_bytes(trade: TradeResponse) -> bytes:
    return orjson.dumps(_trade_to_dict(trade))


async def stream_json(trades: list[TradeResponse]) -> AsyncIterator[bytes]:
    yield b"["
    for i, trade in enumerate(trades):
        if i > 0:
            yield b","
        yield orjson.dumps(_trade_to_dict(trade))
    yield b"]"


def get_json_filename() -> str:
    return f"trades_export_{datetime.now(timezone.utc).strftime('%Y%m%d_%H%M%S')}.json"
