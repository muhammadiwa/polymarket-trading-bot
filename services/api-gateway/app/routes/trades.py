import time
from datetime import datetime, timedelta, timezone
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, Query, status
from fastapi.responses import StreamingResponse

from app.config import config
from app.db import get_pool
from app.export.csv_exporter import get_csv_filename, stream_csv
from app.export.json_exporter import get_json_filename, stream_json
from app.metrics import (
    TRADE_EXPORT_DURATION,
    TRADE_EXPORT_ROWS,
    TRADE_EXPORT_TOTAL,
    TRADE_QUERY_LATENCY,
    TRADE_QUERY_TOTAL,
)
from app.middleware.auth import verify_jwt
from app.models.trade import (
    FillStatusEnum,
    TradeFilterParams,
    TradeListResponse,
    TradeResponse,
    TradeStatsResponse,
)
from app.repos import trade_repo

router = APIRouter(prefix="/api/v1/trades", tags=["trades"])

MAX_EXPORT_RANGE_DAYS = 90


def _parse_date_range(
    start_date: Optional[str],
    end_date: Optional[str],
) -> tuple[Optional[datetime], Optional[datetime]]:
    start_dt = None
    end_dt = None
    if start_date:
        try:
            start_dt = datetime.fromisoformat(start_date)
        except ValueError:
            raise HTTPException(status_code=400, detail="Invalid start_date format")
    if end_date:
        try:
            end_dt = datetime.fromisoformat(end_date)
        except ValueError:
            raise HTTPException(status_code=400, detail="Invalid end_date format")
    return start_dt, end_dt


@router.get("", response_model=TradeListResponse)
async def list_trades(
    start_date: Optional[str] = Query(None),
    end_date: Optional[str] = Query(None),
    market_id: Optional[str] = Query(None),
    strategy_id: Optional[str] = Query(None),
    side: Optional[str] = Query(None),
    pnl_sign: Optional[str] = Query(None),
    fill_status: Optional[str] = Query(None),
    page: int = Query(1, ge=1),
    page_size: int = Query(50, ge=1, le=100),
    cursor: Optional[str] = Query(None),
    _user: dict = Depends(verify_jwt),
):
    start_dt, end_dt = _parse_date_range(start_date, end_date)

    fill_status_enum = None
    if fill_status:
        try:
            fill_status_enum = FillStatusEnum(fill_status)
        except ValueError:
            raise HTTPException(status_code=400, detail="Invalid fill_status")

    filters = TradeFilterParams(
        start_date=start_dt,
        end_date=end_dt,
        market_id=market_id,
        strategy_id=strategy_id,
        side=side,
        pnl_sign=pnl_sign,
        fill_status=fill_status_enum,
        page=page,
        page_size=page_size,
        cursor=cursor,
    )

    start = time.monotonic()
    pool = await get_pool()
    async with pool.acquire() as conn:
        trades, total_count, next_cursor = await trade_repo.list_trades(conn, filters)

    elapsed = (time.monotonic() - start) * 1000
    TRADE_QUERY_LATENCY.observe(elapsed)
    TRADE_QUERY_TOTAL.inc()

    return TradeListResponse(
        trades=trades,
        total_count=total_count,
        page=page,
        page_size=page_size,
        next_cursor=next_cursor,
    )


@router.get("/stats", response_model=TradeStatsResponse)
async def get_trade_stats(
    start_date: Optional[str] = Query(None),
    end_date: Optional[str] = Query(None),
    market_id: Optional[str] = Query(None),
    strategy_id: Optional[str] = Query(None),
    side: Optional[str] = Query(None),
    _user: dict = Depends(verify_jwt),
):
    start_dt, end_dt = _parse_date_range(start_date, end_date)

    filters = TradeFilterParams(
        start_date=start_dt,
        end_date=end_dt,
        market_id=market_id,
        strategy_id=strategy_id,
        side=side,
    )

    start = time.monotonic()
    pool = await get_pool()
    async with pool.acquire() as conn:
        stats = await trade_repo.get_stats(conn, filters)

    elapsed = (time.monotonic() - start) * 1000
    TRADE_QUERY_LATENCY.observe(elapsed)
    TRADE_QUERY_TOTAL.inc()

    return stats


