import time

from app.models import NotificationRecord, NotificationStatus
from ports.storage_port import StoragePort

_PURGE_INTERVAL = 300  # seconds


class HistoryManager:
    def __init__(self, storage: StoragePort, retention_limit: int = 1000):
        self._storage = storage
        self._retention_limit = retention_limit
        self._last_purge: float = time.monotonic()

    async def _maybe_purge(self) -> None:
        now = time.monotonic()
        if now - self._last_purge >= _PURGE_INTERVAL:
            self._last_purge = now
            await self._storage.purge_old(self._retention_limit)

    async def record_delivery(
        self, record: NotificationRecord, delivered: bool
    ) -> None:
        if delivered:
            record.status = NotificationStatus.DELIVERED
        else:
            record.status = NotificationStatus.FAILED
        await self._storage.save(record)
        await self._maybe_purge()

    async def record_throttled(self, record: NotificationRecord) -> None:
        record.status = NotificationStatus.THROTTLED
        await self._storage.save(record)
        await self._maybe_purge()

    async def get_history(
        self, limit: int = 50, offset: int = 0
    ) -> list[NotificationRecord]:
        return await self._storage.get_history(limit=limit, offset=offset)
