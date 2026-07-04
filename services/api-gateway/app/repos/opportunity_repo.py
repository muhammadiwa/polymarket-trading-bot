from datetime import datetime
from typing import Optional

import asyncpg

OPPORTUNITY_COLUMNS = (
    "id, market, market_slug, score, spread, fill_probability, "
    "timestamp, status, filter_reason, execution_latency_ms"
)


def _row_to_opportunity(row: asyncpg.Record) -> dict:
    ts = row["timestamp"]
    return {
        "id": str(row["id"]),
        "market": row["market"],
        "market_slug": row["market_slug"],
        "score": str(row["score"]),
        "spread": str(row["spread"]),
        "fill_probability": str(row["fill_probability"]),
        "timestamp": ts.isoformat() if ts is not None else None,
        "status": row["status"],
        "filter_reason": row["filter_reason"],
        "execution_latency_ms": row["execution_latency_ms"],
    }


async def list_opportunities(
    conn: asyncpg.Connection,
    cursor: Optional[str] = None,
    page_size: int = 50,
    status_filter: Optional[str] = None,
) -> tuple[list[dict], int, Optional[str]]:
    conditions: list[str] = []
    params: list = []
    idx = 1

    if status_filter:
        conditions.append(f"status = ${idx}")
        params.append(status_filter)
        idx += 1

    if cursor:
        cursor_dt = datetime.fromisoformat(cursor)
        conditions.append(f"timestamp < ${idx}")
        params.append(cursor_dt)
        idx += 1

    where = " AND ".join(conditions) if conditions else "1=1"

    count_query = f"SELECT COUNT(*) FROM opportunities WHERE {where}"
    total_count: int = await conn.fetchval(count_query, *params)

    # #21: Enumerate specific columns instead of SELECT *
    query = (
        f"SELECT {OPPORTUNITY_COLUMNS} FROM opportunities WHERE {where} "
        f"ORDER BY timestamp DESC "
        f"LIMIT ${idx}"
    )
    params.append(page_size + 1)
    rows = await conn.fetch(query, *params)

    next_cursor: Optional[str] = None
    if len(rows) > page_size:
        last_row = rows[page_size - 1]
        ts = last_row["timestamp"]
        next_cursor = ts.isoformat() if ts is not None else None
        rows = rows[:page_size]

    opportunities = [_row_to_opportunity(row) for row in rows]
    return opportunities, total_count, next_cursor
