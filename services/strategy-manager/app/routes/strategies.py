import logging
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, Query, status

from app.db import get_pool
from app.events import StrategyEventPublisher
from app.middleware.auth import verify_jwt
from app.models.strategy import StrategyCreate, StrategyListResponse, StrategyResponse, StrategyUpdate
from app.repos import strategy_repo

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/strategies", tags=["strategies"])

# Will be set by main.py
event_publisher: StrategyEventPublisher = None


def set_event_publisher(publisher: StrategyEventPublisher):
    global event_publisher
    event_publisher = publisher


@router.post("", response_model=StrategyResponse, status_code=status.HTTP_201_CREATED)
async def create_strategy(body: StrategyCreate, _user: dict = Depends(verify_jwt)):
    pool = await get_pool()
    async with pool.acquire() as conn:
        strategy = await strategy_repo.create_strategy(conn, body)

    if event_publisher:
        await event_publisher.publish_strategy_updated(
            strategy.id, strategy.name, strategy.status, "created",
            {"min_profit_threshold": strategy.min_profit_threshold, "score_threshold": strategy.score_threshold},
        )

    return strategy


@router.get("", response_model=StrategyListResponse)
async def list_strategies(
    status_filter: Optional[str] = Query(None, alias="status", pattern="^(active|inactive|paused)$"),
    account_id: Optional[str] = Query(None),
    limit: int = Query(50, ge=1, le=200),
    offset: int = Query(0, ge=0),
    _user: dict = Depends(verify_jwt),
):
    pool = await get_pool()
    async with pool.acquire() as conn:
        items, total = await strategy_repo.list_strategies(conn, status_filter, account_id, limit, offset)

    return StrategyListResponse(items=items, total=total)


@router.get("/{strategy_id}", response_model=StrategyResponse)
async def get_strategy(strategy_id: str, _user: dict = Depends(verify_jwt)):
    pool = await get_pool()
    async with pool.acquire() as conn:
        strategy = await strategy_repo.get_strategy(conn, strategy_id)

    if strategy is None:
        raise HTTPException(status_code=404, detail="Strategy not found")
    return strategy


@router.put("/{strategy_id}", response_model=StrategyResponse)
async def update_strategy(strategy_id: str, body: StrategyUpdate, _user: dict = Depends(verify_jwt)):
    pool = await get_pool()
    async with pool.acquire() as conn:
        strategy = await strategy_repo.update_strategy(conn, strategy_id, body)

    if strategy is None:
        raise HTTPException(status_code=404, detail="Strategy not found")

    if event_publisher:
        await event_publisher.publish_strategy_updated(
            strategy.id, strategy.name, strategy.status, "updated",
            body.model_dump(exclude_unset=True),
        )

    return strategy


@router.delete("/{strategy_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_strategy(strategy_id: str, _user: dict = Depends(verify_jwt)):
    pool = await get_pool()
    async with pool.acquire() as conn:
        deleted = await strategy_repo.delete_strategy(conn, strategy_id)

    if not deleted:
        raise HTTPException(status_code=404, detail="Strategy not found")

    if event_publisher:
        await event_publisher.publish_strategy_updated(strategy_id, "", "deleted", "deleted", {})


@router.post("/{strategy_id}/activate", response_model=StrategyResponse)
async def activate_strategy(strategy_id: str, _user: dict = Depends(verify_jwt)):
    pool = await get_pool()
    async with pool.acquire() as conn:
        strategy = await strategy_repo.activate_strategy(conn, strategy_id)

    if strategy is None:
        raise HTTPException(status_code=404, detail="Strategy not found")

    if event_publisher:
        await event_publisher.publish_strategy_updated(
            strategy.id, strategy.name, strategy.status, "activated", {},
        )

    return strategy


@router.post("/{strategy_id}/deactivate", response_model=StrategyResponse)
async def deactivate_strategy(strategy_id: str, _user: dict = Depends(verify_jwt)):
    pool = await get_pool()
    async with pool.acquire() as conn:
        strategy = await strategy_repo.deactivate_strategy(conn, strategy_id)

    if strategy is None:
        raise HTTPException(status_code=404, detail="Strategy not found")

    if event_publisher:
        await event_publisher.publish_strategy_updated(
            strategy.id, strategy.name, strategy.status, "deactivated", {},
        )

    return strategy
