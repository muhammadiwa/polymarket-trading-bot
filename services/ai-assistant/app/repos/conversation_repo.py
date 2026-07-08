import logging
from typing import Optional

import asyncpg

from app.models.assistant import ConversationMessage

logger = logging.getLogger(__name__)


async def save_message(
    conn: asyncpg.Connection,
    user_id: str,
    role: str,
    content: str,
    metadata: Optional[dict] = None,
) -> str:
    """Save a conversation message."""
    row = await conn.fetchrow(
        """
        INSERT INTO assistant_conversations (user_id, role, content, metadata)
        VALUES ($1::uuid, $2, $3, $4::jsonb)
        RETURNING id
        """,
        user_id, role, content, metadata,
    )
    return str(row["id"])


async def get_history(
    conn: asyncpg.Connection,
    user_id: str,
    limit: int = 50,
) -> list[ConversationMessage]:
    """Get conversation history for a user."""
    rows = await conn.fetch(
        """
        SELECT id, role, content, metadata, created_at
        FROM assistant_conversations
        WHERE user_id = $1::uuid
        ORDER BY created_at DESC
        LIMIT $2
        """,
        user_id, limit,
    )
    return [
        ConversationMessage(
            id=str(row["id"]),
            role=row["role"],
            content=row["content"],
            metadata=row["metadata"],
            created_at=row["created_at"],
        )
        for row in rows
    ]
