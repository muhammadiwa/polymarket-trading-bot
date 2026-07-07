import logging
from decimal import Decimal
from typing import Optional

import asyncpg

from app.models.ab_test import ABTestResponse, VariantStats

logger = logging.getLogger(__name__)


async def create_ab_test(conn: asyncpg.Connection, suggestion_id: str, strategy_id: str, min_sample_size: int) -> dict:
    row = await conn.fetchrow(
        """
        INSERT INTO optimizer_ab_tests (suggestion_id, strategy_id, min_sample_size)
        VALUES ($1::uuid, $2, $3)
        RETURNING *
        """,
        suggestion_id, strategy_id, min_sample_size,
    )
    return _row_to_ab_test(row)


async def get_ab_test(conn: asyncpg.Connection, ab_test_id: str) -> Optional[dict]:
    row = await conn.fetchrow(
        "SELECT * FROM optimizer_ab_tests WHERE id = $1::uuid",
        ab_test_id,
    )
    if row is None:
        return None
    return _row_to_ab_test(row)


async def get_ab_test_by_suggestion(conn: asyncpg.Connection, suggestion_id: str) -> Optional[dict]:
    row = await conn.fetchrow(
        "SELECT * FROM optimizer_ab_tests WHERE suggestion_id = $1::uuid ORDER BY created_at DESC LIMIT 1",
        suggestion_id,
    )
    if row is None:
        return None
    return _row_to_ab_test(row)


async def insert_ab_result(
    conn: asyncpg.Connection,
    ab_test_id: str,
    variant: str,
    market_id: str,
    side: str,
    entry_price: Decimal,
    exit_price: Optional[Decimal],
    quantity: Decimal,
    pnl: Optional[Decimal],
) -> str:
    row = await conn.fetchrow(
        """
        INSERT INTO optimizer_ab_test_results (ab_test_id, variant, market_id, side, entry_price, exit_price, quantity, pnl)
        VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8)
        RETURNING id
        """,
        ab_test_id, variant, market_id, side, entry_price, exit_price, quantity, pnl,
    )
    return str(row["id"])


async def increment_sample_size(conn: asyncpg.Connection, ab_test_id: str) -> int:
    row = await conn.fetchrow(
        """
        UPDATE optimizer_ab_tests
        SET current_sample_size = current_sample_size + 1
        WHERE id = $1::uuid
        RETURNING current_sample_size
        """,
        ab_test_id,
    )
    return row["current_sample_size"] if row else 0


async def complete_ab_test(
    conn: asyncpg.Connection,
    ab_test_id: str,
    p_value: float,
    mean_difference: Decimal,
    recommendation: str,
) -> Optional[dict]:
    row = await conn.fetchrow(
        """
        UPDATE optimizer_ab_tests
        SET status = 'completed',
            p_value = $2,
            mean_difference = $3,
            recommendation = $4,
            completed_at = NOW()
        WHERE id = $1::uuid AND status = 'running'
        RETURNING *
        """,
        ab_test_id, p_value, mean_difference, recommendation,
    )
    if row is None:
        return None
    return _row_to_ab_test(row)


async def fail_ab_test(conn: asyncpg.Connection, ab_test_id: str, reason: str) -> Optional[dict]:
    row = await conn.fetchrow(
        """
        UPDATE optimizer_ab_tests
        SET status = 'failed',
            failed_reason = $2,
            completed_at = NOW()
        WHERE id = $1::uuid AND status = 'running'
        RETURNING *
        """,
        ab_test_id, reason,
    )
    if row is None:
        return None
    return _row_to_ab_test(row)


async def get_variant_pnls(conn: asyncpg.Connection, ab_test_id: str, variant: str) -> list[Decimal]:
    rows = await conn.fetch(
        """
        SELECT pnl FROM optimizer_ab_test_results
        WHERE ab_test_id = $1::uuid AND variant = $2 AND pnl IS NOT NULL
        ORDER BY simulated_at
        """,
        ab_test_id, variant,
    )
    return [row["pnl"] for row in rows]


async def get_variant_stats(conn: asyncpg.Connection, ab_test_id: str, variant: str) -> VariantStats:
    row = await conn.fetchrow(
        """
        SELECT
            COUNT(*) as cnt,
            COALESCE(AVG(pnl), 0) as mean_pnl,
            COALESCE(STDDEV(pnl), 0) as std_pnl
        FROM optimizer_ab_test_results
        WHERE ab_test_id = $1::uuid AND variant = $2 AND pnl IS NOT NULL
        """,
        ab_test_id, variant,
    )
    return VariantStats(
        count=row["cnt"],
        mean_pnl=Decimal(str(row["mean_pnl"])),
        std_pnl=Decimal(str(row["std_pnl"])),
    )


def _row_to_ab_test(row: asyncpg.Record) -> dict:
    return {
        "id": str(row["id"]),
        "suggestion_id": str(row["suggestion_id"]),
        "strategy_id": row["strategy_id"],
        "status": row["status"],
        "min_sample_size": row["min_sample_size"],
        "current_sample_size": row["current_sample_size"],
        "p_value": float(row["p_value"]) if row["p_value"] is not None else None,
        "mean_difference": float(row["mean_difference"]) if row["mean_difference"] is not None else None,
        "recommendation": row["recommendation"],
        "started_at": row["started_at"],
        "completed_at": row.get("completed_at"),
        "failed_reason": row.get("failed_reason"),
        "created_at": row["created_at"],
    }
