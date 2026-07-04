import logging
import time
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, Query

from app.db import get_pool
from app.metrics import OPPORTUNITY_QUERY_TOTAL, OPPORTUNITY_QUERY_LATENCY
from app.middleware.auth import verify_jwt
from app.models.opportunity import OpportunityListResponse, OpportunityStatus
from app.repos.opportunity_repo import list_opportunities

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api", tags=["opportunities"])


@router.get("/opportunities", response_model=OpportunityListResponse)
async def get_opportunities(
    cursor: Optional[str] = Query(None, description="Cursor for pagination (ISO timestamp)"),
    page_size: int = Query(50, ge=1, le=100),
    status: Optional[str] = Query(None, description="Filter by status: detected, executed, filtered"),
    _user: dict = Depends(verify_jwt),
):
    # #9: Validate status_filter against enum
    status_filter = None
    if status is not None:
        try:
            status_filter = OpportunityStatus(status).value
        except ValueError:
            raise HTTPException(
                status_code=400,
                detail=f"Invalid status value: {status}. Must be one of: detected, executed, filtered",
            )

    start = time.monotonic()

    try:
        pool = await get_pool()
        async with pool.acquire() as conn:
            opportunities, total_count, next_cursor = await list_opportunities(
                conn,
                cursor=cursor,
                page_size=page_size,
                status_filter=status_filter,
            )

        elapsed = (time.monotonic() - start) * 1000
        OPPORTUNITY_QUERY_LATENCY.observe(elapsed)
        OPPORTUNITY_QUERY_TOTAL.inc()

        return OpportunityListResponse(
            opportunities=opportunities,
            total_count=total_count,
            next_cursor=next_cursor,
        )

    except HTTPException:
        raise
    except Exception:
        logger.exception("Failed to query opportunities")
        raise HTTPException(status_code=500, detail="Internal server error")
