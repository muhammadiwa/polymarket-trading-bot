import asyncio
import json
import logging
from contextlib import asynccontextmanager
from datetime import datetime, timezone
from uuid import uuid4

import nats
from fastapi import FastAPI
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
                    # #1: Single NATS connection for all anomalies in this batch
                    nc = None
                    try:
                        nc = await nats.connect(config.NATS_URL)
                        for anomaly in anomalies:
                            anomaly_id = await analytics_repo.log_anomaly(conn, anomaly)
                            if anomaly_id:
                                ANOMALIES_DETECTED.inc()
                                logger.warning("anomaly detected",
                                    extra={"anomaly_id": anomaly_id, "type": anomaly["anomaly_type"],
                                           "severity": anomaly["severity"], "metric": anomaly["metric_name"]})

                                # #2: Publish with guaranteed close
                                event = {
                                    "event_id": str(uuid4()),
                                    "event_type": "AnomalyDetected",
                                    "timestamp": datetime.now(timezone.utc).isoformat(),
                                    "source": "analytics",
                                    "payload": {**anomaly, "id": anomaly_id},
                                }
                                await nc.publish("pqap.analytics.anomaly", json.dumps(event).encode())
                                ANOMALY_ALERTS_SENT.inc()
                    except Exception as e:
                        logger.error("failed to publish anomaly alert", exc_info=e)
                    finally:
                        # #2: Guarantee connection close
                        if nc:
                            try:
                                await nc.close()
                            except Exception:
                                pass

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
    await close_pool()
    logger.info("analytics service stopped")


app = FastAPI(title="Analytics", version="1.0.0", lifespan=lifespan)

metrics_app = make_asgi_app()
app.mount("/metrics", metrics_app)

app.include_router(analytics.router)


@app.get("/health")
async def health():
    return {"status": "ok"}
