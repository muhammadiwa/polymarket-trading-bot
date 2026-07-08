import logging
from datetime import datetime, timedelta, timezone
from decimal import Decimal, InvalidOperation
from typing import Optional

import asyncpg

logger = logging.getLogger(__name__)

PERIOD_MAP = {
    "today": timedelta(days=1),
    "yesterday": timedelta(days=1),
    "week": timedelta(days=7),
    "month": timedelta(days=30),
    "year": timedelta(days=365),
}


def _empty_pnl_summary(period: str) -> dict:
    """Return empty PnL summary for when no trades exist."""
    return {
        "period": period,
        "total_pnl": Decimal("0"),
        "trade_count": 0,
        "winning_trades": 0,
        "losing_trades": 0,
        "win_rate": 0.0,
        "avg_pnl": Decimal("0"),
        "best_trade": Decimal("0"),
        "worst_trade": Decimal("0"),
    }


async def get_pnl_summary(conn: asyncpg.Connection, user_id: str, period: str = "week") -> dict:
    """Get PnL summary for a given period.
    Note: trades table uses strategy_id, not user_id. Filter by strategy if available.
    """
    delta = PERIOD_MAP.get(period, timedelta(days=7))
    start = datetime.now(timezone.utc) - delta

    # #8: Filter by user's strategies via strategy_id if available
    # For now, query all trades (trades table has strategy_id, not user_id)
    row = await conn.fetchrow(
        """
        SELECT
            COALESCE(SUM(pnl), 0) as total_pnl,
            COUNT(*) as trade_count,
            COUNT(*) FILTER (WHERE pnl > 0) as winning_trades,
            COUNT(*) FILTER (WHERE pnl < 0) as losing_trades,
            COALESCE(AVG(pnl), 0) as avg_pnl,
            COALESCE(MAX(pnl), 0) as best_trade,
            COALESCE(MIN(pnl), 0) as worst_trade
        FROM trades
        WHERE fill_timestamp >= $1 AND fill_status IN ('FILLED', 'PARTIAL_FILL')
        """,
        start,
    )

    if row is None or row["trade_count"] == 0:
        return _empty_pnl_summary(period)

    total_count = row["trade_count"]
    wins = row["winning_trades"]
    win_rate = (wins / total_count * 100) if total_count > 0 else 0

    return {
        "period": period,
        "total_pnl": Decimal(str(row["total_pnl"])),
        "trade_count": row["trade_count"],
        "winning_trades": row["winning_trades"],
        "losing_trades": row["losing_trades"],
        "win_rate": round(win_rate, 2),
        "avg_pnl": Decimal(str(row["avg_pnl"])),
        "best_trade": Decimal(str(row["best_trade"])),
        "worst_trade": Decimal(str(row["worst_trade"])),
    }


async def get_trades_by_market(conn: asyncpg.Connection, market_id: str, limit: int = 20) -> list[dict]:
    """Get trades for a specific market."""
    rows = await conn.fetch(
        """
        SELECT id, market_id, side, entry_price, exit_price, quantity, pnl,
               fill_status, fill_timestamp, strategy_id
        FROM trades
        WHERE market_id = $1 AND fill_status IN ('FILLED', 'PARTIAL_FILL')
        ORDER BY fill_timestamp DESC
        LIMIT $2
        """,
        market_id, limit,
    )
    return [_row_to_trade(r) for r in rows]


async def get_trade_by_id(conn: asyncpg.Connection, trade_id: str) -> Optional[dict]:
    """Get a single trade by ID."""
    row = await conn.fetchrow(
        """
        SELECT id, market_id, side, entry_price, exit_price, quantity, pnl,
               fill_status, fill_timestamp, strategy_id
        FROM trades
        WHERE id = $1::uuid
        """,
        trade_id,
    )
    if row is None:
        return None
    return _row_to_trade(row)


async def get_trade_context(conn: asyncpg.Connection, trade_id: str) -> dict:
    """Get trade with decision context (risk events, opportunity data)."""
    trade = await get_trade_by_id(conn, trade_id)
    if trade is None:
        return {"trade": None, "risk_events": [], "opportunities": []}

    risk_events = await conn.fetch(
        """
        SELECT id, event_type, decision, reason, context, created_at
        FROM risk_events
        WHERE context->>'trade_id' = $1
        ORDER BY created_at DESC
        LIMIT 10
        """,
        trade_id,
    )

    return {
        "trade": trade,
        "risk_events": [dict(r) for r in risk_events],
    }


async def get_pnl_by_strategy(conn: asyncpg.Connection, period: str = "week") -> list[dict]:
    """Get PnL breakdown by strategy."""
    delta = PERIOD_MAP.get(period, timedelta(days=7))
    start = datetime.now(timezone.utc) - delta

    rows = await conn.fetch(
        """
        SELECT strategy_id,
               COALESCE(SUM(pnl), 0) as total_pnl,
               COUNT(*) as trade_count,
               COUNT(*) FILTER (WHERE pnl > 0) as winning_trades
        FROM trades
        WHERE fill_timestamp >= $1 AND fill_status IN ('FILLED', 'PARTIAL_FILL')
        GROUP BY strategy_id
        ORDER BY total_pnl DESC
        """,
        start,
    )

    result = []
    for row in rows:
        total_count = row["trade_count"]
        win_rate = (row["winning_trades"] / total_count * 100) if total_count > 0 else 0
        result.append({
            "strategy_id": row["strategy_id"],
            "total_pnl": Decimal(str(row["total_pnl"])),
            "trade_count": total_count,
            "win_rate": round(win_rate, 2),
        })
    return result


def _safe_decimal(value) -> Optional[Decimal]:
    """Safely convert value to Decimal, returning None for None/invalid values."""
    if value is None:
        return None
    try:
        return Decimal(str(value))
    except (InvalidOperation, ValueError, TypeError):
        return None


def _row_to_trade(row: asyncpg.Record) -> dict:
    return {
        "id": str(row["id"]),
        "market_id": row["market_id"],
        "side": row["side"],
        "entry_price": _safe_decimal(row.get("entry_price")),
        "exit_price": _safe_decimal(row.get("exit_price")),
        "quantity": _safe_decimal(row.get("quantity")),
        "pnl": _safe_decimal(row.get("pnl")),
        "fill_status": row.get("fill_status"),
        "fill_timestamp": row.get("fill_timestamp"),
        "strategy_id": row.get("strategy_id"),
        "slippage_pct": _safe_decimal(row.get("slippage_pct")),
    }


async def get_trades(conn: asyncpg.Connection, limit: int = 500) -> list[dict]:
    """Get recent filled trades for analysis."""
    rows = await conn.fetch(
        """
        SELECT id, market_id, side, entry_price, exit_price, quantity, pnl,
               fill_status, fill_timestamp, strategy_id, slippage_pct
        FROM trades
        WHERE fill_status IN ('FILLED', 'PARTIAL_FILL')
        ORDER BY fill_timestamp DESC
        LIMIT $1
        """,
        limit,
    )
    return [_row_to_trade(r) for r in rows]
