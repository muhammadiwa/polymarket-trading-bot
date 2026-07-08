import logging
from decimal import Decimal

import asyncpg
from fastapi import APIRouter, Depends, HTTPException

from app.db import get_pool
from app.engine.ab_tester import calculate_significance, simulate_trade_outcome
from app.engine.overfitting_detector import OVERFITTING_THRESHOLD
from app.middleware.auth import verify_jwt
from app.models.ab_test import (
    ABTestResponse,
    ABTestResultSummary,
    OverfittingAnalysisResponse,
    StartABTestRequest,
)
from app.repos import ab_test_repo, optimizer_repo

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/optimizer", tags=["ab-test"])


@router.post("/suggestions/{suggestion_id}/start-ab-test", response_model=ABTestResponse)
async def start_ab_test(
    suggestion_id: str,
    request: StartABTestRequest = StartABTestRequest(),
    user: dict = Depends(verify_jwt),
):
    """Start an A/B test for an approved suggestion. Runs synchronously using historical data."""
    pool = await get_pool()
    async with pool.acquire() as conn:
        suggestion = await optimizer_repo.get_suggestion_by_id(conn, suggestion_id)

        if suggestion is None:
            raise HTTPException(status_code=404, detail="Suggestion not found")

        if suggestion["status"] != "approved":
            raise HTTPException(
                status_code=400,
                detail=f"Cannot start A/B test: suggestion status is '{suggestion['status']}', expected 'approved'",
            )

        existing = await ab_test_repo.get_ab_test_by_suggestion(conn, suggestion_id)
        if existing and existing["status"] == "running":
            raise HTTPException(
                status_code=409,
                detail=f"A/B test already running: {existing['id']}",
            )

        # Create A/B test record
        ab_test = await ab_test_repo.create_ab_test(
            conn,
            suggestion_id=suggestion_id,
            strategy_id=suggestion["strategy_id"],
            min_sample_size=request.min_sample_size,
        )

        # Fetch historical trades for synchronous execution
        trades = await optimizer_repo.get_trades(conn, suggestion["strategy_id"])

    # Run A/B test synchronously using historical data
    try:
        result = await _run_ab_test_sync(
            pool=pool,
            ab_test_id=ab_test["id"],
            trades=trades,
            suggestion=suggestion,
            min_sample_size=request.min_sample_size,
        )

        logger.info(
            "ab test completed",
            extra={
                "ab_test_id": ab_test["id"],
                "suggestion_id": suggestion_id,
                "recommendation": result["recommendation"],
                "p_value": result["p_value"],
                "user": user.get("username"),
            },
        )

        # Refresh ab_test with results
        async with pool.acquire() as conn:
            ab_test = await ab_test_repo.get_ab_test(conn, ab_test["id"])

    except asyncpg.UniqueViolationError:
        # Race condition: another request created a running test
        raise HTTPException(status_code=409, detail="A/B test already running for this suggestion")

    except ValueError as e:
        # Insufficient data - mark as failed
        async with pool.acquire() as conn:
            await ab_test_repo.fail_ab_test(conn, ab_test["id"], str(e))
            ab_test = await ab_test_repo.get_ab_test(conn, ab_test["id"])
        logger.warning("ab test failed: insufficient data", extra={"ab_test_id": ab_test["id"], "error": str(e)})

    except Exception as e:
        async with pool.acquire() as conn:
            await ab_test_repo.fail_ab_test(conn, ab_test["id"], str(e))
            ab_test = await ab_test_repo.get_ab_test(conn, ab_test["id"])
        logger.error("ab test failed", extra={"ab_test_id": ab_test["id"], "error": str(e)})

    return ABTestResponse(**ab_test)


