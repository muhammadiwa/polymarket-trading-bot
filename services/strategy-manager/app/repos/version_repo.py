import json
import logging
from datetime import datetime, timezone
from typing import Optional

import asyncpg

logger = logging.getLogger(__name__)


async def create_version(
    conn: asyncpg.Connection,
    strategy_id: str,
    parameters: dict,
    change_summary: str,
    changed_by: Optional[str] = None,
) -> dict:
    """Create a new version for a strategy. Returns the version record."""
    # Atomic version number generation with retry on collision
    for attempt in range(3):
        try:
            version_row = await conn.fetchrow(
                """
                INSERT INTO strategy_versions (strategy_id, version_number, parameters, change_summary, changed_by)
                VALUES (
                    $1::uuid,
                    (SELECT COALESCE(MAX(version_number), 0) + 1 FROM strategy_versions WHERE strategy_id = $1::uuid),
                    $2::jsonb, $3, $4::uuid
                )
                RETURNING id, strategy_id, version_number, parameters, change_summary, changed_by, created_at
                """,
                strategy_id, json.dumps(parameters), change_summary, changed_by,
            )
            return _version_to_dict(version_row)
        except asyncpg.UniqueViolationError:
            if attempt == 2:
                raise
            logger.warning("version number collision, retrying", extra={"strategy_id": strategy_id, "attempt": attempt + 1})
    raise RuntimeError("Failed to create version after 3 attempts")


async def get_versions(
    conn: asyncpg.Connection, strategy_id: str, limit: int = 50, offset: int = 0
) -> tuple[list[dict], int]:
    """List all versions for a strategy, newest first. Returns (items, total)."""
    count_row = await conn.fetchrow(
        "SELECT COUNT(*) as total FROM strategy_versions WHERE strategy_id = $1::uuid",
        strategy_id,
    )
    total = count_row["total"]

    rows = await conn.fetch(
        """
        SELECT id, strategy_id, version_number, parameters, change_summary, changed_by, created_at
        FROM strategy_versions
        WHERE strategy_id = $1::uuid
        ORDER BY version_number DESC
        LIMIT $2 OFFSET $3
        """,
        strategy_id, limit, offset,
    )
    return [_version_to_dict(r) for r in rows], total


async def get_version(
    conn: asyncpg.Connection, strategy_id: str, version_id: str
) -> Optional[dict]:
    """Get a specific version."""
    row = await conn.fetchrow(
        """
        SELECT id, strategy_id, version_number, parameters, change_summary, changed_by, created_at
        FROM strategy_versions
        WHERE id = $1::uuid AND strategy_id = $2::uuid
        """,
        version_id, strategy_id,
    )
    if row is None:
        return None
    return _version_to_dict(row)


async def get_version_by_number(
    conn: asyncpg.Connection, strategy_id: str, version_number: int
) -> Optional[dict]:
    """Get a version by its number."""
    row = await conn.fetchrow(
        """
        SELECT id, strategy_id, version_number, parameters, change_summary, changed_by, created_at
        FROM strategy_versions
        WHERE strategy_id = $1::uuid AND version_number = $2
        """,
        strategy_id, version_number,
    )
    if row is None:
        return None
    return _version_to_dict(row)


def generate_change_summary(old_params: dict, new_params: dict) -> str:
    """Generate a human-readable change summary."""
    changes = []
    all_keys = set(old_params.keys()) | set(new_params.keys())
    for key in sorted(all_keys):
        old_val = old_params.get(key)
        new_val = new_params.get(key)
        if old_val != new_val:
            changes.append(f"{key}: {old_val} → {new_val}")
    return "\n".join(changes) if changes else "No parameter changes"


def _version_to_dict(row: asyncpg.Record) -> dict:
    params = row["parameters"]
    if isinstance(params, str):
        params = json.loads(params)
    return {
        "id": str(row["id"]),
        "strategy_id": str(row["strategy_id"]),
        "version_number": row["version_number"],
        "parameters": params,
        "change_summary": row["change_summary"],
        "changed_by": str(row["changed_by"]) if row["changed_by"] else None,
        "created_at": row["created_at"],
    }
