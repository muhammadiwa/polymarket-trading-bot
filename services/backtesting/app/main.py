import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI
from prometheus_client import make_asgi_app

from app.config import config
from app.db import close_pools, init_pools
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

metrics_app = make_asgi_app()
app.mount("/metrics", metrics_app)

app.include_router(backtest.router)
app.include_router(replay.router)


@app.get("/health")
async def health():
    return {"status": "ok"}
