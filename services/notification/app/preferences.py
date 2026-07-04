import logging
from datetime import datetime, timezone

from app.models import (
    CategoryConfig,
    Channel,
    ChannelConfig,
    NotificationPreferences,
    Severity,
)
from ports.storage_port import StoragePort

logger = logging.getLogger(__name__)

DEFAULT_ID = "00000000-0000-0000-0000-000000000001"


class PreferencesManager:
    def __init__(self, storage: StoragePort) -> None:
        self._storage = storage
        self._cache: NotificationPreferences | None = None

    async def load(self) -> NotificationPreferences:
        prefs = await self._storage.load_preferences()
        if prefs is not None:
            self._cache = prefs
            logger.info("Loaded notification preferences from database")
        else:
            self._cache = NotificationPreferences()
            await self._storage.save_preferences(self._cache)
            logger.info("Created default notification preferences")
        return self._cache

    async def get(self) -> NotificationPreferences:
        if self._cache is None:
            return await self.load()
        return self._cache

    async def update(
        self,
        categories: CategoryConfig | None = None,
        channels: ChannelConfig | None = None,
    ) -> NotificationPreferences:
        if categories is not None and not categories.critical:
            raise ValueError("critical category cannot be disabled")
        current = await self.get()
        if categories is not None:
            current.categories = categories
            current.categories.critical = True
        if channels is not None:
            current.channels = channels
        current.updated_at = datetime.now(timezone.utc)
        await self._storage.save_preferences(current)
        self._cache = current
        logger.info(
            "Updated notification preferences: categories=%s channels=%s",
            current.categories.model_dump(),
            current.channels.model_dump(),
        )
        return current

    def is_enabled(self, severity: Severity) -> bool:
        if self._cache is None:
            return severity == Severity.CRITICAL
        return self._cache.is_enabled(severity)

    def is_channel_enabled(self, channel: Channel) -> bool:
        if self._cache is None:
            return channel == Channel.TELEGRAM
        return self._cache.is_channel_enabled(channel)
