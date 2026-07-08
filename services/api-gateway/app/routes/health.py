import asyncio
import json
import logging
import time
from datetime import datetime, timezone
from typing import Optional

import httpx
import redis.asyncio as aioredis
from fastapi import APIRouter, Depends, Query
from fastapi.websockets import WebSocket, WebSocketDisconnect

from app.config import config
from app.metrics import (
    HEALTH_POLL_TOTAL,
    HEALTH_POLL_LATENCY,
    HEALTH_STALE_TOTAL,
    ADMIN_HEALTH_CHECKS_TOTAL,
    ADMIN_HEALTH_CHECK_LATENCY,
    ADMIN_ACTIVE_ALERTS,
    ADMIN_WS_CONNECTIONS,
)
from app.middleware.auth import verify_jwt, extract_user, require_admin
from app.models.health import ServiceHealth, SystemHealthResponse

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/system", tags=["system-health"])

_redis_pool: aioredis.Redis | None = None
HTTP_TIMEOUT = httpx.Timeout(5.0, connect=2.0)

# #5: Module-level HTTP client with lifecycle management (avoids per-request allocation)
_http_client: httpx.AsyncClient | None = None


async def _get_http_client() -> httpx.AsyncClient:
    global _http_client
    if _http_client is None or _http_client.is_closed:
        _http_client = httpx.AsyncClient(timeout=HTTP_TIMEOUT)
    return _http_client


async def close_http_client() -> None:
    global _http_client
    if _http_client is not None:
        try:
            await _http_client.aclose()
        except Exception:
            logger.warning("Failed to close HTTP client")
        _http_client = None


# #16: Service URLs loaded from configuration (not hardcoded)
SERVICE_URLS = {
    "scanner": config.SCANNER_URL,
    "arb_engine": config.ARB_ENGINE_URL,
    "execution_engine": config.EXECUTION_ENGINE_URL,
    "risk_manager": config.RISK_MANAGER_URL,
    "position_manager": config.POSITION_MANAGER_URL,
}

SERVICE_NAMES = {
    "scanner": "Scanner",
    "arb_engine": "Arb Engine",
    "execution_engine": "Execution Engine",
    "risk_manager": "Risk Manager",
    "position_manager": "Position Manager",
}

REDIS_KEY = "pqap:system:health"
REDIS_TTL = 5

# Health thresholds for alerting
HEALTH_THRESHOLDS = {
    "cpu_percent": {"warning": 70.0, "critical": 90.0},
    "memory_percent": {"warning": 80.0, "critical": 95.0},
    "error_rate": {"warning": 5.0, "critical": 10.0},
}

# Active alerts storage (in-memory for now, can be moved to Redis)
_active_alerts: list[dict] = []
_alert_id_counter = 0


