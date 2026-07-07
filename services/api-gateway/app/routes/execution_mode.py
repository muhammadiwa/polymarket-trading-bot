import logging
from typing import Literal

import redis.asyncio as aioredis
from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel

from app.config import config
from app.middleware.auth import extract_user, require_admin

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/execution-mode", tags=["execution-mode"])

# #4: Shared Redis client for connection pooling
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


async def _get_mode_from_redis() -> str:
    """Read execution mode from Redis."""
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
    """Write execution mode to Redis."""
    try:
        r = _get_redis()
        await r.set("pqap:execution_mode", mode)
    except Exception as e:
        logger.error("failed to set execution mode in Redis", exc_info=e)
        raise


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
    """Set execution mode (admin only)."""
    require_admin(user)

    # Get current mode before switching
    current_mode = await _get_mode_from_redis()

    await _set_mode_in_redis(body.mode)
    logger.info("execution mode changed", extra={"mode": body.mode, "user": user.get("username")})

    # #11: If switching from PAPER to LIVE, log warning about open paper positions
    if current_mode == "PAPER" and body.mode == "LIVE":
        logger.warning("switching from PAPER to LIVE — open paper positions may exist")
        # In production: auto-close paper positions or warn user

    return ExecutionModeResponse(mode=body.mode, message=f"Execution mode set to {body.mode}")
