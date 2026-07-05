from datetime import datetime
from typing import Optional
from uuid import UUID

from pydantic import BaseModel, Field


class StrategyCreate(BaseModel):
    name: str = Field(min_length=1, max_length=255)
    description: str = Field(default="", max_length=2000)
    min_profit_threshold: float = Field(default=0.01, gt=0, le=1)
    score_threshold: float = Field(default=0.01, gt=0, le=1)
    max_position_size: float = Field(default=1000.0, gt=0)
    max_daily_trades: int = Field(default=50, gt=0, le=10000)
    risk_limit_pct: float = Field(default=5.0, gt=0, le=100)
    capital_weight: float = Field(default=100.0, ge=0, le=100)
    account_id: Optional[UUID] = None  # #9: Validate as UUID


class StrategyUpdate(BaseModel):
    name: Optional[str] = Field(default=None, min_length=1, max_length=255)
    description: Optional[str] = Field(default=None, max_length=2000)
    min_profit_threshold: Optional[float] = Field(default=None, gt=0, le=1)
    score_threshold: Optional[float] = Field(default=None, gt=0, le=1)
    max_position_size: Optional[float] = Field(default=None, gt=0)
    max_daily_trades: Optional[int] = Field(default=None, gt=0, le=10000)
    risk_limit_pct: Optional[float] = Field(default=None, gt=0, le=100)
    capital_weight: Optional[float] = Field(default=None, ge=0, le=100)


class StrategyResponse(BaseModel):
    id: str
    name: str
    description: str
    status: str
    min_profit_threshold: float
    score_threshold: float
    max_position_size: float
    max_daily_trades: int
    risk_limit_pct: float
    capital_weight: float
    account_id: Optional[str]
    created_at: datetime
    updated_at: datetime
    activated_at: Optional[datetime]


class StrategyListResponse(BaseModel):
    items: list[StrategyResponse]
    total: int


class VersionResponse(BaseModel):
    id: str
    strategy_id: str
    version_number: int
    parameters: dict
    change_summary: str
    changed_by: Optional[str]
    created_at: datetime


class VersionListResponse(BaseModel):
    items: list[VersionResponse]
    total: int


class WeightUpdateRequest(BaseModel):
    weights: dict[str, float]  # {strategy_id: weight}


class WeightUpdateResponse(BaseModel):
    weights: dict[str, float]
    total: float
    updated_count: int
