from datetime import datetime
from decimal import Decimal
from enum import Enum
from typing import Literal, Optional

from pydantic import BaseModel, Field


class FillStatusEnum(str, Enum):
    PENDING = "PENDING"
    PLACED = "PLACED"
    FILLED = "FILLED"
    PARTIAL_FILL = "PARTIAL_FILL"
    CANCELLED = "CANCELLED"
    FAILED = "FAILED"
    EXPIRED = "EXPIRED"


class TradeResponse(BaseModel):
    id: str
    client_order_id: str
    strategy_id: str
    market_id: str
    market_slug: str
    side: Literal["YES", "NO"]
    order_type: Literal["GTC", "FOK", "GTD", "FAK"]
    price: Decimal
    quantity: Decimal
    filled_quantity: Decimal
    fill_status: FillStatusEnum
    latency_ms: int
    pnl: Decimal
    fee: Decimal
    slippage_pct: Decimal
    signal_timestamp: datetime
    order_timestamp: datetime
    fill_timestamp: Optional[datetime] = None
    opportunity_id: Optional[str] = None
    risk_decision: str
    failure_reason: Optional[str] = None
    account_id: Optional[str] = None
    created_at: datetime

    model_config = {"json_encoders": {Decimal: str}}


class TradeFilterParams(BaseModel):
    start_date: Optional[datetime] = None
    end_date: Optional[datetime] = None
    market_id: Optional[str] = None
    strategy_id: Optional[str] = None
    side: Optional[Literal["YES", "NO"]] = None
    pnl_sign: Optional[Literal["positive", "negative"]] = None
    fill_status: Optional[FillStatusEnum] = None
    page: int = Field(default=1, ge=1)
    page_size: int = Field(default=50, ge=1, le=100)
    cursor: Optional[str] = None


class TradeListResponse(BaseModel):
    trades: list[TradeResponse]
    total_count: int
    page: int
    page_size: int
    next_cursor: Optional[str] = None


class TradeStatsResponse(BaseModel):
    total_trades: int
    total_pnl: Decimal
    win_rate: Decimal
    avg_latency_ms: Decimal
    trades_by_strategy: dict[str, int]

    model_config = {"json_encoders": {Decimal: str}}
