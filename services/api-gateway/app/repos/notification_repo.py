from datetime import datetime
from typing import Optional
from uuid import UUID

import asyncpg

from app.models.notification import (
    CategoryConfig,
    ChannelConfig,
    NotificationHistoryItem,
    NotificationPreferencesResponse,
)

_VALID_CATEGORIES = {"critical", "warning", "info", "debug"}
_VALID_STATUSES = {"sent", "failed", "throttled", "suppressed"}


async def get_preferences(conn: asyncpg.Connection) -> Optional[NotificationPreferencesResponse]:
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
    return NotificationPreferencesResponse(
        categories=CategoryConfig(
            critical=True,
            warning=row["warning"],
            info=row["info"],
            debug=row["debug"],
        ),
        channels=ChannelConfig(
            telegram=row["telegram_enabled"],
            email=row["email_enabled"],
            chat_id=row["telegram_chat_id"] or "",
            email_to=row["email_address"] or "",
        ),
        updated_at=row["updated_at"],
    )


async def update_preferences(
    conn: asyncpg.Connection,
    categories: Optional[CategoryConfig],
    channels: Optional[ChannelConfig],
) -> NotificationPreferencesResponse:
    if categories is not None and not categories.critical:
        raise ValueError("critical category cannot be disabled")

    current = await get_preferences(conn)
    if current is None:
        raise ValueError("Preferences not found")

    if categories is not None:
        current.categories = categories
        current.categories.critical = True
    if channels is not None:
        current.channels = channels

    await conn.execute(
        """
        UPDATE notification_preferences SET
            critical = $1, warning = $2, info = $3, debug = $4,
            telegram_enabled = $5, email_enabled = $6,
            telegram_chat_id = $7, email_address = $8,
            updated_at = NOW()
        WHERE id = '00000000-0000-0000-0000-000000000001'
        """,
        current.categories.critical,
        current.categories.warning,
        current.categories.info,
        current.categories.debug,
        current.channels.telegram,
        current.channels.email,
        current.channels.chat_id or None,
        current.channels.email_to or None,
    )

    updated = await get_preferences(conn)
    if updated is None:
        raise ValueError("Failed to load updated preferences")
    return updated


async def get_history(
    conn: asyncpg.Connection,
    limit: int = 50,
    offset: int = 0,
    category: Optional[str] = None,
    status: Optional[str] = None,
    start_date: Optional[datetime] = None,
    end_date: Optional[datetime] = None,
) -> tuple[list[NotificationHistoryItem], int]:
    if category is not None and category not in _VALID_CATEGORIES:
        raise ValueError(f"Invalid category: {category!r}")
    if status is not None and status not in _VALID_STATUSES:
        raise ValueError(f"Invalid status: {status!r}")

    conditions: list[str] = []
    params: list[object] = []
    idx = 1

    if category:
        conditions.append(f"category = ${idx}")
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

    count_row = await conn.fetchrow(
        f"SELECT COUNT(*) as total FROM notification_history {where_clause}",
        *params,
    )
    total = count_row["total"]

    rows = await conn.fetch(
        f"""
        SELECT id, category, title, message, channel, status, sent_at, created_at
        FROM notification_history
        {where_clause}
        ORDER BY created_at DESC
        LIMIT ${idx} OFFSET ${idx + 1}
        """,
        *params,
        limit,
        offset,
    )

    items = [
        NotificationHistoryItem(
            id=str(r["id"]),
            category=r["category"],
            title=r["title"],
            message=r["message"],
            channel=r["channel"],
            status=r["status"],
            sent_at=r["sent_at"],
            created_at=r["created_at"],
        )
        for r in rows
    ]

    return items, total


async def get_history_by_id(
    conn: asyncpg.Connection, notification_id: str
) -> Optional[NotificationHistoryItem]:
    try:
        UUID(notification_id)
    except ValueError:
        return None
    row = await conn.fetchrow(
        """
        SELECT id, category, title, message, channel, status, sent_at, created_at
        FROM notification_history
        WHERE id = $1
        """,
        notification_id,
    )
    if row is None:
        return None
    return NotificationHistoryItem(
        id=str(row["id"]),
        category=row["category"],
        title=row["title"],
        message=row["message"],
        channel=row["channel"],
        status=row["status"],
        sent_at=row["sent_at"],
        created_at=row["created_at"],
    )
