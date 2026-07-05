import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI
from prometheus_client import make_asgi_app

from app.config import config
from app.db import close_pool, init_pool
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

# Mount Prometheus metrics
metrics_app = make_asgi_app()
app.mount("/metrics", metrics_app)

# Include routes
app.include_router(strategies.router)


@app.get("/health")
async def health():
    return {"status": "ok"}
