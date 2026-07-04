from datetime import datetime, timezone
from enum import Enum
from typing import Any, Optional
from uuid import UUID, uuid4

from pydantic import BaseModel, Field


class Severity(str, Enum):
    CRITICAL = "critical"
    WARNING = "warning"
    INFO = "info"
    DEBUG = "debug"


class NotificationStatus(str, Enum):
    SENT = "sent"
    FAILED = "failed"
    THROTTLED = "throttled"
    SUPPRESSED = "suppressed"


class Channel(str, Enum):
    TELEGRAM = "telegram"
    EMAIL = "email"


class NotificationPayload(BaseModel):
    title: str
    message: str
    severity: Severity = Severity.INFO
    category: str = ""
    channel: str = "telegram"
    priority: str = ""
    bypass_throttle: bool = False
    metadata: dict[str, Any] = Field(default_factory=dict)


class NotificationRequest(BaseModel):
    event_id: UUID
    event_type: str
    timestamp: datetime
    source: str
    payload: NotificationPayload


class NotificationRecord(BaseModel):
    id: UUID = Field(default_factory=uuid4)
    event_type: str
    severity: Severity
    title: str
    message: str
    channel: Channel = Channel.TELEGRAM
    status: NotificationStatus
    delivered_at: Optional[datetime] = None
    created_at: datetime = Field(default_factory=lambda: datetime.now(timezone.utc))
    metadata: dict[str, Any] = Field(default_factory=dict)


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


class NotificationPreferences(BaseModel):
    id: str = "00000000-0000-0000-0000-000000000001"
    categories: CategoryConfig = Field(default_factory=CategoryConfig)
    channels: ChannelConfig = Field(default_factory=ChannelConfig)
    updated_at: datetime = Field(default_factory=lambda: datetime.now(timezone.utc))

    def is_enabled(self, severity: Severity) -> bool:
        if severity == Severity.CRITICAL:
            return True
        mapping = {
            Severity.WARNING: self.categories.warning,
            Severity.INFO: self.categories.info,
            Severity.DEBUG: self.categories.debug,
        }
        return mapping.get(severity, False)

    def is_channel_enabled(self, channel: Channel) -> bool:
        if channel == Channel.TELEGRAM:
            return self.channels.telegram
        if channel == Channel.EMAIL:
            return self.channels.email
        return False

    def to_row(self) -> tuple[bool, bool, bool, bool, bool, bool, str, str]:
        return (
            self.categories.critical,
            self.categories.warning,
            self.categories.info,
            self.categories.debug,
            self.channels.telegram,
            self.channels.email,
            self.channels.chat_id,
            self.channels.email_to,
        )

    @classmethod
    def from_row(cls, row: Any) -> "NotificationPreferences":
        return cls(
            categories=CategoryConfig(
                critical=True,
                warning=row["warning"],
                info=row["info"],
                debug=row["debug"],
            ),
            channels=ChannelConfig(
                telegram=row["telegram_enabled"],
                email=row["email_enabled"],
                chat_id=row["telegram_chat_id"] or "",
                email_to=row["email_address"] or "",
            ),
            updated_at=row["updated_at"],
        )
