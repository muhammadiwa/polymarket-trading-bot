import logging
from datetime import datetime, timezone
from typing import Optional

import asyncpg

from app.models.optimizer import SuggestionResponse

logger = logging.getLogger(__name__)


async def get_trades(conn: asyncpg.Connection, strategy_id: str, limit: int = 10000) -> list[dict]:
    rows = await conn.fetch(
        """
        SELECT pnl, score, quantity, side, fill_status, fill_timestamp, market_id, strategy_id
        FROM trades
        WHERE strategy_id = $1 AND fill_status IN ('FILLED', 'PARTIAL_FILL')
        ORDER BY fill_timestamp DESC
        LIMIT $2
        """,
        strategy_id, limit,
    )
    return [dict(r) for r in rows]


async def count_trades(conn: asyncpg.Connection, strategy_id: str) -> int:
    row = await conn.fetchrow(
        "SELECT COUNT(*) as cnt FROM trades WHERE strategy_id = $1 AND fill_status IN ('FILLED', 'PARTIAL_FILL')",
        strategy_id,
    )
    return row["cnt"] if row else 0


async def save_suggestion(conn: asyncpg.Connection, suggestion: dict) -> str:
    row = await conn.fetchrow(
        """
        INSERT INTO optimizer_suggestions
            (strategy_id, pattern_type, parameter_name, current_value, suggested_value,
             expected_impact, methodology, confidence, p_value)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id
        """,
        suggestion["strategy_id"],
        suggestion["pattern_type"],
        suggestion["parameter_name"],
        suggestion["current_value"],
        suggestion["suggested_value"],
        suggestion["expected_impact"],
        suggestion["methodology"],
        suggestion["confidence"],
        suggestion.get("p_value"),
    )
    return str(row["id"])


async def get_suggestions(
    conn: asyncpg.Connection,
    strategy_id: Optional[str] = None,
    status: Optional[str] = None,
    limit: int = 50,
    offset: int = 0,
) -> tuple[list[dict], int]:
    conditions = []
    params = []
    idx = 1

    if strategy_id:
        conditions.append(f"strategy_id = ${idx}")
        params.append(strategy_id)
        idx += 1
    if status:
        conditions.append(f"status = ${idx}")
        params.append(status)
        idx += 1

    where = f"WHERE {' AND '.join(conditions)}" if conditions else ""

    count_row = await conn.fetchrow(f"SELECT COUNT(*) as total FROM optimizer_suggestions {where}", *params)
    total = count_row["total"]

    rows = await conn.fetch(
        f"SELECT * FROM optimizer_suggestions {where} ORDER BY created_at DESC LIMIT ${idx} OFFSET ${idx + 1}",
        *params, limit, offset,
    )
    return [_row_to_suggestion(r) for r in rows], total


async def update_suggestion_status(
    conn: asyncpg.Connection,
    suggestion_id: str,
    status: str,
    reviewed_by: str,
) -> Optional[dict]:
    row = await conn.fetchrow(
        """
        UPDATE optimizer_suggestions
        SET status = $1, reviewed_by = $2::uuid, reviewed_at = NOW()
        WHERE id = $3::uuid AND status = 'pending'
        RETURNING *
        """,
        status, reviewed_by, suggestion_id,
    )
    if row is None:
        return None
    return _row_to_suggestion(row)


def _row_to_suggestion(row: asyncpg.Record) -> dict:
    return {
        "id": str(row["id"]),
        "strategy_id": row["strategy_id"],
        "pattern_type": row["pattern_type"],
        "parameter_name": row["parameter_name"],
        "current_value": row["current_value"],
        "suggested_value": row["suggested_value"],
        "expected_impact": row["expected_impact"],
        "methodology": row["methodology"],
        "confidence": float(row["confidence"]),
        "p_value": float(row["p_value"]) if row["p_value"] else None,
        "status": row["status"],
        "reviewed_by": str(row["reviewed_by"]) if row.get("reviewed_by") else None,
        "reviewed_at": row.get("reviewed_at"),
        "created_at": row["created_at"],
    }
