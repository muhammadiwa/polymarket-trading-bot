import json
import logging
from datetime import datetime, timezone
from typing import Literal
from uuid import uuid4

import nats
import redis.asyncio as aioredis
from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel

from app.config import config
from app.db import get_pool
from app.middleware.auth import extract_user, require_admin

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/execution-mode", tags=["execution-mode"])

_redis_client: aioredis.Redis = None


def _get_redis() -> aioredis.Redis:
    global _redis_client
    if _redis_client is None:
        _redis_client = aioredis.from_url(config.REDIS_URL)
    return _redis_client


class ExecutionModeRequest(BaseModel):
    mode: Literal["LIVE", "PAPER"]


class ExecutionModeResponse(BaseModel):
    mode: str
    message: str
    restart_required: bool = False
    open_paper_positions: int = 0


class RestartConfirmation(BaseModel):
    confirm: bool


async def _get_mode_from_redis() -> str:
    try:
        r = _get_redis()
        mode = await r.get("pqap:execution_mode")
        if mode:
            val = mode.decode() if isinstance(mode, bytes) else str(mode)
            if val in ("LIVE", "PAPER"):
                return val
        return "LIVE"
    except Exception as e:
        logger.warning("failed to read execution mode from Redis, defaulting to LIVE", exc_info=e)
        return "LIVE"


async def _set_mode_in_redis(mode: str) -> None:
    try:
        r = _get_redis()
        await r.set("pqap:execution_mode", mode)
    except Exception as e:
        logger.error("failed to set execution mode in Redis", exc_info=e)
        raise


async def _get_open_paper_positions_count() -> int:
    """Query open paper positions count from database."""
    try:
        pool = await get_pool()
        async with pool.acquire() as conn:
            row = await conn.fetchrow(
                "SELECT COUNT(*) as cnt FROM paper_positions WHERE status = 'open'"
            )
            return row["cnt"] if row else 0
    except Exception as e:
        logger.warning("failed to query paper positions", exc_info=e)
        return 0


async def _publish_mode_switch_warning(from_mode: str, to_mode: str, open_count: int, username: str):
    """Publish warning notification to NATS."""
    try:
        nc = await nats.connect(config.NATS_URL)
        event = {
            "event_id": str(uuid4()),
            "event_type": "NotificationRequest",
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "source": "api-gateway",
            "payload": {
                "category": "risk",
                "title": f"Mode Switch: {from_mode} → {to_mode}",
                "message": f"Execution mode changed from {from_mode} to {to_mode}. {open_count} paper position(s) open. Restart required to take effect.",
                "channel": "telegram",
                "priority": "high",
                "bypass_throttle": True,
            },
        }
        await nc.publish("pqap.notification.request", json.dumps(event).encode())
        await nc.close()
        logger.info("mode switch warning published", extra={"from": from_mode, "to": to_mode})
    except Exception as e:
        logger.error("failed to publish mode switch warning", exc_info=e)


@router.get("")
async def get_execution_mode(user: dict = Depends(extract_user)):
    """Get current execution mode (LIVE or PAPER)."""
    mode = await _get_mode_from_redis()
    return {"mode": mode}


@router.put("")
async def set_execution_mode(
    body: ExecutionModeRequest,
    user: dict = Depends(extract_user),
):
    """Set execution mode (admin only). Requires restart to take effect."""
    require_admin(user)

    current_mode = await _get_mode_from_redis()
    if current_mode == body.mode:
        return ExecutionModeResponse(mode=body.mode, message="Mode already set", restart_required=False)

    await _set_mode_in_redis(body.mode)

    # Get open paper positions count
    open_count = await _get_open_paper_positions_count()

    # Log mode change
    logger.info("execution mode changed",
        extra={
            "from_mode": current_mode,
            "to_mode": body.mode,
            "user": user.get("username"),
            "open_paper_positions": open_count,
            "timestamp": datetime.now(timezone.utc).isoformat(),
        })

    # PAPER→LIVE: publish warning notification
    if current_mode == "PAPER" and body.mode == "LIVE":
        await _publish_mode_switch_warning(current_mode, body.mode, open_count, user.get("username", "unknown"))

    return ExecutionModeResponse(
        mode=body.mode,
        message=f"Mode set to {body.mode}. Restart required to take effect.",
        restart_required=True,
        open_paper_positions=open_count,
    )


@router.post("/restart")
async def confirm_restart(
    body: RestartConfirmation,
    user: dict = Depends(extract_user),
):
    """Confirm restart for mode switch. Admin only."""
    require_admin(user)

    if not body.confirm:
        raise HTTPException(status_code=400, detail="Restart not confirmed")

    mode = await _get_mode_from_redis()
    logger.info("restart confirmed for mode switch",
        extra={"mode": mode, "user": user.get("username")})

    return {
        "message": f"Restart confirmed. System will restart in {mode} mode.",
        "mode": mode,
    }
