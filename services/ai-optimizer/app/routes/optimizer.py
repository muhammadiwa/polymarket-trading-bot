import logging
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, Query

from app.db import get_pool
from app.engine.pattern_analyzer import analyze_trades
from app.middleware.auth import verify_jwt
from app.models.optimizer import AnalysisResult, SuggestionListResponse, SuggestionResponse
from app.repos import optimizer_repo

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/optimizer", tags=["optimizer"])

# #3: Simple in-memory rate limiter per strategy
_analysis_cooldown: dict[str, float] = {}
COOLDOWN_SECONDS = 60  # Minimum 60 seconds between analyses per strategy


@router.post("/analyze", response_model=AnalysisResult)
async def run_analysis(
    strategy_id: str = Query(..., min_length=1, max_length=64),
    min_trades: int = Query(100, ge=10),
    _user: dict = Depends(verify_jwt),
):
    """Run pattern analysis on trade history for a strategy."""
    # #3: Rate limit per strategy
    import time
    now = time.time()
    last_run = _analysis_cooldown.get(strategy_id, 0)
    if now - last_run < COOLDOWN_SECONDS:
        remaining = int(COOLDOWN_SECONDS - (now - last_run))
        raise HTTPException(
            status_code=429,
            detail=f"Analysis for this strategy was run recently. Try again in {remaining} seconds.",
        )
    _analysis_cooldown[strategy_id] = now

    pool = await get_pool()
    async with pool.acquire() as conn:
        # Check minimum trades
        trade_count = await optimizer_repo.count_trades(conn, strategy_id)
        if trade_count < min_trades:
            raise HTTPException(
                status_code=400,
                detail=f"Insufficient data: {trade_count} trades found, {min_trades} required",
            )

        # Fetch trades
        trades = await optimizer_repo.get_trades(conn, strategy_id)

    # Run analysis
    patterns = await analyze_trades(trades)

    # Save suggestions
    saved_count = 0
    async with pool.acquire() as conn:
        for pattern in patterns:
            pattern["strategy_id"] = strategy_id
            await optimizer_repo.save_suggestion(conn, pattern)
            saved_count += 1

    logger.info("analysis completed", extra={"strategy_id": strategy_id, "patterns": len(patterns), "suggestions": saved_count})

    return AnalysisResult(
        patterns_found=len(patterns),
        suggestions_generated=saved_count,
        strategy_id=strategy_id,
    )


@router.get("/suggestions", response_model=SuggestionListResponse)
async def list_suggestions(
    strategy_id: Optional[str] = Query(None),
    status: Optional[str] = Query(None, pattern="^(pending|approved|rejected)$"),
    limit: int = Query(50, ge=1, le=200),
    offset: int = Query(0, ge=0),
    _user: dict = Depends(verify_jwt),
):
    """List optimizer suggestions."""
    pool = await get_pool()
    async with pool.acquire() as conn:
        suggestions, total = await optimizer_repo.get_suggestions(conn, strategy_id, status, limit, offset)

    return SuggestionListResponse(suggestions=suggestions, total=total)


@router.post("/suggestions/{suggestion_id}/approve", response_model=SuggestionResponse)
async def approve_suggestion(
    suggestion_id: str,
    user: dict = Depends(verify_jwt),
):
    """Approve an optimizer suggestion."""
    pool = await get_pool()
    async with pool.acquire() as conn:
        result = await optimizer_repo.update_suggestion_status(conn, suggestion_id, "approved", user.get("user_id"))

    if result is None:
        raise HTTPException(status_code=404, detail="Suggestion not found or already reviewed")

    logger.info("suggestion approved", extra={"suggestion_id": suggestion_id, "user": user.get("username")})
    return result


@router.post("/suggestions/{suggestion_id}/reject", response_model=SuggestionResponse)
async def reject_suggestion(
    suggestion_id: str,
    user: dict = Depends(verify_jwt),
):
    """Reject an optimizer suggestion."""
    pool = await get_pool()
    async with pool.acquire() as conn:
        result = await optimizer_repo.update_suggestion_status(conn, suggestion_id, "rejected", user.get("user_id"))

    if result is None:
        raise HTTPException(status_code=404, detail="Suggestion not found or already reviewed")

    logger.info("suggestion rejected", extra={"suggestion_id": suggestion_id, "user": user.get("username")})
    return result