def _check_thresholds(service_name: str, metrics: dict) -> list[dict]:
    """Check if any metrics exceed thresholds and generate alerts."""
    global _alert_id_counter
    alerts = []

    # CPU check
    cpu = metrics.get("cpu_percent", 0.0)
    if cpu >= HEALTH_THRESHOLDS["cpu_percent"]["critical"]:
        _alert_id_counter += 1
        alerts.append({
            "id": f"alert-{_alert_id_counter}",
            "service": service_name,
            "metric": "cpu_percent",
            "threshold": HEALTH_THRESHOLDS["cpu_percent"]["critical"],
            "currentValue": cpu,
            "severity": "critical",
            "triggeredAt": datetime.now(timezone.utc).isoformat(),
            "message": f"{service_name}: CPU at {cpu:.1f}% (critical threshold: {HEALTH_THRESHOLDS['cpu_percent']['critical']}%)",
        })
    elif cpu >= HEALTH_THRESHOLDS["cpu_percent"]["warning"]:
        _alert_id_counter += 1
        alerts.append({
            "id": f"alert-{_alert_id_counter}",
            "service": service_name,
            "metric": "cpu_percent",
            "threshold": HEALTH_THRESHOLDS["cpu_percent"]["warning"],
            "currentValue": cpu,
            "severity": "warning",
            "triggeredAt": datetime.now(timezone.utc).isoformat(),
            "message": f"{service_name}: CPU at {cpu:.1f}% (warning threshold: {HEALTH_THRESHOLDS['cpu_percent']['warning']}%)",
        })

    # Memory check
    memory_mb = metrics.get("memory_mb", 0.0)
    memory_limit_mb = metrics.get("memory_limit_mb", 0.0)
    if memory_limit_mb > 0:
        memory_percent = (memory_mb / memory_limit_mb) * 100
        if memory_percent >= HEALTH_THRESHOLDS["memory_percent"]["critical"]:
            _alert_id_counter += 1
            alerts.append({
                "id": f"alert-{_alert_id_counter}",
                "service": service_name,
                "metric": "memory_percent",
                "threshold": HEALTH_THRESHOLDS["memory_percent"]["critical"],
                "currentValue": memory_percent,
                "severity": "critical",
                "triggeredAt": datetime.now(timezone.utc).isoformat(),
                "message": f"{service_name}: Memory at {memory_percent:.1f}% (critical threshold: {HEALTH_THRESHOLDS['memory_percent']['critical']}%)",
            })
        elif memory_percent >= HEALTH_THRESHOLDS["memory_percent"]["warning"]:
            _alert_id_counter += 1
            alerts.append({
                "id": f"alert-{_alert_id_counter}",
                "service": service_name,
                "metric": "memory_percent",
                "threshold": HEALTH_THRESHOLDS["memory_percent"]["warning"],
                "currentValue": memory_percent,
                "severity": "warning",
                "triggeredAt": datetime.now(timezone.utc).isoformat(),
                "message": f"{service_name}: Memory at {memory_percent:.1f}% (warning threshold: {HEALTH_THRESHOLDS['memory_percent']['warning']}%)",
            })

    # Error rate check
    error_rate = metrics.get("error_rate", 0.0)
    if error_rate >= HEALTH_THRESHOLDS["error_rate"]["critical"]:
        _alert_id_counter += 1
        alerts.append({
            "id": f"alert-{_alert_id_counter}",
            "service": service_name,
            "metric": "error_rate",
            "threshold": HEALTH_THRESHOLDS["error_rate"]["critical"],
            "currentValue": error_rate,
            "severity": "critical",
            "triggeredAt": datetime.now(timezone.utc).isoformat(),
            "message": f"{service_name}: Error rate at {error_rate:.1f}/min (critical threshold: {HEALTH_THRESHOLDS['error_rate']['critical']}/min)",
        })
    elif error_rate >= HEALTH_THRESHOLDS["error_rate"]["warning"]:
        _alert_id_counter += 1
        alerts.append({
            "id": f"alert-{_alert_id_counter}",
            "service": service_name,
            "metric": "error_rate",
            "threshold": HEALTH_THRESHOLDS["error_rate"]["warning"],
            "currentValue": error_rate,
            "severity": "warning",
            "triggeredAt": datetime.now(timezone.utc).isoformat(),
            "message": f"{service_name}: Error rate at {error_rate:.1f}/min (warning threshold: {HEALTH_THRESHOLDS['error_rate']['warning']}/min)",
        })

    return alerts


async def _get_redis() -> aioredis.Redis | None:
    global _redis_pool
    try:
        if _redis_pool is None:
            _redis_pool = aioredis.from_url(config.REDIS_URL, decode_responses=True)
        return _redis_pool
    except Exception:
        logger.warning("Failed to create Redis connection pool")
        return None


async def close_redis_pool() -> None:
    global _redis_pool
    if _redis_pool is not None:
        try:
            await _redis_pool.close()
        except Exception:
            logger.warning("Failed to close Redis pool")
        _redis_pool = None


def _empty_service_health(key: str) -> dict:
    return {
        "name": SERVICE_NAMES.get(key, key),
        "status": "down",
        "wsConnected": False,
        "cpuPercent": 0.0,
        "memoryMB": 0.0,
        "memoryLimitMB": 0.0,
        "errorRate": 0.0,
        "lastHeartbeat": None,
    }


async def _poll_service(client: httpx.AsyncClient, key: str, base_url: str) -> dict:
    try:
        resp = await client.get(f"{base_url}/health")
        resp.raise_for_status()
        data = resp.json()

        metrics = data.get("metrics", {})
        return {
            "name": SERVICE_NAMES.get(key, key),
            "status": "up",
            "wsConnected": metrics.get("ws_connected", False),
            "cpuPercent": float(metrics.get("cpu_percent", 0.0)),
            "memoryMB": float(metrics.get("memory_mb", 0.0)),
            "memoryLimitMB": float(metrics.get("memory_limit_mb", 0.0)),
            "errorRate": float(metrics.get("error_rate", 0.0)),
            "lastHeartbeat": data.get("timestamp", None),
        }
    except Exception:
        logger.debug("Health poll failed for %s", key)
        return _empty_service_health(key)


