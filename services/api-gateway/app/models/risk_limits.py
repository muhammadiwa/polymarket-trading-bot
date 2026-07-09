from datetime import datetime
from typing import Optional
from uuid import UUID

from pydantic import BaseModel, ConfigDict, Field
from pydantic.alias_generators import to_camel


class RiskLimitsResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    account_id: UUID
    daily_loss_limit: str
    max_position_per_market: str
    max_position_per_strategy: str
    drawdown_threshold: str


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
    account_name: str
    daily_loss_limit: str
    daily_loss_used: str
    max_position_per_market: str
    current_exposure: str
    status: str


class CrossAccountRiskResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    total_exposure: str
    total_daily_loss: str
    accounts: list[AccountRiskSummary]
    overall_status: str
    last_updated: datetime
