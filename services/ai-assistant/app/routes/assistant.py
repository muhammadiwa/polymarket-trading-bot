import asyncio
import logging
import time
from datetime import datetime, timezone
from typing import Optional

import httpx
from fastapi import APIRouter, Depends, HTTPException, Response

from app.db import get_pool
from app.engine.decision_explainer import explain_trade
from app.engine.llm_client import LLMClient
from app.engine.performance_qa import answer_question
from app.engine.risk_advisor import suggest_risk_parameters
from app.middleware.auth import verify_jwt
from app.models.assistant import AskRequest, AskResponse, ConversationHistory, TradeExplanation
from app.repos import conversation_repo

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/assistant", tags=["assistant"])

# Rate limiting with TTL eviction
_rate_limit_cache: dict[str, float] = {}
_rate_limit_lock = asyncio.Lock()
RATE_LIMIT_SECONDS = 2
RATE_LIMIT_TTL = 300  # 5 minutes
RATE_LIMIT_MAX_ENTRIES = 10000

llm = LLMClient()


async def _check_rate_limit(user_id: str) -> bool:
    """Check if user is within rate limit. Returns True if allowed."""
    now = time.time()

    async with _rate_limit_lock:
        # Evict stale entries if cache is large
        if len(_rate_limit_cache) > RATE_LIMIT_MAX_ENTRIES:
            cutoff = now - RATE_LIMIT_TTL
            stale = [k for k, v in _rate_limit_cache.items() if v < cutoff]
            for k in stale:
                del _rate_limit_cache[k]

        last_request = _rate_limit_cache.get(user_id, 0)
        if now - last_request < RATE_LIMIT_SECONDS:
            return False
        _rate_limit_cache[user_id] = now
        return True


def _rate_limit_error() -> HTTPException:
    """Create rate limit error with Retry-After header."""
    return HTTPException(
        status_code=429,
        detail="Rate limit exceeded. Please wait 2 seconds between requests.",
        headers={"Retry-After": str(RATE_LIMIT_SECONDS)},
    )


@router.post("/ask", response_model=AskResponse)
async def ask_question(
    request: AskRequest,
    response: Response,
    user: dict = Depends(verify_jwt),
):
    """Ask a performance question about trading."""
    user_id = user.get("user_id")
    if not user_id:
        raise HTTPException(status_code=400, detail="Missing user_id in token")

    if not await _check_rate_limit(user_id):
        raise _rate_limit_error()

    try:
        result = await answer_question(pool=await get_pool(), llm=llm, question=request.question)
    except httpx.TimeoutException:
        logger.warning("llm timeout", extra={"user_id": user_id})
        raise HTTPException(status_code=504, detail="LLM request timed out. Please try again.")
    except httpx.HTTPStatusError as e:
        logger.error("llm http error", extra={"status": e.response.status_code, "user_id": user_id})
        raise HTTPException(status_code=502, detail="LLM service error. Please try again.")
    except ValueError as e:
        logger.error("llm value error", extra={"error": str(e), "user_id": user_id})
        raise HTTPException(status_code=502, detail="LLM returned invalid response. Please try again.")
    except Exception as e:
        logger.error("unexpected error", extra={"error": str(e), "user_id": user_id}, exc_info=True)
        raise HTTPException(status_code=500, detail="Internal server error.")

    # Save conversation (best effort - don't fail the request)
    conversation_id = ""
    try:
        pool = await get_pool()
        async with pool.acquire() as conn:
            await conversation_repo.save_message(conn, user_id, "user", request.question)
            conversation_id = await conversation_repo.save_message(
                conn, user_id, "assistant", result["answer"],
                metadata={
                    "data_points": result["data_points"],
                    "source_trade_ids": result["source_trade_ids"],
                },
            )
    except Exception as e:
        logger.warning("failed to save conversation", extra={"error": str(e), "user_id": user_id})

    return AskResponse(
        answer=result["answer"],
        data_points=result["data_points"],
        source_trade_ids=result["source_trade_ids"],
        response_time_ms=result["response_time_ms"],
        conversation_id=conversation_id,
    )


