import logging
import os
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from prometheus_client import make_asgi_app

from app.config import config
from app.db import close_pool, get_pool, init_pool
from app.events import StrategyEventPublisher
from app.routes import strategies

logging.basicConfig(level=getattr(logging, config.LOG_LEVEL.upper(), logging.INFO))
logger = logging.getLogger(__name__)

event_publisher = StrategyEventPublisher(config.NATS_URL)


@asynccontextmanager
async def lifespan(app: FastAPI):
    await init_pool()
    await event_publisher.connect()
    strategies.set_event_publisher(event_publisher)
    logger.info("strategy-manager service started")
    yield
    await event_publisher.close()
    await close_pool()
    logger.info("strategy-manager service stopped")


app = FastAPI(title="Strategy Manager", version="1.0.0", lifespan=lifespan)

CORS_ORIGINS = [o.strip() for o in os.getenv("CORS_ORIGINS", "http://localhost:3000").split(",") if o.strip()]

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=CORS_ORIGINS,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Mount Prometheus metrics
metrics_app = make_asgi_app()
app.mount("/metrics", metrics_app)

# Include routes
app.include_router(strategies.router)


@app.get("/health")
async def health():
    checks = {"status": "ok"}
    try:
        pool = await get_pool()
        async with pool.acquire() as conn:
            await conn.fetchval("SELECT 1")
        checks["database"] = "ok"
    except Exception:
        checks["database"] = "error"
        checks["status"] = "degraded"
    return checks
