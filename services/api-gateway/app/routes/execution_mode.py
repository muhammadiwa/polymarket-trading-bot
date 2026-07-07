import logging
from typing import Literal

from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel

from app.middleware.auth import extract_user, require_admin

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/execution-mode", tags=["execution-mode"])


class ExecutionModeRequest(BaseModel):
    mode: Literal["LIVE", "PAPER"]


class ExecutionModeResponse(BaseModel):
    mode: str
    message: str


async def _get_mode_from_redis() -> str:
    """Read execution mode from Redis."""
    try:
        import redis.asyncio as aioredis
        from app.config import config
        r = aioredis.from_url(config.REDIS_URL)
        mode = await r.get("pqap:execution_mode")
        await r.close()
        if mode:
            val = mode.decode() if isinstance(mode, bytes) else str(mode)
            # #4: Validate mode
            if val in ("LIVE", "PAPER"):
                return val
        return "LIVE"
    except Exception as e:
        # #8: Log Redis errors
        logger.warning("failed to read execution mode from Redis, defaulting to LIVE", exc_info=e)
        return "LIVE"


async def _set_mode_in_redis(mode: str) -> None:
    """Write execution mode to Redis."""
    try:
        import redis.asyncio as aioredis
        from app.config import config
        r = aioredis.from_url(config.REDIS_URL)
        await r.set("pqap:execution_mode", mode)
        await r.close()
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

    await _set_mode_in_redis(body.mode)
    logger.info("execution mode changed", extra={"mode": body.mode, "user": user.get("username")})

    return ExecutionModeResponse(mode=body.mode, message=f"Execution mode set to {body.mode}")