def _compute_overall(services: dict[str, dict]) -> str:
    # #24: Align with spec - any service down means unhealthy
    statuses = [s["status"] for s in services.values()]
    if all(s == "up" for s in statuses):
        return "healthy"
    if any(s == "down" for s in statuses):
        return "unhealthy"
    return "degraded"


async def _aggregate_health(include_alerts: bool = False) -> dict:
    r = await _get_redis()

    if r is not None and not include_alerts:
        try:
            cached = await r.get(REDIS_KEY)
            if cached:
                # #18: Only increment stale counter when cache age exceeds TTL
                result = json.loads(cached)
                last_updated = result.get("lastUpdated", "")
                if last_updated:
                    try:
                        cached_time = datetime.fromisoformat(last_updated)
                        age_seconds = (datetime.now(timezone.utc) - cached_time).total_seconds()
                        if age_seconds > REDIS_TTL:
                            HEALTH_STALE_TOTAL.inc()
                    except (ValueError, TypeError):
                        pass
                return result
        except Exception:
            logger.warning("Redis read failed, falling through to live poll")

    start = time.monotonic()

    services: dict[str, dict] = {}
    client = await _get_http_client()
    tasks = {
        key: asyncio.create_task(_poll_service(client, key, url))
        for key, url in SERVICE_URLS.items()
    }
    for key, task in tasks.items():
        services[key] = await task

    elapsed = (time.monotonic() - start) * 1000
    HEALTH_POLL_LATENCY.observe(elapsed)
    HEALTH_POLL_TOTAL.inc()

    # Check thresholds and generate alerts
    all_alerts = []
    for key, service_data in services.items():
        if service_data.get("status") == "up":
            alerts = _check_thresholds(service_data["name"], service_data)
            all_alerts.extend(alerts)

    _update_active_alerts(all_alerts)

    overall = _compute_overall(services)

    now = datetime.now(timezone.utc).isoformat()

    result = {
        "scanner": services.get("scanner", _empty_service_health("scanner")),
        "arbEngine": services.get("arb_engine", _empty_service_health("arb_engine")),
        "executionEngine": services.get("execution_engine", _empty_service_health("execution_engine")),
        "riskManager": services.get("risk_manager", _empty_service_health("risk_manager")),
        "positionManager": services.get("position_manager", _empty_service_health("position_manager")),
        "overall": overall,
        "lastUpdated": now,
    }

    if include_alerts:
        result["alerts"] = _active_alerts

    if r is not None and not include_alerts:
        try:
            await r.set(REDIS_KEY, json.dumps(result), ex=REDIS_TTL)
        except Exception:
            logger.warning("Redis write failed")

    return result


@router.get("/health", response_model=SystemHealthResponse)
async def get_system_health(_user: dict = Depends(verify_jwt)):
    return await _aggregate_health()


# ─────────────────────────────────────────────────────────────────────────────
# Admin Health Endpoints
# ─────────────────────────────────────────────────────────────────────────────


# Admin health endpoints use /api/admin prefix (registered separately in main.py)
admin_health_router = APIRouter(prefix="/api/admin", tags=["admin-health"])


@admin_health_router.get("/health")
async def get_admin_health(user: dict = Depends(extract_user)):
    """Get aggregated health status with alerts. Requires admin role."""
    require_admin(user)
    ADMIN_HEALTH_CHECKS_TOTAL.inc()

    start = time.monotonic()
    result = await _aggregate_health(include_alerts=True)
    elapsed = (time.monotonic() - start) * 1000
    ADMIN_HEALTH_CHECK_LATENCY.observe(elapsed)

    return result


@admin_health_router.get("/health/services")
async def get_admin_health_services(user: dict = Depends(extract_user)):
    """Get per-service health details. Requires admin role."""
    require_admin(user)

    result = await _aggregate_health(include_alerts=False)
    services = [
        result.get("scanner"),
        result.get("arbEngine"),
        result.get("executionEngine"),
        result.get("riskManager"),
        result.get("positionManager"),
    ]
    return {"services": services, "overall": result.get("overall"), "lastUpdated": result.get("lastUpdated")}


@admin_health_router.get("/health/alerts")
async def get_admin_health_alerts(user: dict = Depends(extract_user)):
    """Get active health alerts. Requires admin role."""
    require_admin(user)
    return {"alerts": _active_alerts, "total": len(_active_alerts)}


# WebSocket connection manager for real-time health updates
_admin_ws_clients: set[WebSocket] = set()

# Redis key for alerts (shared across workers)
REDIS_ALERTS_KEY = "pqap:admin:alerts"


