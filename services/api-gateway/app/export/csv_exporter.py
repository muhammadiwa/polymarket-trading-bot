import csv
import io
from datetime import datetime, timezone
from decimal import Decimal
from typing import AsyncIterator

from app.models.trade import TradeResponse

CSV_HEADERS = [
    "id",
    "client_order_id",
    "strategy_id",
    "market_id",
    "market_slug",
    "side",
    "order_type",
    "price",
    "quantity",
    "filled_quantity",
    "fill_status",
    "latency_ms",
    "pnl",
    "fee",
    "slippage_pct",
    "signal_timestamp",
    "order_timestamp",
    "fill_timestamp",
    "opportunity_id",
    "risk_decision",
    "failure_reason",
    "account_id",
    "created_at",
]


def _format_value(val: object) -> str:
    if val is None:
        return ""
    if isinstance(val, Decimal):
        return str(val)
    if isinstance(val, datetime):
        return val.isoformat()
    if isinstance(val, bool):
        return str(val).lower()
    return str(val)


def generate_csv_header() -> str:
    buf = io.StringIO()
    writer = csv.writer(buf, quoting=csv.QUOTE_MINIMAL)
    writer.writerow(CSV_HEADERS)
    return buf.getvalue()


def generate_csv_rows(trades: list[TradeResponse]) -> str:
    buf = io.StringIO()
    writer = csv.writer(buf, quoting=csv.QUOTE_MINIMAL)
    for trade in trades:
        writer.writerow([
            _format_value(trade.id),
            _format_value(trade.client_order_id),
            _format_value(trade.strategy_id),
            _format_value(trade.market_id),
            _format_value(trade.market_slug),
            _format_value(trade.side),
            _format_value(trade.order_type),
            _format_value(trade.price),
            _format_value(trade.quantity),
            _format_value(trade.filled_quantity),
            _format_value(trade.fill_status.value),
            _format_value(trade.latency_ms),
            _format_value(trade.pnl),
            _format_value(trade.fee),
            _format_value(trade.slippage_pct),
            _format_value(trade.signal_timestamp),
            _format_value(trade.order_timestamp),
            _format_value(trade.fill_timestamp),
            _format_value(trade.opportunity_id),
            _format_value(trade.risk_decision),
            _format_value(trade.failure_reason),
            _format_value(trade.account_id),
            _format_value(trade.created_at),
        ])
    return buf.getvalue()


async def stream_csv(trades: list[TradeResponse]) -> AsyncIterator[str]:
    yield generate_csv_header()
    batch_size = 1000
    for i in range(0, len(trades), batch_size):
        batch = trades[i : i + batch_size]
        yield generate_csv_rows(batch)


def get_csv_filename() -> str:
    return f"trades_export_{datetime.now(timezone.utc).strftime('%Y%m%d_%H%M%S')}.csv"
