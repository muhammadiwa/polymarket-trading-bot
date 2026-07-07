import logging
from typing import Literal

from fastapi import APIRouter, Depends, HTTPException

from app.middleware.auth import extract_user, require_admin
from app.routes.ws import _get_shared_nats

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/execution-mode", tags=["execution-mode"])

# In-memory cache — refreshed from Redis on read
_current_mode: str = "LIVE"


async def _get_mode_from_redis() -> str:
    """Read execution mode from Redis."""
    global _current_mode
    try:
        import redis.asyncio as aioredis
        from app.config import config
        r = aioredis.from_url(config.REDIS_URL)
        mode = await r.get("pqap:execution_mode")
        await r.close()
        if mode:
            _current_mode = mode.decode() if isinstance(mode, bytes) else str(mode)
    except Exception:
        pass
    return _current_mode


async def _set_mode_in_redis(mode: str) -> None:
    """Write execution mode to Redis."""
    global _current_mode
    try:
        import redis.asyncio as aioredis
        from app.config import config
        r = aioredis.from_url(config.REDIS_URL)
        await r.set("pqap:execution_mode", mode)
        await r.close()
        _current_mode = mode
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
    body: dict,
    user: dict = Depends(extract_user),
):
    """Set execution mode (admin only)."""
    require_admin(user)

    mode = body.get("mode", "").upper()
    if mode not in ("LIVE", "PAPER"):
        raise HTTPException(status_code=400, detail="Mode must be 'LIVE' or 'PAPER'")

    await _set_mode_in_redis(mode)
    logger.info("execution mode changed", extra={"mode": mode, "user": user.get("username")})

    return {"mode": mode, "message": f"Execution mode set to {mode}"}
