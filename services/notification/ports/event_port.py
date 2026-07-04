from abc import ABC, abstractmethod
from collections.abc import Awaitable, Callable

from app.models import NotificationRequest


EventHandler = Callable[[NotificationRequest], Awaitable[None]]


class EventPort(ABC):
    @abstractmethod
    async def connect(self) -> None:
        pass

    @abstractmethod
    async def subscribe(self, subject: str, handler: EventHandler) -> None:
        pass

    @abstractmethod
    async def close(self) -> None:
        pass
