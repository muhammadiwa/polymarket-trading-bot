import logging
from typing import Optional
from uuid import UUID

from fastapi import APIRouter, Depends, HTTPException, Query

from app.db import get_pool
from app.middleware.auth import verify_jwt
from app.models.portfolio import PortfolioOverview, RebalanceRequest, RebalanceResponse, TierTransition
from app.repos import portfolio_repo

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/portfolio", tags=["portfolio"])


def _validate_uuid(value: Optional[str], field_name: str = "account_id") -> Optional[str]:
    if value is None:
        return None
    try:
        UUID(value)
    except ValueError:
        raise HTTPException(status_code=400, detail=f"Invalid {field_name} format")
    return value


@router.get("/overview", response_model=PortfolioOverview)
async def get_overview(
    account_id: Optional[str] = Query(None),
    _user: dict = Depends(verify_jwt),
):
    _validate_uuid(account_id)
    pool = await get_pool()
    async with pool.acquire() as conn:
        overview = await portfolio_repo.get_overview(conn, account_id)
    return overview


@router.put("/capital", response_model=PortfolioOverview)
async def update_capital(
    total_capital: float,
    deployed_capital: float,
    account_id: Optional[str] = Query(None),
    _user: dict = Depends(verify_jwt),
):
    if total_capital < 0:
        raise HTTPException(status_code=400, detail="Total capital cannot be negative")
    if deployed_capital < 0:
        raise HTTPException(status_code=400, detail="Deployed capital cannot be negative")
    if deployed_capital > total_capital:
        raise HTTPException(status_code=400, detail="Deployed capital cannot exceed total")

    _validate_uuid(account_id)
    pool = await get_pool()
    async with pool.acquire() as conn:
        overview = await portfolio_repo.update_capital(conn, total_capital, deployed_capital, account_id)
    return overview


@router.post("/rebalance", response_model=RebalanceResponse)
async def rebalance(
    body: RebalanceRequest,
    account_id: Optional[str] = Query(None),
    user: dict = Depends(verify_jwt),
):
    if not body.weights:
        raise HTTPException(status_code=400, detail="Weights cannot be empty")

    total = sum(body.weights.values())
    if abs(total - 100.0) > 0.01:
        raise HTTPException(status_code=400, detail=f"Weights must sum to 100% (±0.01%). Current: {total:.4f}%")

    pool = await get_pool()
    async with pool.acquire() as conn:
        old_weights = {}  # TODO: fetch from strategy-manager service
        await portfolio_repo.log_rebalance(conn, old_weights, body.weights, user.get("user_id"), account_id)

    logger.info("portfolio rebalance executed", extra={"weights": body.weights, "user": user.get("user_id")})

    return RebalanceResponse(
        old_weights=old_weights,
        new_weights=body.weights,
        updated_count=len(body.weights),
    )


@router.get("/tiers", response_model=list[TierTransition])
async def get_tier_transitions(
    account_id: Optional[str] = Query(None),
    limit: int = Query(50, ge=1, le=200),
    _user: dict = Depends(verify_jwt),
):
    _validate_uuid(account_id)
    pool = await get_pool()
    async with pool.acquire() as conn:
        transitions = await portfolio_repo.get_tier_transitions(conn, account_id, limit)
    return transitions
