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
    DELIVERED = "delivered"
    FAILED = "failed"
    THROTTLED = "throttled"


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


class NotificationPreferences(BaseModel):
    critical_enabled: bool = True
    warning_enabled: bool = True
    info_enabled: bool = True
    debug_enabled: bool = False

    def is_enabled(self, severity: Severity) -> bool:
        if severity == Severity.CRITICAL:
            return True
        mapping = {
            Severity.WARNING: self.warning_enabled,
            Severity.INFO: self.info_enabled,
            Severity.DEBUG: self.debug_enabled,
        }
        return mapping.get(severity, False)

    def to_row(self) -> tuple[bool, bool, bool, bool]:
        return (
            self.critical_enabled,
            self.warning_enabled,
            self.info_enabled,
            self.debug_enabled,
        )

    @classmethod
    def from_row(cls, row: tuple[bool, bool, bool, bool]) -> "NotificationPreferences":
        return cls(
            critical_enabled=row[0],
            warning_enabled=row[1],
            info_enabled=row[2],
            debug_enabled=row[3],
        )
