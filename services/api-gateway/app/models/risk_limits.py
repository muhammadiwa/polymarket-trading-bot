from datetime import datetime
from typing import Literal, Optional
from uuid import UUID

from pydantic import BaseModel, ConfigDict, Field
from pydantic.alias_generators import to_camel


class RiskLimitsResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    account_id: UUID
    daily_loss_limit: str = Field(..., max_length=20)
    max_position_per_market: str = Field(..., max_length=20)
    max_position_per_strategy: str = Field(..., max_length=20)
    drawdown_threshold: str = Field(..., max_length=20)


class RiskLimitsUpdate(BaseModel):
    daily_loss_limit: Optional[str] = Field(None, pattern=r"^\d+(\.\d{1,4})?$")
    max_position_per_market: Optional[str] = Field(None, pattern=r"^\d+(\.\d{1,4})?$")
    max_position_per_strategy: Optional[str] = Field(None, pattern=r"^\d+(\.\d{1,4})?$")
    drawdown_threshold: Optional[str] = Field(None, pattern=r"^\d+(\.\d{1,4})?$")


class AccountPortfolioSummary(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    account_id: UUID
    account_name: str
    capital: str
    daily_pnl: str
    total_pnl: str
    position_count: int
    utilization_rate: str
    is_active: bool


class CrossAccountPortfolioResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    total_capital: str
    total_daily_pnl: str
    total_pnl: str
    total_positions: int
    accounts: list[AccountPortfolioSummary]
    last_updated: datetime


class AccountRiskSummary(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    account_id: UUID
    account_name: str = Field(..., max_length=100)
    daily_loss_limit: str = Field(..., max_length=20)
    daily_loss_used: str = Field(..., max_length=20)
    max_position_per_market: str = Field(..., max_length=20)
    current_exposure: str = Field(..., max_length=20)
    status: Literal['healthy', 'warning', 'critical']


class CrossAccountRiskResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    total_exposure: str = Field(..., max_length=20)
    total_daily_loss: str = Field(..., max_length=20)
    accounts: list[AccountRiskSummary]
    overall_status: Literal['healthy', 'warning', 'critical']
    last_updated: datetime
