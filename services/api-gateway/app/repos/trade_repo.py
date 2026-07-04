from datetime import datetime
from decimal import Decimal
from typing import Optional

import asyncpg

from app.models.trade import (
    FillStatusEnum,
    TradeFilterParams,
    TradeResponse,
    TradeStatsResponse,
)


def _row_to_trade(row: asyncpg.Record) -> TradeResponse:
    return TradeResponse(
        id=str(row["id"]),
        client_order_id=str(row["client_order_id"]),
        strategy_id=row["strategy_id"],
        market_id=row["market_id"],
        market_slug=row["market_slug"],
        side=row["side"],
        order_type=row["order_type"],
        price=Decimal(str(row["price"])),
        quantity=Decimal(str(row["quantity"])),
        filled_quantity=Decimal(str(row["filled_quantity"])),
        fill_status=FillStatusEnum(row["fill_status"]),
        latency_ms=row["latency_ms"],
        pnl=Decimal(str(row["pnl"])),
        fee=Decimal(str(row["fee"])),
        slippage_pct=Decimal(str(row["slippage_pct"])),
        signal_timestamp=row["signal_timestamp"],
        order_timestamp=row["order_timestamp"],
        fill_timestamp=row["fill_timestamp"],
        opportunity_id=str(row["opportunity_id"]) if row["opportunity_id"] else None,
        risk_decision=row["risk_decision"],
        failure_reason=row["failure_reason"],
        account_id=str(row["account_id"]) if row["account_id"] else None,
        created_at=row["created_at"],
    )


def _build_where(filters: TradeFilterParams) -> tuple[str, list]:
    conditions: list[str] = []
    params: list = []
    idx = 1

    if filters.start_date is not None:
        conditions.append(f"created_at >= ${idx}")
        params.append(filters.start_date)
        idx += 1
    if filters.end_date is not None:
        conditions.append(f"created_at <= ${idx}")
        params.append(filters.end_date)
        idx += 1
    if filters.market_id is not None:
        conditions.append(f"market_id = ${idx}")
        params.append(filters.market_id)
        idx += 1
    if filters.strategy_id is not None:
        conditions.append(f"strategy_id = ${idx}")
        params.append(filters.strategy_id)
        idx += 1
    if filters.side is not None:
        conditions.append(f"side = ${idx}")
        params.append(filters.side)
        idx += 1
    if filters.pnl_sign == "positive":
        conditions.append("pnl > 0")
    elif filters.pnl_sign == "negative":
        conditions.append("pnl < 0")
    if filters.fill_status is not None:
        conditions.append(f"fill_status = ${idx}")
        params.append(filters.fill_status.value)
        idx += 1

    where = " AND ".join(conditions) if conditions else "1=1"
    return where, params


async def list_trades(
    conn: asyncpg.Connection,
    filters: TradeFilterParams,
) -> tuple[list[TradeResponse], int, Optional[str]]:
    where, params = _build_where(filters)

    count_query = f"SELECT COUNT(*) FROM trades WHERE {where}"
    total_count: int = await conn.fetchval(count_query, *params)

    next_cursor: Optional[str] = None

    if filters.cursor:
        cursor_dt = datetime.fromisoformat(filters.cursor)
        cursor_cond = f"created_at < ${len(params) + 1}"
        if where == "1=1":
            where = cursor_cond
        else:
            where = f"({where}) AND {cursor_cond}"
        params.append(cursor_dt)

    query = (
        f"SELECT * FROM trades WHERE {where} "
        f"ORDER BY created_at DESC "
        f"LIMIT ${len(params) + 1}"
    )
    params.append(filters.page_size + 1)
    rows = await conn.fetch(query, *params)

    if len(rows) > filters.page_size:
        last_row = rows[filters.page_size - 1]
        next_cursor = last_row["created_at"].isoformat()
        rows = rows[:filters.page_size]

    trades = [_row_to_trade(row) for row in rows]
    return trades, total_count, next_cursor


async def get_trade(
    conn: asyncpg.Connection,
    trade_id: str,
) -> Optional[TradeResponse]:
    row = await conn.fetchrow(
        "SELECT * FROM trades WHERE id = $1",
        trade_id,
    )
    if row is None:
        return None
    return _row_to_trade(row)


async def export_trades(
    conn: asyncpg.Connection,
    filters: TradeFilterParams,
    batch_size: int = 1000,
):
    where, params = _build_where(filters)
    query = f"SELECT * FROM trades WHERE {where} ORDER BY created_at DESC"
    async with conn.transaction():
        cursor = await conn.cursor(query, *params)
        while True:
            rows = await cursor.fetch(batch_size)
            if not rows:
                break
            for row in rows:
                yield _row_to_trade(row)


async def get_stats(
    conn: asyncpg.Connection,
    filters: TradeFilterParams,
) -> TradeStatsResponse:
    where, params = _build_where(filters)

    stats_query = f"""
        SELECT
            COUNT(*) as total_trades,
            COALESCE(SUM(pnl), 0) as total_pnl,
            COALESCE(AVG(latency_ms), 0) as avg_latency_ms,
            COALESCE(COUNT(*) FILTER (WHERE pnl > 0), 0) as winning_trades
        FROM trades
        WHERE {where}
    """
    row = await conn.fetchrow(stats_query, *params)

    total_trades = row["total_trades"]
    total_pnl = Decimal(str(row["total_pnl"]))
    avg_latency = Decimal(str(row["avg_latency_ms"]))
    winning_trades = row["winning_trades"]

    win_rate = (
        Decimal(str(winning_trades)) / Decimal(str(total_trades))
        if total_trades > 0
        else Decimal("0")
    )

    strategy_query = f"""
        SELECT strategy_id, COUNT(*) as cnt
        FROM trades
        WHERE {where}
        GROUP BY strategy_id
        ORDER BY cnt DESC
    """
    strategy_rows = await conn.fetch(strategy_query, *params)
    trades_by_strategy = {row["strategy_id"]: row["cnt"] for row in strategy_rows}

    return TradeStatsResponse(
        total_trades=total_trades,
        total_pnl=total_pnl,
        win_rate=win_rate,
        avg_latency_ms=avg_latency,
        trades_by_strategy=trades_by_strategy,
    )
