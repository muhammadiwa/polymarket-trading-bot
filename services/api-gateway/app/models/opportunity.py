from datetime import datetime
from decimal import Decimal
from enum import Enum
from typing import Optional

from pydantic import BaseModel, Field


class OpportunityStatus(str, Enum):
    DETECTED = "detected"
    EXECUTED = "executed"
    FILTERED = "filtered"


class OpportunityResponse(BaseModel):
    id: str
    market: str
    market_slug: str
    score: str
    spread: str
    fill_probability: str
    timestamp: str
    status: OpportunityStatus
    filter_reason: Optional[str] = None
    execution_latency_ms: Optional[int] = None

    model_config = {"json_encoders": {Decimal: str}}


class OpportunityListResponse(BaseModel):
    opportunities: list[OpportunityResponse]
    total_count: int
    next_cursor: Optional[str] = None
