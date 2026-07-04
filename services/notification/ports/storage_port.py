from abc import ABC, abstractmethod
from datetime import datetime
from typing import Optional

from app.models import NotificationPreferences, NotificationRecord


class StoragePort(ABC):
    @abstractmethod
    async def connect(self) -> None:
        pass

    @abstractmethod
    async def save(self, record: NotificationRecord) -> None:
        pass

    @abstractmethod
    async def get_history(
        self,
        limit: int = 50,
        offset: int = 0,
        category: Optional[str] = None,
        status: Optional[str] = None,
        start_date: Optional[datetime] = None,
        end_date: Optional[datetime] = None,
    ) -> list[NotificationRecord]:
        pass

    @abstractmethod
    async def get_notification_by_id(
        self, notification_id: str
    ) -> Optional[NotificationRecord]:
        pass

    @abstractmethod
    async def purge_old(self, keep: int) -> int:
        pass

    @abstractmethod
    async def load_preferences(self) -> Optional[NotificationPreferences]:
        pass

    @abstractmethod
    async def save_preferences(self, prefs: NotificationPreferences) -> None:
        pass

    @abstractmethod
    async def close(self) -> None:
        pass
