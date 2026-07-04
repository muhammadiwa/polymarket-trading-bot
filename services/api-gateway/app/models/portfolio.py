from decimal import Decimal

from pydantic import BaseModel, field_validator
from typing import Literal


class PortfolioOverview(BaseModel):
    totalCapital: str
    dailyPnL: str
    totalPnL: str
    utilizationRate: str
    lastUpdated: str

    @field_validator("totalCapital", "dailyPnL", "totalPnL")
    @classmethod
    def validate_decimal(cls, v: str) -> str:
        Decimal(v)
        return v

    @field_validator("utilizationRate")
    @classmethod
    def validate_rate(cls, v: str) -> str:
        d = Decimal(v)
        if d < 0 or d > 1:
            raise ValueError("utilizationRate must be between 0 and 1")
        return v


class Position(BaseModel):
    id: str
    market: str
    side: Literal["YES", "NO"]
    entryPrice: str
    currentPrice: str
    quantity: str
    unrealizedPnL: str
    updatedAt: str

    @field_validator("entryPrice", "currentPrice", "quantity", "unrealizedPnL")
    @classmethod
    def validate_decimal(cls, v: str) -> str:
        Decimal(v)
        return v

    @field_validator("market")
    @classmethod
    def validate_market(cls, v: str) -> str:
        if not v or len(v) > 500:
            raise ValueError("market must be non-empty and under 500 chars")
        return v
