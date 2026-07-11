import logging
from typing import Optional
from uuid import UUID

import asyncpg
from fastapi import APIRouter, Depends, HTTPException, Query, status

from app.db import get_pool
from app.events import StrategyEventPublisher
from app.middleware.auth import verify_jwt
from app.models.strategy import (
    StrategyCreate, StrategyListResponse, StrategyResponse, StrategyUpdate,
    VersionListResponse, VersionResponse, WeightUpdateRequest, WeightUpdateResponse,
)
from app.repos import strategy_repo, version_repo

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/strategies", tags=["strategies"])

# Will be set by main.py
event_publisher: StrategyEventPublisher = None


def set_event_publisher(publisher: StrategyEventPublisher):
    global event_publisher
    event_publisher = publisher


def _validate_uuid(value: str, field_name: str = "id") -> str:
    """#6: Validate UUID format, raise 400 on invalid."""
    try:
        UUID(value)
    except ValueError:
        raise HTTPException(status_code=400, detail=f"Invalid {field_name} format")
    return value


@router.post("", response_model=StrategyResponse, status_code=status.HTTP_201_CREATED)
async def create_strategy(body: StrategyCreate, _user: dict = Depends(verify_jwt)):
    pool = await get_pool()
    async with pool.acquire() as conn:
        try:
            strategy = await strategy_repo.create_strategy(conn, body)
        except asyncpg.UniqueViolationError:
            raise HTTPException(status_code=409, detail="Strategy name already exists")

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
    _validate_uuid(strategy_id)
    pool = await get_pool()
    async with pool.acquire() as conn:
        strategy = await strategy_repo.get_strategy(conn, strategy_id)

    if strategy is None:
        raise HTTPException(status_code=404, detail="Strategy not found")
    return strategy


@router.put("/{strategy_id}", response_model=StrategyResponse)
async def update_strategy(strategy_id: str, body: StrategyUpdate, _user: dict = Depends(verify_jwt)):
    _validate_uuid(strategy_id)
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
    _validate_uuid(strategy_id)
    pool = await get_pool()
    async with pool.acquire() as conn:
        try:
            deleted = await strategy_repo.delete_strategy(conn, strategy_id)
        except ValueError as e:
            raise HTTPException(status_code=400, detail=str(e))

    if not deleted:
        raise HTTPException(status_code=404, detail="Strategy not found")

    if event_publisher:
        await event_publisher.publish_strategy_updated(strategy_id, "", "deleted", "deleted", {})


@router.post("/{strategy_id}/activate", response_model=StrategyResponse)
async def activate_strategy(strategy_id: str, _user: dict = Depends(verify_jwt)):
    _validate_uuid(strategy_id)
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
    _validate_uuid(strategy_id)
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


# --- Version History Endpoints ---


@router.get("/{strategy_id}/versions", response_model=VersionListResponse)
async def list_versions(
    strategy_id: str,
    limit: int = Query(50, ge=1, le=200),
    offset: int = Query(0, ge=0),
    _user: dict = Depends(verify_jwt),
):
    _validate_uuid(strategy_id)
    pool = await get_pool()
    async with pool.acquire() as conn:
        # Verify strategy exists
        strategy = await strategy_repo.get_strategy(conn, strategy_id)
        if strategy is None:
            raise HTTPException(status_code=404, detail="Strategy not found")
        items = await version_repo.get_versions(conn, strategy_id, limit, offset)

    items_list, total = items
    return VersionListResponse(items=items_list, total=total)


@router.get("/{strategy_id}/versions/{version_id}", response_model=VersionResponse)
async def get_version(strategy_id: str, version_id: str, _user: dict = Depends(verify_jwt)):
    _validate_uuid(strategy_id)
    _validate_uuid(version_id, "version_id")
    pool = await get_pool()
    async with pool.acquire() as conn:
        version = await version_repo.get_version(conn, strategy_id, version_id)

    if version is None:
        raise HTTPException(status_code=404, detail="Version not found")
    return version


@router.post("/{strategy_id}/rollback/{version_id}", response_model=StrategyResponse)
async def rollback_strategy(strategy_id: str, version_id: str, user: dict = Depends(verify_jwt)):
    _validate_uuid(strategy_id)
    _validate_uuid(version_id, "version_id")
    pool = await get_pool()
    async with pool.acquire() as conn:
        # Get the target version
        version = await version_repo.get_version(conn, strategy_id, version_id)
        if version is None:
            raise HTTPException(status_code=404, detail="Version not found")

        # #2: Use dedicated rollback function (creates exactly 2 versions)
        strategy = await strategy_repo.rollback_strategy(
            conn, strategy_id, version["parameters"],
            version["version_number"], user.get("user_id"),
        )

    if strategy is None:
        raise HTTPException(status_code=404, detail="Strategy not found")

    if event_publisher:
        await event_publisher.publish_strategy_updated(
            strategy.id, strategy.name, strategy.status, "rollback",
            {"rollback_to_version": version["version_number"]},
        )

    return strategy


# --- Capital Allocation Weights ---


@router.post("/weights", response_model=WeightUpdateResponse)
async def update_weights(body: WeightUpdateRequest, user: dict = Depends(verify_jwt)):
    if not body.weights:
        raise HTTPException(status_code=400, detail="Weights dict cannot be empty")

    # #7: Validate each weight is 0-100
    for sid, w in body.weights.items():
        if w < 0 or w > 100:
            raise HTTPException(status_code=400, detail=f"Weight for {sid} must be 0-100")

    # #8: Validate sum == 100% ±0.01% (per spec)
    total = sum(body.weights.values())
    if abs(total - 100.0) > 0.01:
        raise HTTPException(
            status_code=400,
            detail=f"Weights must sum to 100% (±0.01%). Current sum: {total:.4f}%",
        )

    pool = await get_pool()
    async with pool.acquire() as conn:
        # #4: Wrap in transaction for atomicity
        async with conn.transaction():
            for strategy_id, weight in body.weights.items():
                _validate_uuid(strategy_id)
                strategy = await strategy_repo.get_strategy(conn, strategy_id)
                if strategy is None:
                    raise HTTPException(status_code=404, detail=f"Strategy {strategy_id} not found")
                # #26: Only allow weight assignment for active strategies
                if strategy.status != "active":
                    raise HTTPException(status_code=400, detail=f"Strategy {strategy_id} is not active (status: {strategy.status})")

                update_data = StrategyUpdate(capital_weight=weight)
                await strategy_repo.update_strategy(conn, strategy_id, update_data, user.get("user_id"))

    if event_publisher:
        await event_publisher.publish_strategy_weights_updated(body.weights)

    return WeightUpdateResponse(
        weights=body.weights,
        total=total,
        updated_count=len(body.weights),
    )
