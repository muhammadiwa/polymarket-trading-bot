from abc import ABC, abstractmethod

from app.models import NotificationRecord


class NotifyPort(ABC):
    @abstractmethod
    async def send(self, record: NotificationRecord) -> bool:
        pass