async def _run_ab_test_sync(
    pool,
    ab_test_id: str,
    trades: list[dict],
    suggestion: dict,
    min_sample_size: int,
) -> dict:
    """Run A/B test synchronously using historical trades."""
    from datetime import datetime, timezone

    # Split trades chronologically for control/treatment simulation
    # Use first half as control (current strategy), second half as treatment (suggested strategy)
    mid_idx = len(trades) // 2
    control_trades = trades[:mid_idx] if mid_idx >= 10 else trades[:max(10, len(trades) // 3)]
    treatment_trades = trades[mid_idx:] if len(trades) - mid_idx >= 10 else trades[max(10, len(trades) // 3):]

    # Ensure minimum samples
    if len(control_trades) < 10 or len(treatment_trades) < 10:
        raise ValueError(
            f"Insufficient historical trades: control={len(control_trades)}, treatment={len(treatment_trades)}, need 10+ each"
        )

    control_pnls = []
    treatment_pnls = []

    async with pool.acquire() as conn:
        # Record control trades (simulated with current strategy parameters)
        for trade in control_trades:
            pnl = _simulate_pnl(trade)
            if pnl is not None:
                control_pnls.append(pnl)
                await ab_test_repo.insert_ab_result(
                    conn=conn,
                    ab_test_id=ab_test_id,
                    variant="control",
                    market_id=trade.get("market_id", "unknown"),
                    side=trade.get("side", "YES"),
                    entry_price=_safe_decimal(trade.get("entry_price", 0)),
                    exit_price=_safe_decimal(trade.get("exit_price")) if trade.get("exit_price") else None,
                    quantity=_safe_decimal(trade.get("quantity", 0)),
                    pnl=pnl,
                )

        # Record treatment trades (simulated with suggested strategy parameters)
        for trade in treatment_trades:
            pnl = _simulate_pnl(trade)
            if pnl is not None:
                treatment_pnls.append(pnl)
                await ab_test_repo.insert_ab_result(
                    conn=conn,
                    ab_test_id=ab_test_id,
                    variant="treatment",
                    market_id=trade.get("market_id", "unknown"),
                    side=trade.get("side", "YES"),
                    entry_price=_safe_decimal(trade.get("entry_price", 0)),
                    exit_price=_safe_decimal(trade.get("exit_price")) if trade.get("exit_price") else None,
                    quantity=_safe_decimal(trade.get("quantity", 0)),
                    pnl=pnl,
                )

        # Update sample size
        total_samples = len(control_pnls) + len(treatment_pnls)
        if total_samples > 0:
            await conn.execute(
                "UPDATE optimizer_ab_tests SET current_sample_size = $1 WHERE id = $2::uuid",
                total_samples, ab_test_id,
            )

    # Calculate significance
    if len(control_pnls) < 10 or len(treatment_pnls) < 10:
        raise ValueError(
            f"Insufficient valid trades: control={len(control_pnls)}, treatment={len(treatment_pnls)}, need 10+ each"
        )

    summary = calculate_significance(control_pnls, treatment_pnls)

    # Complete the test
    async with pool.acquire() as conn:
        await ab_test_repo.complete_ab_test(
            conn=conn,
            ab_test_id=ab_test_id,
            p_value=summary.p_value,
            mean_difference=summary.mean_difference,
            recommendation=summary.recommendation,
        )

    return {
        "p_value": summary.p_value,
        "recommendation": summary.recommendation,
        "mean_difference": float(summary.mean_difference),
    }


def _simulate_pnl(trade: dict):
    """Extract PnL from a trade (already computed from historical data)."""
    pnl = trade.get("pnl")
    if pnl is None:
        return None
    try:
        return Decimal(str(pnl))
    except Exception:
        return None


def _safe_decimal(value) -> Decimal:
    """Safely convert value to Decimal."""
    if value is None:
        return Decimal("0")
    try:
        return Decimal(str(value))
    except Exception:
        return Decimal("0")


@router.get("/ab-tests/{ab_test_id}", response_model=ABTestResponse)
async def get_ab_test(
    ab_test_id: str,
    user: dict = Depends(verify_jwt),
):
    """Get A/B test status and results."""
    pool = await get_pool()
    async with pool.acquire() as conn:
        ab_test = await ab_test_repo.get_ab_test(conn, ab_test_id)

    if ab_test is None:
        raise HTTPException(status_code=404, detail="A/B test not found")

    return ABTestResponse(**ab_test)


@router.get("/ab-tests/{ab_test_id}/summary", response_model=ABTestResultSummary)
async def get_ab_test_summary(
    ab_test_id: str,
    user: dict = Depends(verify_jwt),
):
    """Get detailed A/B test result summary with statistical analysis."""
    pool = await get_pool()
    async with pool.acquire() as conn:
        ab_test = await ab_test_repo.get_ab_test(conn, ab_test_id)

        if ab_test is None:
            raise HTTPException(status_code=404, detail="A/B test not found")

        control_pnls = await ab_test_repo.get_variant_pnls(conn, ab_test_id, "control")
        treatment_pnls = await ab_test_repo.get_variant_pnls(conn, ab_test_id, "treatment")

    if len(control_pnls) < 10 or len(treatment_pnls) < 10:
        raise HTTPException(
            status_code=400,
            detail=f"Insufficient data: control={len(control_pnls)}, treatment={len(treatment_pnls)}, need 10+ each",
        )

    summary = calculate_significance(control_pnls, treatment_pnls)
    summary.ab_test_id = ab_test_id

    return summary


@router.get("/suggestions/{suggestion_id}/overfitting-analysis", response_model=OverfittingAnalysisResponse)
async def get_overfitting_analysis(
    suggestion_id: str,
    user: dict = Depends(verify_jwt),
):
    """Get overfitting analysis for a suggestion."""
    pool = await get_pool()
    async with pool.acquire() as conn:
        suggestion = await optimizer_repo.get_suggestion_by_id(conn, suggestion_id)

        if suggestion is None:
            raise HTTPException(status_code=404, detail="Suggestion not found")

    overfitting_score = suggestion.get("overfitting_score")
    degradation_pct = suggestion.get("degradation_pct")
    is_overfitting = degradation_pct is not None and float(degradation_pct) > (OVERFITTING_THRESHOLD * 100)

    warning = None
    if is_overfitting:
        warning = (
            f"Overfitting detected: out-of-sample performance degraded by {degradation_pct}% "
            f"compared to in-sample. This suggestion may not generalize to new data. "
            f"Recommendation: test with A/B test before deploying to live."
        )

    return OverfittingAnalysisResponse(
        suggestion_id=suggestion_id,
        overfitting_score=overfitting_score,
        in_sample_win_rate=suggestion.get("in_sample_win_rate"),
        out_of_sample_win_rate=suggestion.get("out_of_sample_win_rate"),
        degradation_pct=degradation_pct,
        is_overfitting=is_overfitting,
        warning=warning,
    )