@router.post("/explain-trade/{trade_id}", response_model=TradeExplanation)
async def explain_trade_endpoint(
    trade_id: str,
    response: Response,
    user: dict = Depends(verify_jwt),
):
    """Explain why a specific trade was made."""
    user_id = user.get("user_id")
    if not user_id:
        raise HTTPException(status_code=400, detail="Missing user_id in token")

    if not await _check_rate_limit(user_id):
        raise _rate_limit_error()

    try:
        result = await explain_trade(pool=await get_pool(), llm=llm, trade_id=trade_id)
    except httpx.TimeoutException:
        logger.warning("llm timeout", extra={"trade_id": trade_id})
        raise HTTPException(status_code=504, detail="LLM request timed out. Please try again.")
    except httpx.HTTPStatusError as e:
        logger.error("llm http error", extra={"status": e.response.status_code, "trade_id": trade_id})
        raise HTTPException(status_code=502, detail="LLM service error. Please try again.")
    except ValueError as e:
        logger.error("llm value error", extra={"error": str(e), "trade_id": trade_id})
        raise HTTPException(status_code=502, detail="LLM returned invalid response. Please try again.")
    except Exception as e:
        logger.error("unexpected error", extra={"error": str(e), "trade_id": trade_id}, exc_info=True)
        raise HTTPException(status_code=500, detail="Internal server error.")

    if "error" in result:
        raise HTTPException(status_code=404, detail=result["error"])

    # Save conversation (best effort)
    conversation_id = ""
    try:
        pool = await get_pool()
        async with pool.acquire() as conn:
            question = f"Explain trade {trade_id}"
            await conversation_repo.save_message(conn, user_id, "user", question)
            conversation_id = await conversation_repo.save_message(
                conn, user_id, "assistant", result["explanation"],
                metadata={"trade_id": trade_id},
            )
    except Exception as e:
        logger.warning("failed to save conversation", extra={"error": str(e), "trade_id": trade_id})

    return TradeExplanation(
        trade_id=result["trade_id"],
        market_id=result["market_id"] or "unknown",
        side=result["side"] or "unknown",
        entry_price=result["entry_price"] or 0,
        exit_price=result["exit_price"],
        pnl=result["pnl"],
        explanation=result["explanation"],
        decision_context=result["decision_context"],
        related_events=result["related_events"],
        conversation_id=conversation_id,
    )


@router.get("/history", response_model=ConversationHistory)
async def get_history(
    limit: int = 50,
    user: dict = Depends(verify_jwt),
):
    """Get conversation history."""
    user_id = user.get("user_id")
    if not user_id:
        raise HTTPException(status_code=400, detail="Missing user_id in token")

    # Cap limit to prevent resource exhaustion
    limit = min(limit, 200)

    pool = await get_pool()
    async with pool.acquire() as conn:
        messages = await conversation_repo.get_history(conn, user_id, limit)

    return ConversationHistory(messages=messages, total=len(messages))


@router.post("/suggest-risk-parameters")
async def suggest_risk_params(
    strategy_id: Optional[str] = None,
    user: dict = Depends(verify_jwt),
):
    """Generate conservative risk parameter suggestions based on current state.
    Read-only — suggestions are NOT auto-applied. User must manually update.
    """
    await _check_rate_limit(user.get("user_id", ""))

    # Fetch current risk state from risk-manager API
    risk_state = await _get_risk_state()

    pool = await get_pool()
    async with pool.acquire() as conn:
        suggestions = await suggest_risk_parameters(conn, risk_state, strategy_id)

    logger.info("risk parameter suggestions generated",
        extra={"user": user.get("username"), "count": len(suggestions)})

    return {
        "suggestions": suggestions,
        "analysis_timestamp": datetime.now(timezone.utc).isoformat(),
        "read_only": True,
        "requires_approval": True,
    }


async def _get_risk_state() -> dict:
    """Fetch current risk state from risk-manager API."""
    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            resp = await client.get(f"{config.RISK_MANAGER_URL}/api/v1/risk/state")
            if resp.status_code == 200:
                return resp.json()
    except Exception as e:
        logger.warning("failed to fetch risk state", exc_info=e)
    return {}
