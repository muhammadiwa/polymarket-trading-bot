import logging
from datetime import datetime, timezone
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, Query

from app.db import get_pool
from app.middleware.auth import verify_jwt
from app.models.analytics import AnalyticsSummary, PnLResponse, PerformanceMetrics, RiskMetrics
from app.repos import analytics_repo

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/analytics", tags=["analytics"])


def _parse_date(date_str: str, field_name: str) -> datetime:
    """Parse ISO date string to datetime."""
    try:
        dt = datetime.fromisoformat(date_str)
        if dt.tzinfo is None:
            dt = dt.replace(tzinfo=timezone.utc)
        return dt
    except ValueError:
        raise HTTPException(status_code=400, detail=f"Invalid {field_name} format. Use ISO 8601.")


@router.get("/pnl", response_model=PnLResponse)
async def get_pnl(
    start_date: str = Query(..., description="Start date (ISO 8601)"),
    end_date: str = Query(..., description="End date (ISO 8601)"),
    group_by: str = Query("day", pattern="^(day|week|month)$"),
    strategy_id: Optional[str] = Query(None),
    market_id: Optional[str] = Query(None),
    _user: dict = Depends(verify_jwt),
):
    start = _parse_date(start_date, "start_date")
    end = _parse_date(end_date, "end_date")

    if start >= end:
        raise HTTPException(status_code=400, detail="start_date must be before end_date")

    pool = await get_pool()
    async with pool.acquire() as conn:
        result = await analytics_repo.calculate_pnl(conn, start, end, group_by, strategy_id, market_id)

    return PnLResponse(**result)


@router.get("/metrics", response_model=PerformanceMetrics)
async def get_metrics(
    start_date: str = Query(..., description="Start date (ISO 8601)"),
    end_date: str = Query(..., description="End date (ISO 8601)"),
    strategy_id: Optional[str] = Query(None),
    market_id: Optional[str] = Query(None),
    _user: dict = Depends(verify_jwt),
):
    start = _parse_date(start_date, "start_date")
    end = _parse_date(end_date, "end_date")

    if start >= end:
        raise HTTPException(status_code=400, detail="start_date must be before end_date")

    pool = await get_pool()
    async with pool.acquire() as conn:
        result = await analytics_repo.calculate_performance_metrics(conn, start, end, strategy_id, market_id)

    return PerformanceMetrics(**result)


@router.get("/risk", response_model=RiskMetrics)
async def get_risk(
    start_date: str = Query(..., description="Start date (ISO 8601)"),
    end_date: str = Query(..., description="End date (ISO 8601)"),
    strategy_id: Optional[str] = Query(None),
    market_id: Optional[str] = Query(None),
    _user: dict = Depends(verify_jwt),
):
    start = _parse_date(start_date, "start_date")
    end = _parse_date(end_date, "end_date")

    if start >= end:
        raise HTTPException(status_code=400, detail="start_date must be before end_date")

    pool = await get_pool()
    async with pool.acquire() as conn:
        result = await analytics_repo.calculate_risk_metrics(conn, start, end, strategy_id, market_id)

    return RiskMetrics(**result)


@router.get("/summary", response_model=AnalyticsSummary)
async def get_summary(
    start_date: str = Query(..., description="Start date (ISO 8601)"),
    end_date: str = Query(..., description="End date (ISO 8601)"),
    strategy_id: Optional[str] = Query(None),
    market_id: Optional[str] = Query(None),
    _user: dict = Depends(verify_jwt),
):
    start = _parse_date(start_date, "start_date")
    end = _parse_date(end_date, "end_date")

    if start >= end:
        raise HTTPException(status_code=400, detail="start_date must be before end_date")

    pool = await get_pool()
    async with pool.acquire() as conn:
        pnl = await analytics_repo.calculate_pnl(conn, start, end, "day", strategy_id, market_id)
        perf = await analytics_repo.calculate_performance_metrics(conn, start, end, strategy_id, market_id)
        risk = await analytics_repo.calculate_risk_metrics(conn, start, end, strategy_id, market_id)

    return AnalyticsSummary(
        pnl=PnLResponse(**pnl),
        performance=PerformanceMetrics(**perf),
        risk=RiskMetrics(**risk),
        date_range={"start": start_date, "end": end_date},
    )
