import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from prometheus_client import make_asgi_app

from app.config import config
from app.db import close_pool, init_pool
from app.routes import portfolio
from app.services.nats_publisher import init_nats, close_nats

logging.basicConfig(level=getattr(logging, config.LOG_LEVEL.upper(), logging.INFO))
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    await init_pool()
    await init_nats()
    logger.info("portfolio-manager service started")
    yield
    await close_nats()
    await close_pool()
    logger.info("portfolio-manager service stopped")


app = FastAPI(title="Portfolio Manager", version="1.0.0", lifespan=lifespan)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:3000", "http://localhost:8080"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

metrics_app = make_asgi_app()
app.mount("/metrics", metrics_app)

app.include_router(portfolio.router)


@app.get("/health")
async def health():
    from app.services.nats_publisher import _nats_client
    checks = {"status": "ok"}
    if _nats_client and _nats_client.is_connected:
        checks["nats"] = "connected"
    else:
        checks["nats"] = "disconnected"
        checks["status"] = "degraded"
    return checks
