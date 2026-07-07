import logging
import os
import uuid
from contextlib import asynccontextmanager

from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from starlette.middleware.base import BaseHTTPMiddleware

from app.config import config
from app.db import close_pool
from app.routes.portfolio import router as portfolio_router
from app.routes.risk import router as risk_router
from app.routes.trades import router as trades_router
from app.routes.ws import router as ws_router
from app.routes.health import router as health_router
from app.routes.health import close_redis_pool, close_http_client
from app.routes.notifications import router as notifications_router
from app.routes.opportunities import router as opportunities_router
from app.routes.auth import router as auth_router
from app.routes.admin import router as admin_router
from app.routes.orderbook import router as orderbook_router, init_client as init_orderbook_client, close_client as close_orderbook_client
from app.middleware.csrf import CSRFMiddleware

logging.basicConfig(
    level=logging.INFO,
    format='{"level":"%(levelname)s","service":"api-gateway","msg":"%(message)s"}',
)

ENVIRONMENT = os.getenv("ENVIRONMENT", "development")


@asynccontextmanager
async def lifespan(app: FastAPI):
    await init_orderbook_client()
    yield
    await close_orderbook_client()
    await close_http_client()
    await close_redis_pool()
    await close_pool()


app = FastAPI(
    title="PQAP API Gateway",
    version="1.0.0",
    docs_url="/docs" if ENVIRONMENT != "production" else None,
    redoc_url="/redoc" if ENVIRONMENT != "production" else None,
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=config.CORS_ORIGINS,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


class RequestIDMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next):
        request_id = request.headers.get("X-Request-ID", str(uuid.uuid4()))
        request.state.request_id = request_id
        response = await call_next(request)
        response.headers["X-Request-ID"] = request_id
        return response


app.add_middleware(RequestIDMiddleware)
app.add_middleware(CSRFMiddleware)

app.include_router(auth_router)
app.include_router(admin_router)
app.include_router(trades_router)
app.include_router(portfolio_router)
app.include_router(risk_router)
app.include_router(ws_router)
app.include_router(health_router)
app.include_router(opportunities_router)
app.include_router(notifications_router)
app.include_router(orderbook_router)


@app.get("/health")
async def health():
    return {"status": "ok"}
