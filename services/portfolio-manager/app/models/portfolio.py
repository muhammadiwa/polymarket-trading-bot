from datetime import datetime
from decimal import Decimal
from typing import Optional

from pydantic import BaseModel, Field


# Capital tier definitions
TIER_THRESHOLDS = [
    {"tier": 1, "min": Decimal("0"), "max": Decimal("100"), "max_position_pct": 20, "max_daily_trades": 10, "max_strategies": 2},
    {"tier": 2, "min": Decimal("100"), "max": Decimal("1000"), "max_position_pct": 15, "max_daily_trades": 25, "max_strategies": 3},
    {"tier": 3, "min": Decimal("1000"), "max": Decimal("10000"), "max_position_pct": 10, "max_daily_trades": 50, "max_strategies": 5},
    {"tier": 4, "min": Decimal("10000"), "max": Decimal("Infinity"), "max_position_pct": 5, "max_daily_trades": 100, "max_strategies": 10},
]

PROMOTION_DAYS_REQUIRED = 7


def get_tier_for_capital(capital: Decimal) -> dict:
    for t in TIER_THRESHOLDS:
        if t["min"] <= capital < t["max"]:
            return t
    return TIER_THRESHOLDS[-1]


class PortfolioOverview(BaseModel):
    total_capital: Decimal
    deployed_capital: Decimal
    utilization_rate: Decimal
    current_tier: int
    tier_limits: dict
    days_above_threshold: int
    promotion_threshold: Optional[Decimal]


class RebalanceRequest(BaseModel):
    weights: dict[str, Decimal] = Field(..., description="Strategy weights, must sum to 100%")


class RebalanceResponse(BaseModel):
    old_weights: dict[str, Decimal]
    new_weights: dict[str, Decimal]
    updated_count: int


class TierTransition(BaseModel):
    from_tier: int
    to_tier: int
    capital: Decimal
    reason: str
    created_at: datetime
