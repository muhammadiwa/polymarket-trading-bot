import csv
import io
import logging
from datetime import datetime, timezone
from datetime import timedelta
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, Query
from fastapi.responses import StreamingResponse

from app.db import get_pool
from app.middleware.auth import verify_jwt
from app.models.analytics import AnalyticsSummary, AnomalyEvent, PnLResponse, PerformanceMetrics, RiskMetrics
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
        # #4: Use single connection for consistency across all calculations
        async with conn.transaction():
            pnl = await analytics_repo.calculate_pnl(conn, start, end, "day", strategy_id, market_id)
            perf = await analytics_repo.calculate_performance_metrics(conn, start, end, strategy_id, market_id)
            risk = await analytics_repo.calculate_risk_metrics(conn, start, end, strategy_id, market_id)

    return AnalyticsSummary(
        pnl=PnLResponse(**pnl),
        performance=PerformanceMetrics(**perf),
        risk=RiskMetrics(**risk),
        date_range={"start": start.isoformat(), "end": end.isoformat()},
    )


@router.get("/histogram")
async def get_histogram(
    start_date: str = Query(..., description="Start date (ISO 8601)"),
    end_date: str = Query(..., description="End date (ISO 8601)"),
    strategy_id: Optional[str] = Query(None),
    market_id: Optional[str] = Query(None),
    bins: int = Query(20, ge=5, le=100, description="Number of histogram bins"),
    _user: dict = Depends(verify_jwt),
):
    """Return raw PnL values for frontend histogram binning."""
    start = _parse_date(start_date, "start_date")
    end = _parse_date(end_date, "end_date")
    if start >= end:
        raise HTTPException(status_code=400, detail="start_date must be before end_date")

    pool = await get_pool()
    async with pool.acquire() as conn:
        trades = await analytics_repo.get_trades_in_range(conn, start, end, strategy_id, market_id)

    pnls = [str(t["pnl"]) for t in trades]  # #3: Keep as Decimal strings
    return {"pnls": pnls, "count": len(pnls), "bins": bins}


@router.get("/export")
async def export_trades(
    start_date: str = Query(..., description="Start date (ISO 8601)"),
    end_date: str = Query(..., description="End date (ISO 8601)"),
    strategy_id: Optional[str] = Query(None),
    market_id: Optional[str] = Query(None),
    side: Optional[str] = Query(None, pattern="^(YES|NO)$"),
    pnl_sign: Optional[str] = Query(None, pattern="^(positive|negative|zero)$"),
    _user: dict = Depends(verify_jwt),
):
    """Stream trades as CSV download."""
    start = _parse_date(start_date, "start_date")
    end = _parse_date(end_date, "end_date")
    if start >= end:
        raise HTTPException(status_code=400, detail="start_date must be before end_date")

    # Cap date range at 365 days to prevent memory exhaustion
    if (end - start) > timedelta(days=365):
        raise HTTPException(status_code=400, detail="Date range cannot exceed 365 days")

    pool = await get_pool()

    async def generate():
        header = "timestamp,market_id,market_slug,strategy_id,side,price,quantity,filled_quantity,pnl,fee,slippage_pct,fill_status,latency_ms\n"
        yield header

        async with pool.acquire() as conn:
            trades = await analytics_repo.get_trades_in_range(conn, start, end, strategy_id, market_id, side, pnl_sign)
            for t in trades:
                ts = t["fill_timestamp"].isoformat() if t.get("fill_timestamp") else ""
                row = ",".join([
                    _csv_escape(_safe_str(ts)),
                    _csv_escape(_safe_str(t.get("market_id"))),
                    _csv_escape(_safe_str(t.get("market_slug"))),
                    _csv_escape(_safe_str(t.get("strategy_id"))),
                    _csv_escape(_safe_str(t.get("side"))),
                    _csv_escape(_safe_str(t.get("price"))),       # Escape all fields
                    _csv_escape(_safe_str(t.get("quantity"))),
                    _csv_escape(_safe_str(t.get("filled_quantity"))),
                    _csv_escape(_safe_str(t.get("pnl"))),
                    _csv_escape(_safe_str(t.get("fee"))),
                    _csv_escape(_safe_str(t.get("slippage_pct"))),
                    _csv_escape(_safe_str(t.get("fill_status"))),
                    _csv_escape(_safe_str(t.get("latency_ms"))),
                ])
                yield row + "\n"

    return StreamingResponse(
        generate(),
        media_type="text/csv",
        headers={"Content-Disposition": "attachment; filename=trades_export.csv"},
    )


@router.get("/export/json")
async def export_trades_json(
    start_date: str = Query(..., description="Start date (ISO 8601)"),
    end_date: str = Query(..., description="End date (ISO 8601)"),
    strategy_id: Optional[str] = Query(None),
    market_id: Optional[str] = Query(None),
    side: Optional[str] = Query(None, pattern="^(YES|NO)$"),
    pnl_sign: Optional[str] = Query(None, pattern="^(positive|negative|zero)$"),
    _user: dict = Depends(verify_jwt),
):
    """Export trades as JSON array."""
    start = _parse_date(start_date, "start_date")
    end = _parse_date(end_date, "end_date")
    if start >= end:
        raise HTTPException(status_code=400, detail="start_date must be before end_date")

    # Cap date range at 365 days to prevent memory exhaustion
    if (end - start) > timedelta(days=365):
        raise HTTPException(status_code=400, detail="Date range cannot exceed 365 days")

    pool = await get_pool()
    async with pool.acquire() as conn:
        trades = await analytics_repo.get_trades_in_range(conn, start, end, strategy_id, market_id, side, pnl_sign)

    result = []
    for t in trades:
        result.append({
            "timestamp": t["fill_timestamp"].isoformat() if t.get("fill_timestamp") else None,
            "market_id": t.get("market_id"),
            "market_slug": t.get("market_slug"),
            "strategy_id": t.get("strategy_id"),
            "side": t.get("side"),
            "price": _safe_str(t.get("price")),
            "quantity": _safe_str(t.get("quantity")),
            "filled_quantity": _safe_str(t.get("filled_quantity")),
            "pnl": _safe_str(t.get("pnl")),
            "fee": _safe_str(t.get("fee")),
            "slippage_pct": _safe_str(t.get("slippage_pct")),
            "fill_status": t.get("fill_status"),
            "latency_ms": t.get("latency_ms"),
        })

    return {"trades": result, "count": len(result)}


def _csv_escape(value: str) -> str:
    """Escape CSV field: wrap in quotes if contains comma, quote, newline, or carriage return.
    Also neutralize formula injection (=, +, -, @ prefix)."""
    if value is None:
        return ""
    if not value:
        return ""
    # #2: Neutralize CSV formula injection
    if value and value[0] in ("=", "+", "-", "@", "\t", "\r"):
        value = "'" + value
    if "," in value or '"' in value or "\n" in value or "\r" in value:
        return '"' + value.replace('"', '""') + '"'
    return value


def _safe_str(val) -> str:
    """Convert value to string, handling None."""
    if val is None:
        return ""
    return str(val)


@router.get("/anomalies", response_model=list[AnomalyEvent])
async def get_anomalies(
    severity: Optional[str] = Query(None, pattern="^(low|medium|high|critical)$"),
    limit: int = Query(50, ge=1, le=200),
    _user: dict = Depends(verify_jwt),
):
    """Get recent anomaly events."""
    pool = await get_pool()
    async with pool.acquire() as conn:
        anomalies = await analytics_repo.get_anomalies(conn, limit, severity)
    return [AnomalyEvent(**a) for a in anomalies]
