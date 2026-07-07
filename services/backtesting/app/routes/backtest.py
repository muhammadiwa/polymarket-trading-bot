import asyncio
import logging
from datetime import datetime, timezone

from fastapi import APIRouter, Depends, HTTPException

from app.db import get_pg_pool, get_ts_pool
from app.engine.backtest_engine import run_backtest
from app.middleware.auth import verify_jwt
from app.models.backtest import BacktestRequest, BacktestResults, BacktestStatus
from app.repos import backtest_repo

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/backtesting", tags=["backtesting"])


@router.post("/run", response_model=BacktestStatus)
async def start_backtest(body: BacktestRequest, _user: dict = Depends(verify_jwt)):
    """Start a new backtest run."""
    pool = await get_pg_pool()
    async with pool.acquire() as conn:
        run_id = await backtest_repo.create_run(conn, body)

    # Run backtest in background
    asyncio.create_task(_run_backtest_background(run_id, body))

    return BacktestStatus(run_id=run_id, status="pending")


async def _run_backtest_background(run_id: str, req: BacktestRequest):
    """Background task to run backtest."""
    try:
        pg_pool = await get_pg_pool()
        ts_pool = await get_ts_pool()

        # Update status to running
        async with pg_pool.acquire() as conn:
            await backtest_repo.update_status(conn, run_id, "running")

        # Fetch historical data from TimescaleDB
        async with ts_pool.acquire() as conn:
            opportunities = await backtest_repo.get_opportunities(conn, req.start_date, req.end_date)

        if not opportunities:
            async with pg_pool.acquire() as conn:
                await backtest_repo.save_results(conn, run_id, {
                    "summary": {"total_pnl": "0", "total_trades": 0, "win_rate": "0", "sharpe_ratio": "0", "max_drawdown": "0", "profit_factor": None},
                    "trades": [],
                    "warnings": [],
                })
            return

        # Run backtest
        results = await run_backtest(opportunities, req.simulation)

        # Save results
        async with pg_pool.acquire() as conn:
            await backtest_repo.save_results(conn, run_id, results.model_dump())

        logger.info("backtest completed", extra={"run_id": run_id, "trades": results.summary.total_trades})

    except Exception as e:
        logger.error("backtest failed", extra={"run_id": str(run_id)}, exc_info=e)
        try:
            async with pg_pool.acquire() as conn:
                await backtest_repo.update_status(conn, run_id, "failed", str(e))
        except Exception:
            pass


@router.get("/{run_id}/status", response_model=BacktestStatus)
async def get_status(run_id: str, _user: dict = Depends(verify_jwt)):
    """Get backtest run status."""
    pool = await get_pg_pool()
    async with pool.acquire() as conn:
        run = await backtest_repo.get_run(conn, run_id)

    if run is None:
        raise HTTPException(status_code=404, detail="Backtest run not found")

    return BacktestStatus(
        run_id=str(run["id"]),
        status=run["status"],
        started_at=run.get("started_at"),
        completed_at=run.get("completed_at"),
        error_message=run.get("error_message"),
    )


@router.get("/{run_id}/results")
async def get_results(run_id: str, _user: dict = Depends(verify_jwt)):
    """Get backtest results."""
    pool = await get_pg_pool()
    async with pool.acquire() as conn:
        run = await backtest_repo.get_run(conn, run_id)

    if run is None:
        raise HTTPException(status_code=404, detail="Backtest run not found")

    if run["status"] != "completed":
        raise HTTPException(status_code=400, detail=f"Backtest not completed (status: {run['status']})")

    return {"run_id": str(run["id"]), "status": run["status"], "results": run.get("results")}
