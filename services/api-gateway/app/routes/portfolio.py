import time

import httpx
from fastapi import APIRouter, Depends

from app.config import config
from app.middleware.auth import verify_jwt
from app.models.portfolio import PortfolioOverview, Position
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
