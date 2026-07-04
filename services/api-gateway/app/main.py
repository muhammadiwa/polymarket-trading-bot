import logging
import uuid

from fastapi import FastAPI, Request
from starlette.middleware.base import BaseHTTPMiddleware

from app.db import close_pool
from app.routes.trades import router as trades_router

logging.basicConfig(
    level=logging.INFO,
    format='{"level":"%(levelname)s","service":"api-gateway","msg":"%(message)s"}',
)

app = FastAPI(
    title="PQAP API Gateway",
    version="1.0.0",
    docs_url="/docs",
    redoc_url="/redoc",
)

# TODO(#19): Add rate limiting and authentication middleware before Epic 2.
# Currently the API gateway is publicly accessible without rate limits or auth.


class RequestIDMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next):
        request_id = request.headers.get("X-Request-ID", str(uuid.uuid4()))
        request.state.request_id = request_id
        response = await call_next(request)
        response.headers["X-Request-ID"] = request_id
        return response


app.add_middleware(RequestIDMiddleware)

app.include_router(trades_router)


@app.on_event("shutdown")
async def shutdown():
    await close_pool()


@app.get("/health")
async def health():
    return {"status": "ok"}
