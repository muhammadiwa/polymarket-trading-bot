import logging
import os
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from prometheus_client import make_asgi_app

from app.config import config
from app.db import close_pool, get_pool, init_pool
from app.routes import assistant

logging.basicConfig(level=getattr(logging, config.LOG_LEVEL.upper(), logging.INFO))
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    await init_pool()
    logger.info("ai-assistant service started")
    yield
    # Close LLM client
    from app.routes.assistant import llm
    await llm.close()
    await close_pool()
    logger.info("ai-assistant service stopped")


app = FastAPI(title="AI Assistant", version="1.0.0", lifespan=lifespan)

# CORS configuration — restrict origins in production
_allowed_origins = ["http://localhost:3000", "http://localhost:8080"]
if os.getenv("CORS_ORIGINS"):
    _allowed_origins = os.getenv("CORS_ORIGINS").split(",")

app.add_middleware(
    CORSMiddleware,
    allow_origins=_allowed_origins,
    allow_credentials=True,
    allow_methods=["GET", "POST", "PUT", "DELETE"],
    allow_headers=["Authorization", "Content-Type"],
)

metrics_app = make_asgi_app()
app.mount("/metrics", metrics_app)

app.include_router(assistant.router)


@app.get("/health")
async def health():
    """Health check with dependency verification."""
    checks = {"status": "ok"}

    # Check database connectivity
    try:
        pool = await get_pool()
        async with pool.acquire() as conn:
            await conn.fetchval("SELECT 1")
        checks["database"] = "ok"
    except Exception as e:
        checks["database"] = "error"
        checks["status"] = "degraded"
        logger.warning("health check: database error", extra={"error": str(e)})

    return checks