@router.get("/export")
async def export_trades(
    format: str = Query("csv", pattern="^(csv|json)$"),
    start_date: Optional[str] = Query(None),
    end_date: Optional[str] = Query(None),
    market_id: Optional[str] = Query(None),
    strategy_id: Optional[str] = Query(None),
    side: Optional[str] = Query(None),
    pnl_sign: Optional[str] = Query(None),
    fill_status: Optional[str] = Query(None),
    _user: dict = Depends(verify_jwt),
):
    start_dt, end_dt = _parse_date_range(start_date, end_date)

    if start_dt and end_dt and (end_dt - start_dt) > timedelta(days=MAX_EXPORT_RANGE_DAYS):
        raise HTTPException(
            status_code=400,
            detail=f"Export date range cannot exceed {MAX_EXPORT_RANGE_DAYS} days",
        )

    fill_status_enum = None
    if fill_status:
        try:
            fill_status_enum = FillStatusEnum(fill_status)
        except ValueError:
            raise HTTPException(status_code=400, detail="Invalid fill_status")

    filters = TradeFilterParams(
        start_date=start_dt,
        end_date=end_dt,
        market_id=market_id,
        strategy_id=strategy_id,
        side=side,
        pnl_sign=pnl_sign,
        fill_status=fill_status_enum,
    )

    start = time.monotonic()
    pool = await get_pool()

    if format == "csv":
        async def csv_stream():
            async with pool.acquire() as conn:
                async for trade in trade_repo.export_trades(conn, filters):
                    yield trade

        async def csv_response():
            row_count = 0
            yield _csv_header()
            async with pool.acquire() as conn:
                async for trade in trade_repo.export_trades(conn, filters):
                    row_count += 1
                    yield _csv_row(trade)
            TRADE_EXPORT_DURATION.observe((time.monotonic() - start) * 1000)
            TRADE_EXPORT_TOTAL.labels(format=format).inc()
            TRADE_EXPORT_ROWS.inc(row_count)

        return StreamingResponse(
            csv_response(),
            media_type="text/csv",
            headers={
                "Content-Disposition": f'attachment; filename="{get_csv_filename()}"'
            },
        )
    else:
        async def json_response():
            row_count = 0
            yield b"["
            first = True
            async with pool.acquire() as conn:
                async for trade in trade_repo.export_trades(conn, filters):
                    if not first:
                        yield b","
                    first = False
                    row_count += 1
                    yield _json_trade_bytes(trade)
            yield b"]"
            TRADE_EXPORT_DURATION.observe((time.monotonic() - start) * 1000)
            TRADE_EXPORT_TOTAL.labels(format=format).inc()
            TRADE_EXPORT_ROWS.inc(row_count)

        return StreamingResponse(
            json_response(),
            media_type="application/json",
            headers={
                "Content-Disposition": f'attachment; filename="{get_json_filename()}"'
            },
        )


def _csv_header() -> str:
    from app.export.csv_exporter import generate_csv_header
    return generate_csv_header()


def _csv_row(trade: TradeResponse) -> str:
    from app.export.csv_exporter import generate_csv_rows
    return generate_csv_rows([trade])


def _json_trade_bytes(trade: TradeResponse) -> bytes:
    from app.export.json_exporter import trade_to_bytes
    return trade_to_bytes(trade)


@router.get("/{trade_id}", response_model=TradeResponse)
async def get_trade(
    trade_id: str,
    _user: dict = Depends(verify_jwt),
):
    pool = await get_pool()
    async with pool.acquire() as conn:
        trade = await trade_repo.get_trade(conn, trade_id)

    if trade is None:
        raise HTTPException(status_code=404, detail="Trade not found")

    TRADE_QUERY_TOTAL.inc()
    return trade
