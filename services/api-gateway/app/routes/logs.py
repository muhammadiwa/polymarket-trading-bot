import logging
import time
from datetime import datetime, timezone
from typing import Optional

from fastapi import APIRouter, Depends, Header, HTTPException, Query, status

from app.config import config
from app.db import get_pool
from app.middleware.auth import extract_user, require_admin
from app.metrics import ADMIN_LOG_QUERIES_TOTAL, ADMIN_LOG_QUERY_LATENCY, ADMIN_LOG_INGESTION_TOTAL
from app.models.logs import (
    LogLevel,
    LogQueryResponse,
    SystemLogCreate,
    SystemLogResponse,
)

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/admin/logs", tags=["admin-logs"])


def _verify_internal_key(x_internal_key: str = Header(None)) -> bool:
    """Verify internal API key for service-to-service communication."""
    if not config.INTERNAL_API_KEY:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Internal API key not configured",
        )
    if not x_internal_key or x_internal_key != config.INTERNAL_API_KEY:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid internal API key",
        )
    return True


@router.post("", response_model=SystemLogResponse)
async def ingest_log(
    body: SystemLogCreate,
    _auth: bool = Depends(_verify_internal_key),
):
    """Ingest a log entry. Requires internal API key."""
    ADMIN_LOG_INGESTION_TOTAL.inc()

    pool = await get_pool()
    async with pool.acquire() as conn:
        row = await conn.fetchrow(
            """
            INSERT INTO system_logs (level, service, request_id, message, context)
            VALUES ($1, $2, $3, $4, $5)
            RETURNING id, timestamp, level, service, request_id, message, context
            """,
            body.level.value,
            body.service,
            body.request_id,
            body.message,
            body.context,
        )

    return SystemLogResponse(
        id=row["id"],
        timestamp=row["timestamp"],
        level=row["level"],
        service=row["service"],
        request_id=row["request_id"],
        message=row["message"],
        context=row["context"],
    )


@router.get("", response_model=LogQueryResponse)
async def query_logs(
    level: Optional[str] = Query(None, pattern=r"^(debug|info|warn|error|fatal)$"),
    service: Optional[str] = Query(None, max_length=50),
    start_date: Optional[datetime] = Query(None),
    end_date: Optional[datetime] = Query(None),
    search: Optional[str] = Query(None, max_length=500),
    limit: int = Query(100, ge=1, le=1000),
    offset: int = Query(0, ge=0),
    user: dict = Depends(extract_user),
):
    """Query logs with filters. Requires admin role."""
    require_admin(user)
    ADMIN_LOG_QUERIES_TOTAL.inc()

    start_time = time.monotonic()

    pool = await get_pool()
    async with pool.acquire() as conn:
        # Build dynamic query
        conditions = []
        params = []
        param_idx = 1

        if level:
            conditions.append(f"level = ${param_idx}")
            params.append(level)
            param_idx += 1

        if service:
            conditions.append(f"service = ${param_idx}")
            params.append(service)
            param_idx += 1

        if start_date:
            conditions.append(f"timestamp >= ${param_idx}")
            params.append(start_date)
            param_idx += 1

        if end_date:
            conditions.append(f"timestamp <= ${param_idx}")
            params.append(end_date)
            param_idx += 1

        if search:
            conditions.append(f"message ILIKE ${param_idx}")
            params.append(f"%{search}%")
            param_idx += 1

        where_clause = " AND ".join(conditions) if conditions else "1=1"

        # Get total count
        count_query = f"SELECT COUNT(*) as total FROM system_logs WHERE {where_clause}"
        count_row = await conn.fetchrow(count_query, *params)
        total = count_row["total"]

        # Get logs
        query = f"""
            SELECT id, timestamp, level, service, request_id, message, context
            FROM system_logs
            WHERE {where_clause}
            ORDER BY timestamp DESC
            LIMIT ${param_idx} OFFSET ${param_idx + 1}
        """
        params.extend([limit, offset])
        rows = await conn.fetch(query, *params)

    elapsed = (time.monotonic() - start_time) * 1000
    ADMIN_LOG_QUERY_LATENCY.observe(elapsed)

    logs = [
        SystemLogResponse(
            id=row["id"],
            timestamp=row["timestamp"],
            level=row["level"],
            service=row["service"],
            request_id=row["request_id"],
            message=row["message"],
            context=row["context"],
        )
        for row in rows
    ]

    return LogQueryResponse(
        logs=logs,
        total=total,
        has_more=(offset + limit) < total,
    )


@router.get("/services", response_model=list[str])
async def list_services(user: dict = Depends(extract_user)):
    """List available services from logs. Requires admin role."""
    require_admin(user)

    pool = await get_pool()
    async with pool.acquire() as conn:
        rows = await conn.fetch(
            "SELECT DISTINCT service FROM system_logs ORDER BY service"
        )

    return [row["service"] for row in rows]
