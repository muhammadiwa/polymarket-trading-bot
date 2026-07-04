import json
import logging
from datetime import datetime
from typing import Optional

import asyncpg

from app.models import (
    NotificationPreferences,
    NotificationRecord,
    NotificationStatus,
    Severity,
)
from ports.storage_port import StoragePort

logger = logging.getLogger(__name__)

_VALID_SEVERITIES = {s.value for s in Severity}
_VALID_STATUSES = {s.value for s in NotificationStatus}


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
            async with conn.transaction():
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
                await conn.execute(
                    """
                    INSERT INTO notification_history
                        (id, category, title, message, channel, status, sent_at, created_at)
                    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
                    """,
                    str(record.id),
                    record.severity.value,
                    record.title,
                    record.message,
                    record.channel.value,
                    record.status.value,
                    record.delivered_at,
                    record.created_at,
                )

    async def get_history(
        self,
        limit: int = 50,
        offset: int = 0,
        category: Optional[str] = None,
        status: Optional[str] = None,
        start_date: Optional[datetime] = None,
        end_date: Optional[datetime] = None,
    ) -> list[NotificationRecord]:
        if self._pool is None:
            raise RuntimeError("Database pool not initialized")

        if category is not None and category not in _VALID_SEVERITIES:
            raise ValueError(f"Invalid category: {category!r}")
        if status is not None and status not in _VALID_STATUSES:
            raise ValueError(f"Invalid status: {status!r}")

        conditions: list[str] = []
        params: list[object] = []
        idx = 1

        if category:
            conditions.append(f"severity = ${idx}")
            params.append(category)
            idx += 1
        if status:
            conditions.append(f"status = ${idx}")
            params.append(status)
            idx += 1
        if start_date:
            conditions.append(f"created_at >= ${idx}")
            params.append(start_date)
            idx += 1
        if end_date:
            conditions.append(f"created_at <= ${idx}")
            params.append(end_date)
            idx += 1

        where_clause = f"WHERE {' AND '.join(conditions)}" if conditions else ""

        async with self._pool.acquire() as conn:
            rows = await conn.fetch(
                f"""
                SELECT id, event_type, severity, title, message, channel, status,
                       delivered_at, created_at, metadata
                FROM notifications
                {where_clause}
                ORDER BY created_at DESC
                LIMIT ${idx} OFFSET ${idx + 1}
                """,
                *params,
                limit,
                offset,
            )
            return [self._row_to_record(r) for r in rows]

    async def get_notification_by_id(
        self, notification_id: str
    ) -> Optional[NotificationRecord]:
        if self._pool is None:
            raise RuntimeError("Database pool not initialized")
        async with self._pool.acquire() as conn:
            row = await conn.fetchrow(
                """
                SELECT id, event_type, severity, title, message, channel, status,
                       delivered_at, created_at, metadata
                FROM notifications
                WHERE id = $1
                """,
                notification_id,
            )
            if row is None:
                return None
            return self._row_to_record(row)

    async def purge_old(self, keep: int) -> int:
        if self._pool is None:
            raise RuntimeError("Database pool not initialized")
        async with self._pool.acquire() as conn:
            async with conn.transaction():
                result = await conn.execute(
                    """
                    DELETE FROM notifications
                    WHERE created_at < (
                        SELECT created_at FROM notifications
                        ORDER BY created_at DESC
                        LIMIT 1 OFFSET $1
                    )
                    """,
                    max(keep - 1, 0),
                )
                await conn.execute(
                    """
                    DELETE FROM notification_history
                    WHERE created_at < (
                        SELECT created_at FROM notification_history
                        ORDER BY created_at DESC
                        LIMIT 1 OFFSET $1
                    )
                    """,
                    max(keep - 1, 0),
                )
            count = int(result.split()[-1])
            if count > 0:
                logger.info("Purged %d old notification records", count)
            return count

    async def init_tables(self) -> None:
        if self._pool is None:
            raise RuntimeError("Database pool not initialized")
        async with self._pool.acquire() as conn:
            await conn.execute(
                """
                CREATE TABLE IF NOT EXISTS notification_preferences (
                    id UUID PRIMARY KEY DEFAULT '00000000-0000-0000-0000-000000000001',
                    critical BOOLEAN NOT NULL DEFAULT TRUE,
                    warning BOOLEAN NOT NULL DEFAULT TRUE,
                    info BOOLEAN NOT NULL DEFAULT TRUE,
                    debug BOOLEAN NOT NULL DEFAULT FALSE,
                    telegram_enabled BOOLEAN NOT NULL DEFAULT TRUE,
                    email_enabled BOOLEAN NOT NULL DEFAULT FALSE,
                    telegram_chat_id VARCHAR(255),
                    email_address VARCHAR(255),
                    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
                )
                """
            )
            await conn.execute(
                """
                INSERT INTO notification_preferences (id) VALUES ('00000000-0000-0000-0000-000000000001')
                ON CONFLICT (id) DO NOTHING
                """
            )
            await conn.execute(
                """
                CREATE TABLE IF NOT EXISTS notification_history (
                    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                    category VARCHAR(20) NOT NULL,
                    title VARCHAR(255) NOT NULL,
                    message TEXT NOT NULL,
                    channel VARCHAR(20) NOT NULL,
                    status VARCHAR(20) NOT NULL,
                    sent_at TIMESTAMPTZ,
                    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
                )
                """
            )
            await conn.execute(
                """
                CREATE INDEX IF NOT EXISTS idx_notification_history_created
                ON notification_history(created_at DESC)
                """
            )
            await conn.execute(
                """
                CREATE INDEX IF NOT EXISTS idx_notification_history_category
                ON notification_history(category)
                """
            )
            await conn.execute(
                """
                CREATE INDEX IF NOT EXISTS idx_notifications_status
                ON notifications(status)
                """
            )

    async def save_preferences(self, prefs: NotificationPreferences) -> None:
        if self._pool is None:
            raise RuntimeError("Database pool not initialized")
        async with self._pool.acquire() as conn:
            await conn.execute(
                """
                INSERT INTO notification_preferences
                    (id, critical, warning, info, debug, telegram_enabled, email_enabled,
                     telegram_chat_id, email_address, updated_at)
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
                ON CONFLICT (id) DO UPDATE SET
                    critical = EXCLUDED.critical,
                    warning = EXCLUDED.warning,
                    info = EXCLUDED.info,
                    debug = EXCLUDED.debug,
                    telegram_enabled = EXCLUDED.telegram_enabled,
                    email_enabled = EXCLUDED.email_enabled,
                    telegram_chat_id = EXCLUDED.telegram_chat_id,
                    email_address = EXCLUDED.email_address,
                    updated_at = NOW()
                """,
                prefs.id,
                prefs.categories.critical,
                prefs.categories.warning,
                prefs.categories.info,
                prefs.categories.debug,
                prefs.channels.telegram,
                prefs.channels.email,
                prefs.channels.chat_id or None,
                prefs.channels.email_to or None,
                prefs.updated_at,
            )

    async def load_preferences(self) -> Optional[NotificationPreferences]:
        if self._pool is None:
            raise RuntimeError("Database pool not initialized")
        async with self._pool.acquire() as conn:
            row = await conn.fetchrow(
                """
                SELECT critical, warning, info, debug, telegram_enabled, email_enabled,
                       telegram_chat_id, email_address, updated_at
                FROM notification_preferences
                WHERE id = '00000000-0000-0000-0000-000000000001'
                """
            )
            if row is None:
                return None
            return NotificationPreferences.from_row(row)

    async def close(self) -> None:
        if self._pool:
            await self._pool.close()
            self._pool = None
            logger.info("PostgreSQL connection pool closed")

    @staticmethod
    def _row_to_record(row: asyncpg.Record) -> NotificationRecord:
        raw_metadata = row["metadata"]
        metadata: dict = {}
        if isinstance(raw_metadata, dict):
            metadata = raw_metadata
        elif raw_metadata is not None:
            logger.warning(
                "Unexpected metadata type %s for notification id=%s, defaulting to {}",
                type(raw_metadata).__name__,
                row["id"],
            )
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
            metadata=metadata,
        )
