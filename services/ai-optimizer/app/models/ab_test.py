from datetime import datetime
from decimal import Decimal
from typing import Optional

from pydantic import BaseModel, ConfigDict, Field
from pydantic.alias_generators import to_camel


class StartABTestRequest(BaseModel):
    min_sample_size: int = Field(default=50, ge=10, le=1000)


class ABTestResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    id: str
    suggestion_id: str
    strategy_id: str
    status: str
    min_sample_size: int
    current_sample_size: int
    p_value: Optional[float]
    mean_difference: Optional[float]
    recommendation: Optional[str]
    started_at: datetime
    completed_at: Optional[datetime]
    failed_reason: Optional[str]
    created_at: datetime


class VariantStats(BaseModel):
    count: int
    mean_pnl: Decimal
    std_pnl: Decimal


class ABTestResultSummary(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    ab_test_id: str
    control: VariantStats
    treatment: VariantStats
    p_value: float
    mean_difference: Decimal
    is_significant: bool
    recommendation: str


class OverfittingAnalysisResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    suggestion_id: str
    overfitting_score: Optional[Decimal]
    in_sample_win_rate: Optional[Decimal]
    out_of_sample_win_rate: Optional[Decimal]
    degradation_pct: Optional[Decimal]
    is_overfitting: bool
    warning: Optional[str]
