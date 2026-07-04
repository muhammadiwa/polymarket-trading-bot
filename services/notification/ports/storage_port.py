from abc import ABC, abstractmethod

from app.models import NotificationRecord


class StoragePort(ABC):
    @abstractmethod
    async def connect(self) -> None:
        pass

    @abstractmethod
    async def save(self, record: NotificationRecord) -> None:
        pass

    @abstractmethod
    async def get_history(self, limit: int = 50, offset: int = 0) -> list[NotificationRecord]:
        pass

    @abstractmethod
    async def purge_old(self, keep: int) -> int:
        pass

    @abstractmethod
    async def close(self) -> None:
        pass
