import logging
import os
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from prometheus_client import make_asgi_app

from app.config import config
from app.db import close_pools, get_pg_pool, get_ts_pool, init_pools
from app.routes import backtest, replay

logging.basicConfig(level=getattr(logging, config.LOG_LEVEL.upper(), logging.INFO))
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    await init_pools()
    logger.info("backtesting service started")
    yield
    await close_pools()
    logger.info("backtesting service stopped")


app = FastAPI(title="Backtesting", version="1.0.0", lifespan=lifespan)

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

app.include_router(backtest.router)
app.include_router(replay.router)


@app.get("/health")
async def health():
    checks = {"status": "ok"}
    try:
        pool = await get_pg_pool()
        async with pool.acquire() as conn:
            await conn.fetchval("SELECT 1")
        checks["postgres"] = "ok"
    except Exception:
        checks["postgres"] = "error"
        checks["status"] = "degraded"
    try:
        pool = await get_ts_pool()
        async with pool.acquire() as conn:
            await conn.fetchval("SELECT 1")
        checks["timescaledb"] = "ok"
    except Exception:
        checks["timescaledb"] = "error"
        checks["status"] = "degraded"
    try:
        import redis.asyncio as aioredis
        r = aioredis.from_url(config.REDIS_URL, decode_responses=True)
        await r.ping()
        await r.aclose()
        checks["redis"] = "ok"
    except Exception:
        checks["redis"] = "error"
        checks["status"] = "degraded"
    return checks
