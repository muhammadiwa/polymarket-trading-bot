from datetime import datetime
from decimal import Decimal
from typing import Optional

from pydantic import BaseModel, Field


class AskRequest(BaseModel):
    question: str = Field(min_length=1, max_length=1000)


class AskResponse(BaseModel):
    answer: str
    data_points: list[dict]
    source_trade_ids: list[str]
    response_time_ms: int
    conversation_id: str


class TradeExplanation(BaseModel):
    trade_id: str
    market_id: str
    side: str
    entry_price: Decimal
    exit_price: Optional[Decimal]
    pnl: Optional[Decimal]
    explanation: str
    decision_context: str
    related_events: list[dict]
    conversation_id: str


class ConversationMessage(BaseModel):
    id: str
    role: str
    content: str
    metadata: Optional[dict]
    created_at: datetime


class ConversationHistory(BaseModel):
    messages: list[ConversationMessage]
    total: int
