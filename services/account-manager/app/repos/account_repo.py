import logging
from typing import Optional

import asyncpg

logger = logging.getLogger(__name__)


async def create_account(
    conn: asyncpg.Connection,
    name: str,
    wallet_address: str,
    private_key_encrypted: bytes,
    private_key_iv: bytes,
    private_key_tag: bytes,
) -> dict:
    """Create a new account with default risk limits."""
    row = await conn.fetchrow(
        """
        INSERT INTO accounts (name, wallet_address, private_key_encrypted, private_key_iv, private_key_tag)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING *
        """,
        name, wallet_address, private_key_encrypted, private_key_iv, private_key_tag,
    )

    # Create default risk limits for the new account
    await conn.execute(
        """
        INSERT INTO account_risk_limits (account_id)
        VALUES ($1)
        ON CONFLICT (account_id) DO NOTHING
        """,
        row["id"],
    )

    return _row_to_account(row)


async def get_account_by_id(conn: asyncpg.Connection, account_id: str) -> Optional[dict]:
    """Get account by ID."""
    row = await conn.fetchrow(
        "SELECT * FROM accounts WHERE id = $1::uuid",
        account_id,
    )
    if row is None:
        return None
    return _row_to_account(row)


async def get_account_by_wallet(conn: asyncpg.Connection, wallet_address: str) -> Optional[dict]:
    """Get account by wallet address."""
    row = await conn.fetchrow(
        "SELECT * FROM accounts WHERE wallet_address = $1",
        wallet_address,
    )
    if row is None:
        return None
    return _row_to_account(row)


async def list_accounts(
    conn: asyncpg.Connection,
    is_active: Optional[bool] = None,
    limit: int = 50,
    offset: int = 0,
) -> tuple[list[dict], int]:
    """List accounts with optional filtering."""
    conditions = []
    params = []
    idx = 1

    if is_active is not None:
        conditions.append(f"is_active = ${idx}")
        params.append(is_active)
        idx += 1

    where = f"WHERE {' AND '.join(conditions)}" if conditions else ""

    count_row = await conn.fetchrow(f"SELECT COUNT(*) as total FROM accounts {where}", *params)
    total = count_row["total"]

    rows = await conn.fetch(
        f"SELECT * FROM accounts {where} ORDER BY created_at DESC LIMIT ${idx} OFFSET ${idx + 1}",
        *params, limit, offset,
    )
    return [_row_to_account(r) for r in rows], total


async def update_account(
    conn: asyncpg.Connection,
    account_id: str,
    name: Optional[str] = None,
) -> Optional[dict]:
    """Update account."""
    # Validate name is not empty
    if name is not None and not name.strip():
        name = None  # Ignore empty name

    updates = []
    params = []
    idx = 1

    if name is not None:
        updates.append(f"name = ${idx}")
        params.append(name)
        idx += 1

    if not updates:
        return await get_account_by_id(conn, account_id)

    updates.append("updated_at = NOW()")
    params.append(account_id)

    row = await conn.fetchrow(
        f"UPDATE accounts SET {', '.join(updates)} WHERE id = ${idx}::uuid RETURNING *",
        *params,
    )
    if row is None:
        return None
    return _row_to_account(row)


async def update_encrypted_key(
    conn: asyncpg.Connection,
    account_id: str,
    private_key_encrypted: bytes,
    private_key_iv: bytes,
    private_key_tag: bytes,
) -> Optional[dict]:
    """Update encrypted private key for an account."""
    row = await conn.fetchrow(
        """
        UPDATE accounts
        SET private_key_encrypted = $2, private_key_iv = $3, private_key_tag = $4, updated_at = NOW()
        WHERE id = $1::uuid
        RETURNING *
        """,
        account_id, private_key_encrypted, private_key_iv, private_key_tag,
    )
    if row is None:
        return None
    return _row_to_account(row)


async def delete_account(conn: asyncpg.Connection, account_id: str) -> bool:
    """Delete account (hard delete for rollback on encryption failure)."""
    result = await conn.execute(
        "DELETE FROM accounts WHERE id = $1::uuid",
        account_id,
    )
    return result == "DELETE 1"


async def set_account_active(conn: asyncpg.Connection, account_id: str, is_active: bool) -> Optional[dict]:
    """Activate or deactivate account."""
    row = await conn.fetchrow(
        """
        UPDATE accounts SET is_active = $1, updated_at = NOW()
        WHERE id = $2::uuid
        RETURNING *
        """,
        is_active, account_id,
    )
    if row is None:
        return None
    return _row_to_account(row)


async def get_account_encrypted_key(conn: asyncpg.Connection, account_id: str) -> Optional[dict]:
    """Get encrypted private key components for decryption."""
    row = await conn.fetchrow(
        """
        SELECT private_key_encrypted, private_key_iv, private_key_tag
        FROM accounts WHERE id = $1::uuid
        """,
        account_id,
    )
    if row is None:
        return None
    return {
        "encrypted": row["private_key_encrypted"],
        "iv": row["private_key_iv"],
        "tag": row["private_key_tag"],
    }


def _row_to_account(row: asyncpg.Record) -> dict:
    return {
        "id": str(row["id"]),
        "name": row["name"],
        "wallet_address": row["wallet_address"],
        "is_active": row["is_active"],
        "created_at": row["created_at"],
        "updated_at": row["updated_at"],
    }
