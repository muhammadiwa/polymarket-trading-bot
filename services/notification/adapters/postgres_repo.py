import json
import logging

import asyncpg

from app.models import NotificationPreferences, NotificationRecord, NotificationStatus
from ports.storage_port import StoragePort

logger = logging.getLogger(__name__)


class PostgresRepo(StoragePort):
    def __init__(self, database_url: str):
        self._database_url = database_url
        self._pool: asyncpg.Pool | None = None

    async def connect(self) -> None:
        self._pool = await asyncpg.create_pool(
            self._database_url, min_size=2, max_size=10, timeout=10
        )
        logger.info("PostgreSQL connection pool created")

    async def save(self, record: NotificationRecord) -> None:
        if self._pool is None:
            raise RuntimeError("Database pool not initialized")
        async with self._pool.acquire() as conn:
            await conn.execute(
                """
                INSERT INTO notifications
                    (id, event_type, severity, title, message, channel, status, delivered_at, created_at, metadata)
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb)
                """,
                str(record.id),
                record.event_type,
                record.severity.value,
                record.title,
                record.message,
                record.channel.value,
                record.status.value,
                record.delivered_at,
                record.created_at,
                json.dumps(record.metadata),
            )

    async def get_history(
        self, limit: int = 50, offset: int = 0
    ) -> list[NotificationRecord]:
        if self._pool is None:
            raise RuntimeError("Database pool not initialized")
        async with self._pool.acquire() as conn:
            rows = await conn.fetch(
                """
                SELECT id, event_type, severity, title, message, channel, status,
                       delivered_at, created_at, metadata
                FROM notifications
                ORDER BY created_at DESC
                LIMIT $1 OFFSET $2
                """,
                limit,
                offset,
            )
            return [self._row_to_record(r) for r in rows]

    async def purge_old(self, keep: int) -> int:
        if self._pool is None:
            raise RuntimeError("Database pool not initialized")
        async with self._pool.acquire() as conn:
            result = await conn.execute(
                """
                DELETE FROM notifications
                WHERE id NOT IN (
                    SELECT id FROM notifications
                    ORDER BY created_at DESC
                    LIMIT $1
                )
                """,
                keep,
            )
            count = int(result.split()[-1])
            if count > 0:
                logger.info("Purged %d old notification records", count)
            return count

    async def init_preferences_table(self) -> None:
        if self._pool is None:
            raise RuntimeError("Database pool not initialized")
        async with self._pool.acquire() as conn:
            await conn.execute(
                """
                CREATE TABLE IF NOT EXISTS notification_preferences (
                    id INTEGER PRIMARY KEY DEFAULT 1,
                    critical_enabled BOOLEAN NOT NULL DEFAULT TRUE,
                    warning_enabled BOOLEAN NOT NULL DEFAULT TRUE,
                    info_enabled BOOLEAN NOT NULL DEFAULT TRUE,
                    debug_enabled BOOLEAN NOT NULL DEFAULT FALSE,
                    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
                )
                """
            )

    async def save_preferences(self, prefs: NotificationPreferences) -> None:
        if self._pool is None:
            raise RuntimeError("Database pool not initialized")
        async with self._pool.acquire() as conn:
            await conn.execute(
                """
                INSERT INTO notification_preferences (id, critical_enabled, warning_enabled, info_enabled, debug_enabled, updated_at)
                VALUES (1, $1, $2, $3, $4, NOW())
                ON CONFLICT (id) DO UPDATE SET
                    critical_enabled = EXCLUDED.critical_enabled,
                    warning_enabled = EXCLUDED.warning_enabled,
                    info_enabled = EXCLUDED.info_enabled,
                    debug_enabled = EXCLUDED.debug_enabled,
                    updated_at = NOW()
                """,
                *prefs.to_row(),
            )

    async def load_preferences(self) -> NotificationPreferences | None:
        if self._pool is None:
            raise RuntimeError("Database pool not initialized")
        async with self._pool.acquire() as conn:
            row = await conn.fetchrow(
                "SELECT critical_enabled, warning_enabled, info_enabled, debug_enabled FROM notification_preferences WHERE id = 1"
            )
            if row is None:
                return None
            return NotificationPreferences.from_row(
                (row["critical_enabled"], row["warning_enabled"], row["info_enabled"], row["debug_enabled"])
            )

    async def close(self) -> None:
        if self._pool:
            await self._pool.close()
            self._pool = None
            logger.info("PostgreSQL connection pool closed")

    @staticmethod
    def _row_to_record(row: asyncpg.Record) -> NotificationRecord:
        return NotificationRecord(
            id=row["id"],
            event_type=row["event_type"],
            severity=row["severity"],
            title=row["title"],
            message=row["message"],
            channel=row["channel"],
            status=row["status"],
            delivered_at=row["delivered_at"],
            created_at=row["created_at"],
            metadata=row["metadata"] if isinstance(row["metadata"], dict) else {},
        )
