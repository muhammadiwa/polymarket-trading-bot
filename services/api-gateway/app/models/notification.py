from datetime import datetime
from typing import Optional

from pydantic import BaseModel, Field


class CategoryConfig(BaseModel):
    critical: bool = True
    warning: bool = True
    info: bool = True
    debug: bool = False


class ChannelConfig(BaseModel):
    telegram: bool = True
    email: bool = False
    chat_id: str = ""
    email_to: str = ""


class NotificationPreferencesResponse(BaseModel):
    categories: CategoryConfig
    channels: ChannelConfig
    updated_at: datetime


class NotificationPreferencesUpdate(BaseModel):
    categories: Optional[CategoryConfig] = None
    channels: Optional[ChannelConfig] = None


class NotificationHistoryItem(BaseModel):
    id: str
    category: str
    title: str
    message: str
    channel: str
    status: str
    sent_at: Optional[datetime] = None
    created_at: datetime


class NotificationHistoryResponse(BaseModel):
    items: list[NotificationHistoryItem]
    total: int
    limit: int
    offset: int
