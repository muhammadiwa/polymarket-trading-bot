import asyncio
import json
import logging
from collections import deque
from decimal import Decimal

import nats
from fastapi import APIRouter, WebSocket, WebSocketDisconnect, status
from jose import JWTError, jwt

from app.config import config
from app.metrics import WS_CONNECTIONS, WS_MESSAGES_SENT

logger = logging.getLogger(__name__)

router = APIRouter(tags=["websocket"])

_connection_counts: dict[str, int] = {}
_connection_locks: dict[str, asyncio.Lock] = {}
_global_lock = asyncio.Lock()
MAX_CONNECTIONS_PER_USER = 5

# #3: Shared NATS connection per gateway process
_shared_nc: nats.NATS | None = None
_shared_nc_lock = asyncio.Lock()

# #11: Per-client send buffer size (drop-oldest on overflow)
CLIENT_SEND_BUFFER_SIZE = 64


async def _get_shared_nats() -> nats.NATS:
    global _shared_nc
    async with _shared_nc_lock:
        if _shared_nc is None or _shared_nc.is_closed:
            _shared_nc = await nats.connect(config.NATS_URL)
            logger.info("Shared NATS connection established for gateway process")
        return _shared_nc


async def _get_user_lock(user_id: str) -> asyncio.Lock:
    async with _global_lock:
        if user_id not in _connection_locks:
            _connection_locks[user_id] = asyncio.Lock()
        return _connection_locks[user_id]


async def _authenticate_ws(websocket: WebSocket) -> dict | None:
    # #3: Extract token from query param or header BEFORE accept()
    token = websocket.query_params.get("token")
    if not token:
        auth_header = websocket.headers.get("authorization", "")
        if auth_header.startswith("Bearer "):
            token = auth_header[7:]

    if not token:
        await websocket.close(code=status.WS_1008_POLICY_VIOLATION, reason="Missing token")
        return None

    try:
        payload = jwt.decode(token, config.JWT_SECRET, algorithms=[config.JWT_ALGORITHM])
        return payload
    except JWTError:
        await websocket.close(code=status.WS_1008_POLICY_VIOLATION, reason="Invalid token")
        return None


def _transform_risk_state_to_status(state: dict) -> dict:
    """Transform risk-manager RiskStateUpdated event to RiskStatus format for dashboard."""
    payload = state.get("payload", state)

    daily_budget_remaining = str(payload.get("daily_budget_remaining", "0"))
    daily_loss_limit = str(payload.get("daily_loss_limit", "0"))
    daily_loss = str(payload.get("daily_loss", "0"))

    used_pct = "0"
    try:
        remaining = Decimal(daily_budget_remaining)
        total = Decimal(daily_loss_limit)
        if total > 0:
            raw = (total - remaining) / total
            clamped = max(Decimal("0"), min(Decimal("1"), raw))
            used_pct = str(clamped)
    except Exception:
        pass

    emergency_stop = payload.get("emergency_stop", False)
    batasi_paused = payload.get("batasi_win_paused", False)
    is_paused = emergency_stop or batasi_paused

    paused_reason = None
    if emergency_stop:
        paused_reason = payload.get("emergency_stop_reason", "Emergency stop active")
    elif batasi_paused:
        paused_reason = "Win streak threshold reached"

    return {
        "type": "risk_update",
        "payload": {
            "dailyBudgetRemaining": daily_budget_remaining,
            "dailyBudgetTotal": daily_loss_limit,
            "dailyBudgetUsedFraction": used_pct,
            "currentDrawdown": str(payload.get("drawdown", "0")),
            "drawdownThreshold": str(payload.get("drawdown_limit", "0")),
            "winStreakCurrent": payload.get("win_streak_current", 0),
            "winStreakThreshold": payload.get("win_streak_threshold", 5),
            "circuitBreakerStatus": "open" if emergency_stop else "closed",
            "circuitBreakerTrippedAt": payload.get("emergency_stop_timestamp"),
            "isPaused": is_paused,
            "pausedReason": paused_reason,
            "lastUpdated": payload.get("updated_at"),
        },
        "timestamp": payload.get("updated_at", ""),
    }


