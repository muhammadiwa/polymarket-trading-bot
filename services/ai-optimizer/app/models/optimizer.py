from datetime import datetime
from decimal import Decimal
from typing import Optional

from pydantic import BaseModel, ConfigDict, Field
from pydantic.alias_generators import to_camel


class AnalyzeRequest(BaseModel):
    strategy_id: str = Field(min_length=1, max_length=64)
    min_trades: int = Field(default=100, ge=10)


class SuggestionResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    id: str
    strategy_id: str
    pattern_type: str
    parameter_name: str
    current_value: str
    suggested_value: str
    expected_impact: str
    methodology: str
    confidence: float
    p_value: Optional[float]
    status: str
    reviewed_by: Optional[str]
    reviewed_at: Optional[datetime]
    created_at: datetime
    overfitting_score: Optional[Decimal] = None
    out_of_sample_win_rate: Optional[Decimal] = None
    in_sample_win_rate: Optional[Decimal] = None
    degradation_pct: Optional[Decimal] = None


class SuggestionListResponse(BaseModel):
    suggestions: list[SuggestionResponse]
    total: int


class AnalysisResult(BaseModel):
    patterns_found: int
    suggestions_generated: int
    strategy_id: str
