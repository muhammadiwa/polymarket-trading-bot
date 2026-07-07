import json
import logging
from datetime import datetime, timezone
from typing import Optional

import asyncpg

from app.models.backtest import BacktestRequest, BacktestStatus

logger = logging.getLogger(__name__)


async def create_run(conn: asyncpg.Connection, req: BacktestRequest) -> str:
    row = await conn.fetchrow(
        """
        INSERT INTO backtest_runs (strategy_id, start_date, end_date, status, config)
        VALUES ($1, $2, $3, 'pending', $4::jsonb)
        RETURNING id
        """,
        req.strategy_id, req.start_date, req.end_date,
        json.dumps(req.simulation.model_dump()),
    )
    return str(row["id"])


async def update_status(conn: asyncpg.Connection, run_id: str, status: str, error_message: str = None):
    await conn.execute(
        "UPDATE backtest_runs SET status = $1, error_message = $2, started_at = CASE WHEN $1 = 'running' THEN NOW() ELSE started_at END, completed_at = CASE WHEN $1 IN ('completed', 'failed') THEN NOW() ELSE completed_at END WHERE id = $3::uuid",
        status, error_message, run_id,
    )


async def save_results(conn: asyncpg.Connection, run_id: str, results: dict):
    await conn.execute(
        "UPDATE backtest_runs SET status = 'completed', results = $1::jsonb, completed_at = NOW() WHERE id = $2::uuid",
        json.dumps(results), run_id,
    )


async def get_run(conn: asyncpg.Connection, run_id: str) -> Optional[dict]:
    row = await conn.fetchrow("SELECT * FROM backtest_runs WHERE id = $1::uuid", run_id)
    if row is None:
        return None
    return dict(row)


async def get_opportunities(conn: asyncpg.Connection, start_date: str, end_date: str) -> list[dict]:
    rows = await conn.fetch(
        """
        SELECT market_id, spread, score, fill_probability, liquidity, side, detected_at, filter_reason
        FROM opportunities
        WHERE detected_at BETWEEN $1 AND $2
        ORDER BY detected_at ASC
        """,
        start_date, end_date,
    )
    return [dict(r) for r in rows]
