from datetime import datetime
from decimal import Decimal
from enum import Enum
from typing import Optional

from pydantic import BaseModel, ConfigDict, Field
from pydantic.alias_generators import to_camel


class OpportunityStatus(str, Enum):
    DETECTED = "detected"
    EXECUTED = "executed"
    FILTERED = "filtered"


class OpportunityResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True, json_encoders={Decimal: str})

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


class OpportunityListResponse(BaseModel):
    opportunities: list[OpportunityResponse]
    total_count: int
    next_cursor: Optional[str] = None
