import asyncio
import logging
import time
from datetime import datetime
from typing import Optional

from app.models import NotificationRecord, NotificationStatus
from ports.storage_port import StoragePort

logger = logging.getLogger(__name__)

_PURGE_INTERVAL = 300


class HistoryManager:
    def __init__(self, storage: StoragePort, retention_limit: int = 1000):
        self._storage = storage
        self._retention_limit = retention_limit
        self._last_purge: float = time.monotonic()
        self._purge_task: asyncio.Task[None] | None = None

    async def _maybe_purge(self) -> None:
        now = time.monotonic()
        if now - self._last_purge >= _PURGE_INTERVAL:
            self._last_purge = now
            if self._purge_task is None or self._purge_task.done():
                self._purge_task = asyncio.create_task(self._run_purge())

    async def _run_purge(self) -> None:
        try:
            await self._storage.purge_old(self._retention_limit)
        except Exception:
            logger.exception("Background purge failed")

    async def record_delivery(
        self, record: NotificationRecord, delivered: bool
    ) -> None:
        if delivered:
            record.status = NotificationStatus.SENT
        else:
            record.status = NotificationStatus.FAILED
        await self._storage.save(record)
        await self._maybe_purge()

    async def record_throttled(self, record: NotificationRecord) -> None:
        record.status = NotificationStatus.THROTTLED
        await self._storage.save(record)
        await self._maybe_purge()

    async def record_suppressed(self, record: NotificationRecord) -> None:
        record.status = NotificationStatus.SUPPRESSED
        await self._storage.save(record)
        await self._maybe_purge()

    async def get_history(
        self,
        limit: int = 50,
        offset: int = 0,
        category: Optional[str] = None,
        status: Optional[str] = None,
        start_date: Optional[datetime] = None,
        end_date: Optional[datetime] = None,
    ) -> list[NotificationRecord]:
        return await self._storage.get_history(
            limit=limit,
            offset=offset,
            category=category,
            status=status,
            start_date=start_date,
            end_date=end_date,
        )

    async def get_by_id(self, notification_id: str) -> Optional[NotificationRecord]:
        return await self._storage.get_notification_by_id(notification_id)