async def _save_alerts_to_redis(alerts: list[dict]) -> None:
    """Save alerts to Redis for cross-worker consistency."""
    r = await _get_redis()
    if r is not None:
        try:
            await r.set(REDIS_ALERTS_KEY, json.dumps(alerts), ex=300)  # 5 min TTL
        except Exception:
            logger.warning("Failed to save alerts to Redis")


async def _load_alerts_from_redis() -> list[dict]:
    """Load alerts from Redis."""
    r = await _get_redis()
    if r is not None:
        try:
            data = await r.get(REDIS_ALERTS_KEY)
            if data:
                return json.loads(data)
        except Exception:
            logger.warning("Failed to load alerts from Redis")
    return []


def _update_active_alerts(new_alerts: list[dict]) -> None:
    """Update active alerts list, removing old alerts for same service/metric."""
    global _active_alerts
    # Remove old alerts for services that are now healthy
    services_with_alerts = {(a["service"], a["metric"]) for a in new_alerts}
    _active_alerts = [
        a for a in _active_alerts
        if (a["service"], a["metric"]) in services_with_alerts
    ]
    # Add new alerts
    _active_alerts.extend(new_alerts)
    ADMIN_ACTIVE_ALERTS.set(len(_active_alerts))


async def broadcast_health_update(health_data: dict) -> None:
    """Broadcast health update to all connected admin WebSocket clients."""
    message = json.dumps({
        "type": "health_update",
        "payload": health_data,
        "timestamp": datetime.now(timezone.utc).isoformat(),
    })
    disconnected = set()
    for ws in _admin_ws_clients:
        try:
            await ws.send_text(message)
        except (WebSocketDisconnect, RuntimeError):
            disconnected.add(ws)
        except Exception as e:
            logger.warning(f"Failed to send WebSocket message: {e}")
            disconnected.add(ws)
    _admin_ws_clients.difference_update(disconnected)


ws_router = APIRouter(tags=["admin-websocket"])


@ws_router.websocket("/ws/admin/health")
async def admin_health_websocket(websocket: WebSocket):
    """WebSocket endpoint for real-time health updates. Requires JWT auth."""
    # Don't accept until authenticated - close with 4001 if no auth within 5 seconds
    try:
        # Wait for auth message with timeout
        auth_data = await asyncio.wait_for(websocket.receive_text(), timeout=5.0)
        auth_msg = json.loads(auth_data)

        # Validate JWT token
        token = auth_msg.get("token")
        if not token:
            await websocket.close(code=4001, reason="Missing auth token")
            return

        try:
            from app.middleware.auth import decode_jwt, validate_jwt_claims
            from jose import JWTError
            payload = decode_jwt(token)
            user = validate_jwt_claims(payload)
            if user.get("role") != "admin":
                await websocket.close(code=4003, reason="Admin access required")
                return
        except (JWTError, HTTPException) as e:
            await websocket.close(code=4001, reason=f"Invalid token: {str(e)}")
            return

        # Now accept the connection after successful auth
        await websocket.accept()
        ADMIN_WS_CONNECTIONS.inc()

        # Add to active connections
        _admin_ws_clients.add(websocket)

        # Send initial health data
        health_data = await _aggregate_health(include_alerts=True)
        await websocket.send_text(json.dumps({
            "type": "health_update",
            "payload": health_data,
            "timestamp": datetime.now(timezone.utc).isoformat(),
        }))

        # Keep connection alive and handle messages
        while True:
            try:
                # Wait for ping or close
                data = await websocket.receive_text()
                if data == "ping":
                    await websocket.send_text("pong")
            except WebSocketDisconnect:
                break
            except asyncio.TimeoutError:
                # Send periodic health updates
                health_data = await _aggregate_health(include_alerts=True)
                await websocket.send_text(json.dumps({
                    "type": "health_update",
                    "payload": health_data,
                    "timestamp": datetime.now(timezone.utc).isoformat(),
                }))
            except RuntimeError as e:
                if "disconnect" in str(e).lower():
                    break
                raise
    except asyncio.TimeoutError:
        # No auth received within timeout
        try:
            await websocket.close(code=4001, reason="Authentication timeout")
        except Exception:
            pass
        return
    except WebSocketDisconnect:
        return
    except Exception as e:
        logger.error(f"WebSocket error: {e}")
        try:
            await websocket.close(code=1011, reason="Internal server error")
        except Exception:
            pass
    finally:
        _admin_ws_clients.discard(websocket)
        try:
            ADMIN_WS_CONNECTIONS.dec()
        except Exception:
            pass
