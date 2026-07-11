import asyncio
import json
import logging
import os
from contextlib import asynccontextmanager
from datetime import datetime, timezone
from uuid import uuid4

import nats
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from prometheus_client import Counter, Histogram, make_asgi_app

from app.config import config
from app.db import close_pool, get_pool, init_pool
from app.repos import analytics_repo
from app.routes import analytics

logging.basicConfig(level=getattr(logging, config.LOG_LEVEL.upper(), logging.INFO))
logger = logging.getLogger(__name__)

# Prometheus metrics
ANOMALIES_DETECTED = Counter("pqap_anomalies_detected_total", "Total anomalies detected")
ANOMALY_CHECK_LATENCY = Histogram("pqap_anomaly_check_latency_ms", "Anomaly check latency")
ANOMALY_ALERTS_SENT = Counter("pqap_anomaly_alerts_sent_total", "Anomaly alerts sent")


# Persistent NATS connection for anomaly publishing
_nc: nats.NATS | None = None


async def _get_nats() -> nats.NATS:
    global _nc
    if _nc is None or _nc.is_closed:
        _nc = await nats.connect(config.NATS_URL)
    return _nc


async def anomaly_check_loop():
    """Background task: check for anomalies every ANOMALY_CHECK_INTERVAL seconds."""
    await asyncio.sleep(10)  # Initial delay for service startup
    while True:
        try:
            pool = await get_pool()
            async with pool.acquire() as conn:
                thresholds = {
                    "win_rate_drop": config.ANOMALY_WIN_RATE_DROP,
                    "drawdown_multiplier": config.ANOMALY_DRAWDOWN_MULTIPLIER,
                    "consecutive_losses": config.ANOMALY_CONSECUTIVE_LOSSES,
                    "profit_factor_low": config.ANOMALY_PROFIT_FACTOR_LOW,
                    "sharpe_low": config.ANOMALY_SHARPE_LOW,
                }
                anomalies = await analytics_repo.detect_anomalies(conn, thresholds)

                if anomalies:
                    nc = await _get_nats()
                    try:
                        for anomaly in anomalies:
                            anomaly_id = await analytics_repo.log_anomaly(conn, anomaly)
                            if anomaly_id:
                                ANOMALIES_DETECTED.inc()
                                logger.warning("anomaly detected",
                                    extra={"anomaly_id": anomaly_id, "type": anomaly["anomaly_type"],
                                           "severity": anomaly["severity"], "metric": anomaly["metric_name"]})

                                # Publish anomaly event
                                event = {
                                    "event_id": str(uuid4()),
                                    "event_type": "AnomalyDetected",
                                    "timestamp": datetime.now(timezone.utc).isoformat(),
                                    "source": "analytics",
                                    "payload": {**anomaly, "id": anomaly_id},
                                }
                                await nc.publish("pqap.analytics.anomaly", json.dumps(event).encode())

                                # #6: Also publish to notification service for Telegram delivery
                                severity_map = {"critical": "critical", "high": "high", "medium": "warning", "low": "info"}
                                notif_event = {
                                    "event_id": str(uuid4()),
                                    "event_type": "NotificationRequest",
                                    "timestamp": datetime.now(timezone.utc).isoformat(),
                                    "source": "analytics",
                                    "payload": {
                                        "category": "risk",
                                        "title": f"Anomaly Detected: {anomaly['anomaly_type']}",
                                        "message": f"{anomaly['metric_name']}: threshold={anomaly['threshold_value']}, actual={anomaly['actual_value']}",
                                        "channel": "telegram",
                                        "priority": severity_map.get(anomaly["severity"], "info"),
                                        "bypass_throttle": anomaly["severity"] in ("critical", "high"),
                                    },
                                }
                                await nc.publish("pqap.notification.request", json.dumps(notif_event).encode())
                                ANOMALY_ALERTS_SENT.inc()
                    except Exception as e:
                        logger.error("failed to publish anomaly alert", exc_info=e)

        except Exception as e:
            logger.error("anomaly check failed", exc_info=e)

        await asyncio.sleep(config.ANOMALY_CHECK_INTERVAL)


@asynccontextmanager
async def lifespan(app: FastAPI):
    await init_pool()
    # Start background anomaly checker
    task = asyncio.create_task(anomaly_check_loop())
    # #6: Add exception callback so task failures are logged
    def _on_task_done(t):
        if not t.cancelled() and t.exception():
            logger.error("anomaly check task died", exc_info=t.exception())
    task.add_done_callback(_on_task_done)
    logger.info("analytics service started")
    yield
    task.cancel()
    # Close persistent NATS connection
    global _nc
    if _nc and not _nc.is_closed:
        await _nc.close()
        _nc = None
    await close_pool()
    logger.info("analytics service stopped")


app = FastAPI(title="Analytics", version="1.0.0", lifespan=lifespan)

CORS_ORIGINS = [o.strip() for o in os.getenv("CORS_ORIGINS", "http://localhost:3000").split(",") if o.strip()]

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=CORS_ORIGINS,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

metrics_app = make_asgi_app()
app.mount("/metrics", metrics_app)

app.include_router(analytics.router)


@app.get("/health")
async def health():
    """Health check with dependency verification."""
    checks = {"status": "ok"}
    try:
        pool = await get_pool()
        async with pool.acquire() as conn:
            await conn.fetchval("SELECT 1")
        checks["database"] = "ok"
    except Exception:
        checks["database"] = "error"
        checks["status"] = "degraded"
    try:
        nc = await _get_nats()
        if nc.is_connected:
            checks["nats"] = "ok"
        else:
            checks["nats"] = "disconnected"
            checks["status"] = "degraded"
    except Exception:
        checks["nats"] = "error"
        checks["status"] = "degraded"
    return checks
