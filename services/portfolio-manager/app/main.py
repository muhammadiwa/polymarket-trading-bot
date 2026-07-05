import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI
from prometheus_client import make_asgi_app

from app.config import config
from app.db import close_pool, init_pool
from app.routes import portfolio

logging.basicConfig(level=getattr(logging, config.LOG_LEVEL.upper(), logging.INFO))
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    await init_pool()
    logger.info("portfolio-manager service started")
    yield
    await close_pool()
    logger.info("portfolio-manager service stopped")


app = FastAPI(title="Portfolio Manager", version="1.0.0", lifespan=lifespan)

metrics_app = make_asgi_app()
app.mount("/metrics", metrics_app)

app.include_router(portfolio.router)


@app.get("/health")
async def health():
    return {"status": "ok"}
