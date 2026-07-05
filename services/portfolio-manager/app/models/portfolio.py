from datetime import datetime
from typing import Optional

from pydantic import BaseModel, Field


# Capital tier definitions
TIER_THRESHOLDS = [
    {"tier": 1, "min": 0, "max": 100, "max_position_pct": 20, "max_daily_trades": 10, "max_strategies": 2},
    {"tier": 2, "min": 100, "max": 1000, "max_position_pct": 15, "max_daily_trades": 25, "max_strategies": 3},
    {"tier": 3, "min": 1000, "max": 10000, "max_position_pct": 10, "max_daily_trades": 50, "max_strategies": 5},
    {"tier": 4, "min": 10000, "max": float("inf"), "max_position_pct": 5, "max_daily_trades": 100, "max_strategies": 10},
]

PROMOTION_DAYS_REQUIRED = 7


def get_tier_for_capital(capital: float) -> dict:
    for t in TIER_THRESHOLDS:
        if t["min"] <= capital < t["max"]:
            return t
    return TIER_THRESHOLDS[-1]


class PortfolioOverview(BaseModel):
    total_capital: float
    deployed_capital: float
    utilization_rate: float
    current_tier: int
    tier_limits: dict
    days_above_threshold: int
    promotion_threshold: Optional[float]


class RebalanceRequest(BaseModel):
    weights: dict[str, float] = Field(..., description="Strategy weights, must sum to 100%")


class RebalanceResponse(BaseModel):
    old_weights: dict[str, float]
    new_weights: dict[str, float]
    updated_count: int


class TierTransition(BaseModel):
    from_tier: int
    to_tier: int
    capital: float
    reason: str
    created_at: datetime
