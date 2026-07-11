import time
from datetime import datetime, timezone

import httpx
from fastapi import APIRouter, Depends, Query

from app.config import config
from app.middleware.auth import verify_jwt
from app.models.portfolio import PortfolioOverview, Position
from app.models.risk_limits import AccountPortfolioSummary, CrossAccountPortfolioResponse
from app.metrics import PORTFOLIO_QUERY_LATENCY, PORTFOLIO_QUERY_TOTAL, POSITION_QUERY_LATENCY, POSITION_QUERY_TOTAL

router = APIRouter(prefix="/api", tags=["portfolio"])

HTTP_TIMEOUT = httpx.Timeout(10.0, connect=5.0)


@router.get("/portfolio/overview", response_model=PortfolioOverview)
async def get_portfolio_overview(_user: dict = Depends(verify_jwt)):
    start = time.monotonic()

    try:
        async with httpx.AsyncClient(timeout=HTTP_TIMEOUT) as client:
            resp = await client.get(f"{config.PORTFOLIO_SERVICE_URL}/portfolio/overview")
            resp.raise_for_status()
            overview = PortfolioOverview(**resp.json())
    except httpx.HTTPError:
        overview = PortfolioOverview(
            totalCapital="0.00000000",
            dailyPnL="0.00000000",
            totalPnL="0.00000000",
            utilizationRate="0.0000",
            lastUpdated="1970-01-01T00:00:00Z",
        )

    elapsed = (time.monotonic() - start) * 1000
    PORTFOLIO_QUERY_LATENCY.observe(elapsed)
    PORTFOLIO_QUERY_TOTAL.inc()

    return overview


@router.get("/positions", response_model=list[Position])
async def get_positions(_user: dict = Depends(verify_jwt)):
    start = time.monotonic()

    try:
        async with httpx.AsyncClient(timeout=HTTP_TIMEOUT) as client:
            resp = await client.get(f"{config.POSITION_SERVICE_URL}/positions")
            resp.raise_for_status()
            positions = [Position(**p) for p in resp.json()]
    except httpx.HTTPError:
        positions = []

    elapsed = (time.monotonic() - start) * 1000
    POSITION_QUERY_LATENCY.observe(elapsed)
    POSITION_QUERY_TOTAL.inc()

    return positions


@router.get("/portfolio/cross-account", response_model=CrossAccountPortfolioResponse)
async def get_cross_account_portfolio(_user: dict = Depends(verify_jwt)):
    """Get cross-account portfolio overview."""
    start = time.monotonic()

    try:
        async with httpx.AsyncClient(timeout=HTTP_TIMEOUT) as client:
            resp = await client.get(f"{config.PORTFOLIO_SERVICE_URL}/portfolio/cross-account")
            resp.raise_for_status()
            data = resp.json()
            result = CrossAccountPortfolioResponse(**data)
    except httpx.HTTPError:
        result = CrossAccountPortfolioResponse(
            total_capital="0.00000000",
            total_daily_pnl="0.00000000",
            total_pnl="0.00000000",
            total_positions=0,
            accounts=[],
            last_updated=datetime.now(timezone.utc),
        )

    elapsed = (time.monotonic() - start) * 1000
    PORTFOLIO_QUERY_LATENCY.observe(elapsed)
    PORTFOLIO_QUERY_TOTAL.inc()

    return result


@router.get("/portfolio/accounts", response_model=list[AccountPortfolioSummary])
async def get_account_portfolio_summaries(_user: dict = Depends(verify_jwt)):
    """Get per-account portfolio summaries."""
    start = time.monotonic()

    try:
        async with httpx.AsyncClient(timeout=HTTP_TIMEOUT) as client:
            resp = await client.get(f"{config.PORTFOLIO_SERVICE_URL}/portfolio/accounts")
            resp.raise_for_status()
            result = [AccountPortfolioSummary(**a) for a in resp.json()]
    except httpx.HTTPError:
        result = []

    elapsed = (time.monotonic() - start) * 1000
    PORTFOLIO_QUERY_LATENCY.observe(elapsed)
    PORTFOLIO_QUERY_TOTAL.inc()

    return result
