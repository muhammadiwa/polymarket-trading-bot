import asyncio
import json
import logging
import time
from datetime import datetime, timezone

import httpx
import redis.asyncio as aioredis
from fastapi import APIRouter, Depends

from app.config import config
from app.metrics import HEALTH_POLL_TOTAL, HEALTH_POLL_LATENCY, HEALTH_STALE_TOTAL
from app.middleware.auth import verify_jwt
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


async def _aggregate_health() -> dict:
    r = await _get_redis()

    if r is not None:
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

    if r is not None:
        try:
            await r.set(REDIS_KEY, json.dumps(result), ex=REDIS_TTL)
        except Exception:
            logger.warning("Redis write failed")

    return result


@router.get("/health", response_model=SystemHealthResponse)
async def get_system_health(_user: dict = Depends(verify_jwt)):
    return await _aggregate_health()