@router.websocket("/ws/dashboard")
async def dashboard_ws(websocket: WebSocket):
    user = await _authenticate_ws(websocket)
    if user is None:
        return

    user_id = user.get("sub", "unknown")
    user_lock = await _get_user_lock(user_id)
    async with user_lock:
        current = _connection_counts.get(user_id, 0)
        if current >= MAX_CONNECTIONS_PER_USER:
            await websocket.close(code=status.WS_1008_POLICY_VIOLATION, reason="Too many connections")
            logger.warning("Rate limit exceeded for user %s", user_id)
            return
        _connection_counts[user_id] = current + 1

    try:
        await websocket.accept()
        WS_CONNECTIONS.inc()
        logger.info("Dashboard WebSocket connected for user %s", user_id)
    except Exception:
        async with user_lock:
            _connection_counts[user_id] = max(0, _connection_counts.get(user_id, 1) - 1)
        logger.exception("Failed to accept WebSocket for user %s", user_id)
        return

    nc = None
    risk_sub = None
    opp_detected_sub = None
    opp_executed_sub = None
    opp_filtered_sub = None
    health_sub = None
    ping_task = None

    async def send_ping():
        while True:
            try:
                await asyncio.sleep(30)
                await websocket.send_json({"type": "ping"})
                WS_MESSAGES_SENT.inc()
            except Exception:
                break

    try:
        # #3: Use shared NATS connection instead of creating a dedicated one
        nc = await _get_shared_nats()
        logger.info("Using shared NATS connection for WebSocket forwarding")

        # #2: Subscribe to correct risk-manager subject (state_updated with underscore)
        # and transform to RiskStatus format for the dashboard

        # #11: Per-client send buffer with drop-oldest backpressure
        send_buffer: deque[dict] = deque(maxlen=CLIENT_SEND_BUFFER_SIZE)
        send_event = asyncio.Event()
        send_task: asyncio.Task | None = None

        async def drain_send_buffer():
            while True:
                await send_event.wait()
                send_event.clear()
                while send_buffer:
                    msg = send_buffer.popleft()
                    try:
                        await websocket.send_json(msg)
                        WS_MESSAGES_SENT.inc()
                    except Exception:
                        return

        send_task = asyncio.create_task(drain_send_buffer())

        def _enqueue(msg: dict):
            if len(send_buffer) >= CLIENT_SEND_BUFFER_SIZE:
                send_buffer.popleft()  # drop oldest
            send_buffer.append(msg)
            send_event.set()

        async def forward_risk_update(msg):
            try:
                raw = json.loads(msg.data.decode())
                transformed = _transform_risk_state_to_status(raw)
                _enqueue(transformed)
            except Exception:
                logger.exception("Failed to forward risk update to WebSocket")

        async def forward_opportunity(msg):
            try:
                raw = json.loads(msg.data.decode())
                _enqueue({
                    "type": "opportunity",
                    "payload": raw.get("payload", raw),
                    "timestamp": raw.get("timestamp", ""),
                })
            except Exception:
                logger.exception("Failed to forward opportunity to WebSocket")

        async def forward_health_update(msg):
            try:
                raw = json.loads(msg.data.decode())
                _enqueue({
                    "type": "health_update",
                    "payload": raw.get("payload", raw),
                    "timestamp": raw.get("timestamp", ""),
                })
            except Exception:
                logger.exception("Failed to forward health update to WebSocket")

        risk_sub = await nc.subscribe("pqap.risk.state_updated", cb=forward_risk_update)

        opp_detected_sub = await nc.subscribe("pqap.opportunity.detected", cb=forward_opportunity)
        opp_executed_sub = await nc.subscribe("pqap.opportunity.executed", cb=forward_opportunity)
        opp_filtered_sub = await nc.subscribe("pqap.opportunity.filtered", cb=forward_opportunity)

        # #2: Subscribe to health updates from system services
        health_sub = await nc.subscribe("pqap.system.health.>", cb=forward_health_update)

        ping_task = asyncio.create_task(send_ping())

        while True:
            data = await websocket.receive_text()
            if data == "ping":
                await websocket.send_text("pong")
                WS_MESSAGES_SENT.inc()
            elif data == "pong":
                pass

    except WebSocketDisconnect:
        logger.info("Dashboard WebSocket disconnected for user %s", user_id)
    except Exception:
        logger.exception("Dashboard WebSocket error for user %s", user_id)
    finally:
        if ping_task:
            ping_task.cancel()
        if send_task:
            send_task.cancel()
        if risk_sub:
            await risk_sub.unsubscribe()
        if opp_detected_sub:
            await opp_detected_sub.unsubscribe()
        if opp_executed_sub:
            await opp_executed_sub.unsubscribe()
        if opp_filtered_sub:
            await opp_filtered_sub.unsubscribe()
        if health_sub:
            await health_sub.unsubscribe()
        # #3: Shared NATS connection — do NOT close it here
        async with user_lock:
            _connection_counts[user_id] = max(0, _connection_counts.get(user_id, 1) - 1)
        WS_CONNECTIONS.dec()
