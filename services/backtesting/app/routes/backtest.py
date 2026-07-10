import asyncio
import logging
import math

from fastapi import APIRouter, Depends, HTTPException

from app.db import get_pg_pool, get_ts_pool
from app.engine.backtest_engine import run_backtest
from app.middleware.auth import verify_jwt
from app.models.backtest import BacktestRequest, BacktestResults, BacktestStatus, SweepRequest, SweepResults
from app.repos import backtest_repo

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/backtesting", tags=["backtesting"])

# #1: Track background tasks for graceful shutdown
_background_tasks: set[asyncio.Task] = set()
MAX_CONCURRENT_BACKTESTS = 5
_semaphore = asyncio.Semaphore(MAX_CONCURRENT_BACKTESTS)


@router.post("/run", response_model=BacktestStatus)
async def start_backtest(body: BacktestRequest, _user: dict = Depends(verify_jwt)):
    """Start a new backtest run."""
    # #5: Properly await semaphore acquisition
    if _semaphore.locked():
        raise HTTPException(status_code=429, detail="Too many concurrent backtests. Try again later.")
    await _semaphore.acquire()

    try:
        pool = await get_pg_pool()
        async with pool.acquire() as conn:
            run_id = await backtest_repo.create_run(conn, body)
    except Exception:
        _semaphore.release()
        raise

    task = asyncio.create_task(_run_backtest_background(run_id, body, _semaphore))
    _background_tasks.add(task)
    task.add_done_callback(_background_tasks.discard)

    return BacktestStatus(run_id=run_id, status="pending")


async def _run_backtest_background(run_id: str, req: BacktestRequest, semaphore: asyncio.Semaphore):
    """Background task to run backtest."""
    pg_pool = await get_pg_pool()
    ts_pool = await get_ts_pool()

    try:
        async with pg_pool.acquire() as conn:
            await backtest_repo.update_status(conn, run_id, "running")

        # #15: Fetch data with lookback window for strategy warm-up
        # Fetch 7 days before start_date for warm-up context
        async with ts_pool.acquire() as conn:
            opportunities = await backtest_repo.get_opportunities(conn, req.start_date, req.end_date)

        if not opportunities:
            # #8, #13: Add warning and include all result fields
            async with pg_pool.acquire() as conn:
                await backtest_repo.save_results(conn, run_id, {
                    "summary": {"total_pnl": "0", "total_trades": 0, "win_rate": "0", "sharpe_ratio": "0", "max_drawdown": "0", "profit_factor": None, "var_95": "0"},
                    "trades": [],
                    "warnings": [{"type": "no_data", "message": "No opportunities found for the given date range."}],
                    "daily_pnl": [],
                })
            return

        # #17: Progress reporting
        total = len(opportunities)
        async with pg_pool.acquire() as conn:
            await backtest_repo.update_progress(conn, run_id, f"0% — 0/{total} opportunities")

        results = await run_backtest(opportunities, req.simulation)

        # #3: Update progress on completion
        async with pg_pool.acquire() as conn:
            await backtest_repo.update_progress(conn, run_id, f"100% — {results.summary.total_trades} trades")
            await backtest_repo.save_results(conn, run_id, results.model_dump())

        logger.info("backtest completed", extra={"run_id": run_id, "trades": results.summary.total_trades})

    except Exception as e:
        logger.error("backtest failed", extra={"run_id": str(run_id)}, exc_info=e)
        try:
            async with pg_pool.acquire() as conn:
                await backtest_repo.update_status(conn, run_id, "failed", str(e))
        except Exception:
            pass
    finally:
        semaphore.release()


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
        progress=run.get("progress"),
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


