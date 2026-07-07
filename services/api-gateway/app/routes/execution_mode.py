import logging
from datetime import datetime, timezone
from typing import Literal

import redis.asyncio as aioredis
from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel

from app.config import config
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

    # Log mode change
    logger.info("execution mode changed",
        extra={
            "from_mode": current_mode,
            "to_mode": body.mode,
            "user": user.get("username"),
            "timestamp": datetime.now(timezone.utc).isoformat(),
        })

    # PAPER→LIVE: warn about open paper positions
    if current_mode == "PAPER" and body.mode == "LIVE":
        logger.warning("PAPER→LIVE switch — restart required to take effect")
        return ExecutionModeResponse(
            mode=body.mode,
            message="Mode set to LIVE. Restart required to take effect. Open paper positions will be preserved for reference.",
            restart_required=True,
        )

    return ExecutionModeResponse(
        mode=body.mode,
        message=f"Mode set to {body.mode}. Restart required to take effect.",
        restart_required=True,
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
        "message": f"Restart confirmed. System will restart in LIVE mode." if mode == "LIVE" else f"Restart confirmed. System will restart in PAPER mode.",
        "mode": mode,
    }
