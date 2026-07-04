import time
from datetime import datetime, timezone
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, Query

from app.db import get_pool
from app.middleware.auth import verify_jwt
from app.models.notification import (
    NotificationHistoryResponse,
    NotificationPreferencesResponse,
    NotificationPreferencesUpdate,
)
from app.repos import notification_repo

router = APIRouter(prefix="/api/notifications", tags=["notifications"])


@router.get("/preferences", response_model=NotificationPreferencesResponse)
async def get_preferences(
    _user: dict = Depends(verify_jwt),
):
    pool = await get_pool()
    async with pool.acquire() as conn:
        prefs = await notification_repo.get_preferences(conn)

    if prefs is None:
        raise HTTPException(status_code=404, detail="Preferences not found")

    return prefs


@router.put("/preferences", response_model=NotificationPreferencesResponse)
async def update_preferences(
    body: NotificationPreferencesUpdate,
    _user: dict = Depends(verify_jwt),
):
    pool = await get_pool()
    try:
        async with pool.acquire() as conn:
            prefs = await notification_repo.update_preferences(
                conn,
                categories=body.categories,
                channels=body.channels,
            )
    except ValueError as exc:
        raise HTTPException(status_code=404, detail=str(exc))

    return prefs


@router.get("/history", response_model=NotificationHistoryResponse)
async def get_history(
    category: Optional[str] = Query(None, pattern="^(critical|warning|info|debug)$"),
    status: Optional[str] = Query(None, pattern="^(sent|failed|throttled|suppressed)$"),
    start_date: Optional[str] = Query(None),
    end_date: Optional[str] = Query(None),
    limit: int = Query(50, ge=1, le=200),
    offset: int = Query(0, ge=0),
    _user: dict = Depends(verify_jwt),
):
    start_dt = None
    end_dt = None
    if start_date:
        try:
            start_dt = datetime.fromisoformat(start_date)
            if start_dt.tzinfo is None:
                start_dt = start_dt.replace(tzinfo=timezone.utc)
        except ValueError:
            raise HTTPException(status_code=400, detail="Invalid start_date format")
    if end_date:
        try:
            end_dt = datetime.fromisoformat(end_date)
            if end_dt.tzinfo is None:
                end_dt = end_dt.replace(tzinfo=timezone.utc)
        except ValueError:
            raise HTTPException(status_code=400, detail="Invalid end_date format")

    pool = await get_pool()
    async with pool.acquire() as conn:
        items, total = await notification_repo.get_history(
            conn,
            limit=limit,
            offset=offset,
            category=category,
            status=status,
            start_date=start_dt,
            end_date=end_dt,
        )

    return NotificationHistoryResponse(
        items=items,
        total=total,
        limit=limit,
        offset=offset,
    )


@router.get("/history/{notification_id}")
async def get_notification(
    notification_id: str,
    _user: dict = Depends(verify_jwt),
):
    pool = await get_pool()
    async with pool.acquire() as conn:
        item = await notification_repo.get_history_by_id(conn, notification_id)

    if item is None:
        raise HTTPException(status_code=404, detail="Notification not found")

    return item