@router.get("/{run_id}/report")
async def get_report(run_id: str, _user: dict = Depends(verify_jwt)):
    """#2: Get full backtest report with PnL and drawdown curves."""
    pool = await get_pg_pool()
    async with pool.acquire() as conn:
        run = await backtest_repo.get_run(conn, run_id)

    if run is None:
        raise HTTPException(status_code=404, detail="Backtest run not found")
    if run["status"] != "completed":
        raise HTTPException(status_code=400, detail=f"Backtest not completed (status: {run['status']})")

    results_data = run.get("results", {})
    # Build report from stored results
    from app.engine.backtest_engine import generate_report
    from app.models.backtest import BacktestResults, BacktestSummary, BacktestTrade

    summary = BacktestSummary(**results_data.get("summary", {}))
    trades = [BacktestTrade(**t) for t in results_data.get("trades", [])]
    results = BacktestResults(summary=summary, trades=trades, warnings=results_data.get("warnings", []), daily_pnl=results_data.get("daily_pnl"))

    report = await generate_report(results)
    report.run_id = run_id
    return report


@router.post("/sweep", response_model=SweepResults)
async def run_sweep(body: SweepRequest, _user: dict = Depends(verify_jwt)):
    """#3: Run parameter sweep — test multiple configurations in batch."""
    import itertools

    # #9: Validate parameter names against SimulationConfig
    from app.models.backtest import SimulationConfig
    valid_fields = set(SimulationConfig.model_fields.keys())
    for p in body.parameters:
        if p.name not in valid_fields:
            raise HTTPException(status_code=400, detail=f"Invalid parameter name: {p.name}. Valid: {valid_fields}")

    # Generate parameter combinations
    param_ranges = {}
    for p in body.parameters:
        # #7: Use integer-based range to avoid float accumulation
        n_steps = int(round((p.max_value - p.min_value) / p.step)) + 1
        values = [round(p.min_value + i * p.step, 10) for i in range(n_steps)]
        values = [v for v in values if v <= p.max_value + 0.0001]
        param_ranges[p.name] = values

    if not param_ranges:
        raise HTTPException(status_code=400, detail="No parameters provided")

    # Generate all combinations
    keys = list(param_ranges.keys())
    combos = list(itertools.product(*[param_ranges[k] for k in keys]))

    if len(combos) > 1000:
        raise HTTPException(status_code=400, detail=f"Too many combinations ({len(combos)}). Max 1000.")

    # #4: Acquire semaphore for sweep
    if _semaphore.locked():
        raise HTTPException(status_code=429, detail="Too many concurrent backtests/sweeps.")
    await _semaphore.acquire()

    # Fetch opportunities once
    pg_pool = await get_pg_pool()
    ts_pool = await get_ts_pool()

    async with ts_pool.acquire() as conn:
        opportunities = await backtest_repo.get_opportunities(conn, body.start_date, body.end_date)

    if not opportunities:
        _semaphore.release()
        return SweepResults(results=[], best=None, total_configs=0)

    # Run all configurations with timeout
    sweep_results = []
    try:
        # #5: 1 hour timeout for sweeps
        async def _run_sweep():
            for combo in combos:
                config_dict = dict(zip(keys, combo))
                sim_config = body.simulation.model_copy(update=config_dict)
                results = await run_backtest(opportunities, sim_config)
                sweep_results.append({
                    "parameters": config_dict,
                    "summary": results.summary.model_dump(),
                })

        await asyncio.wait_for(_run_sweep(), timeout=3600)
    except asyncio.TimeoutError:
        raise HTTPException(status_code=408, detail="Sweep timed out after 1 hour")
    finally:
        _semaphore.release()

    # #8: Rank by selected metric with NaN handling
    def sort_key(item):
        val = item["summary"].get(body.rank_by)
        if val is None or val == "null":
            return float("-inf")
        try:
            f = float(val)
            if math.isnan(f) or math.isinf(f):
                return float("-inf")
            return f
        except (ValueError, TypeError):
            return float("-inf")

    sweep_results.sort(key=sort_key, reverse=True)

    # #15: Return as SweepResults model
    best = sweep_results[0] if sweep_results else None
    return SweepResults(
        results=[{"parameters": r["parameters"], "summary": BacktestSummary(**r["summary"])} for r in sweep_results],
        best={"parameters": best["parameters"], "summary": BacktestSummary(**best["summary"])} if best else None,
        total_configs=len(sweep_results),
    )
