from datetime import datetime, timezone
from decimal import Decimal
from uuid import uuid4

from pydantic import BaseModel, Field


class CapitalUpdatedPayload(BaseModel):
    total_capital: Decimal
    daily_pnl: Decimal
    unrealized_pnl: Decimal
    capital_tier: str


class CapitalUpdated(BaseModel):
    event_id: str = Field(default_factory=lambda: str(uuid4()))
    event_type: str = "CapitalUpdated"
    timestamp: datetime = Field(default_factory=lambda: datetime.now(timezone.utc))
    source: str = "portfolio-manager"
    payload: CapitalUpdatedPayload

    def to_dict(self) -> dict:
        return {
            "event_id": self.event_id,
            "event_type": self.event_type,
            "timestamp": self.timestamp.isoformat(),
            "source": self.source,
            "payload": {
                "total_capital": str(self.payload.total_capital),
                "daily_pnl": str(self.payload.daily_pnl),
                "unrealized_pnl": str(self.payload.unrealized_pnl),
                "capital_tier": self.payload.capital_tier,
            },
        }
