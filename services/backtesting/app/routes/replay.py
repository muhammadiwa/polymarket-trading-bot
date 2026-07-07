import asyncio
import json
import logging
import uuid
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException
from fastapi.responses import StreamingResponse

from app.db import get_pg_pool, get_ts_pool
from app.engine.replay_engine import replay_events
from app.middleware.auth import verify_jwt
from app.models.backtest import ReplayRequest
from app.repos import backtest_repo

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/replay", tags=["replay"])

# In-memory replay sessions with cleanup
_sessions: dict[str, dict] = {}
MAX_SESSIONS = 100


@router.post("")
async def start_replay(body: ReplayRequest, _user: dict = Depends(verify_jwt)):
    """Start a new replay session."""
    # #5: Limit sessions
    if len(_sessions) >= MAX_SESSIONS:
        raise HTTPException(status_code=429, detail="Too many active replay sessions")

    session_id = str(uuid.uuid4())
    ts_pool = await get_ts_pool()
    async with ts_pool.acquire() as conn:
        opportunities = await backtest_repo.get_opportunities(conn, body.start_date, body.end_date)

    if not opportunities:
        raise HTTPException(status_code=404, detail="No opportunities found for the given date range")

    _sessions[session_id] = {
        "opportunities": opportunities,
        "speed": body.speed,
        "index": 0,
        "status": "ready",
        "user_id": _user.get("user_id"),
        "created_at": asyncio.get_event_loop().time(),
    }

    return {"session_id": session_id, "status": "ready", "total_events": len(opportunities)}


@router.get("/{session_id}/events")
async def stream_events(session_id: str, _user: dict = Depends(verify_jwt)):
    """Stream replay events via SSE."""
    session = _sessions.get(session_id)
    if not session:
        raise HTTPException(status_code=404, detail="Replay session not found")

    session["status"] = "running"
    opportunities = session["opportunities"]
    speed = session["speed"]

    async def event_generator():
        try:
            async for event in replay_events(opportunities, speed):
                yield f"event: {event.event_type}\ndata: {json.dumps(event.data)}\n\n"
        finally:
            # #3: Always clean up session on disconnect
            _sessions.pop(session_id, None)

    return StreamingResponse(
        event_generator(),
        media_type="text/event-stream",
        headers={"Cache-Control": "no-cache", "Connection": "keep-alive"},
    )


@router.post("/{session_id}/step")
async def step_forward(session_id: str, _user: dict = Depends(verify_jwt)):
    """Step forward one event (for pause/step mode)."""
    session = _sessions.get(session_id)
    if not session:
        raise HTTPException(status_code=404, detail="Replay session not found")

    idx = session["index"]
    opportunities = session["opportunities"]

    if idx >= len(opportunities):
        # #6: Return done event when finished
        return {"event": {"event_type": "done", "timestamp": "", "data": {"total_events": len(opportunities) * 2, "total_decisions": len(opportunities)}}, "has_more": False}

    opp = opportunities[idx]
    session["index"] = idx + 1

    from decimal import Decimal
    ts = str(opp.get("detected_at", ""))
    market_id = opp.get("market_id", "")
    spread = Decimal(str(opp.get("spread", "0")))
    score = Decimal(str(opp.get("score", "0")))
    filter_reason = opp.get("filter_reason")
    side = opp.get("side", "YES")

    if filter_reason:
        decision, reason, risk_result = "FILTER", filter_reason, "N/A"
    elif score < Decimal("0.01"):
        decision, reason, risk_result = "SKIP", "Score below threshold", "N/A"
    else:
        decision, reason, risk_result = "EXECUTE", "Score above threshold", "ALLOWED"

    # #6: Return both market_update and decision events
    market_event = {
        "event_type": "market_update",
        "timestamp": ts,
        "data": {"market_id": market_id, "spread": str(spread), "score": str(score), "side": side},
    }
    decision_event = {
        "event_type": "decision",
        "timestamp": ts,
        "data": {
            "market_id": market_id,
            "detected": "YES+NO arbitrage" if side == "YES" else "Cross-market arbitrage",
            "decision": decision,
            "reason": reason,
            "score": str(score),
            "risk_result": risk_result,
        },
    }

    return {"events": [market_event, decision_event], "has_more": idx + 1 < len(opportunities)}


@router.post("/{session_id}/speed")
async def update_speed(session_id: str, speed: int = Query(ge=1, le=10), _user: dict = Depends(verify_jwt)):
    """#2: Update replay speed dynamically."""
    session = _sessions.get(session_id)
    if not session:
        raise HTTPException(status_code=404, detail="Replay session not found")

    session["speed"] = speed
    return {"session_id": session_id, "speed": speed}


@router.get("/{session_id}/status")
async def get_status(session_id: str, _user: dict = Depends(verify_jwt)):
    """Get replay session status."""
    session = _sessions.get(session_id)
    if not session:
        raise HTTPException(status_code=404, detail="Replay session not found")

    return {
        "session_id": session_id,
        "status": session["status"],
        "total_events": len(session["opportunities"]),
        "current_index": session["index"],
        "speed": session["speed"],
    }


@router.delete("/{session_id}")
async def delete_session(session_id: str, _user: dict = Depends(verify_jwt)):
    """#3: Delete replay session to free resources."""
    session = _sessions.pop(session_id, None)
    if not session:
        raise HTTPException(status_code=404, detail="Replay session not found")
    return {"message": "Session deleted"}
