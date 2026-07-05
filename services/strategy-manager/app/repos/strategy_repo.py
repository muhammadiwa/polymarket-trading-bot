from datetime import datetime, timezone
from typing import Optional

import asyncpg

from app.models.strategy import StrategyCreate, StrategyResponse, StrategyUpdate
from app.repos import version_repo

# #1: Allowlist for dynamic update fields — defense-in-depth against injection
ALLOWED_UPDATE_FIELDS = {
    "name", "description", "min_profit_threshold", "score_threshold",
    "max_position_size", "max_daily_trades", "risk_limit_pct", "capital_weight",
}


def _row_to_response(row: asyncpg.Record) -> StrategyResponse:
    return StrategyResponse(
        id=str(row["id"]),
        name=row["name"],
        description=row["description"],
        status=row["status"],
        min_profit_threshold=float(row["min_profit_threshold"]),
        score_threshold=float(row["score_threshold"]),
        max_position_size=float(row["max_position_size"]),
        max_daily_trades=row["max_daily_trades"],
        risk_limit_pct=float(row["risk_limit_pct"]),
        capital_weight=float(row["capital_weight"]),
        account_id=str(row["account_id"]) if row["account_id"] else None,
        created_at=row["created_at"],
        updated_at=row["updated_at"],
        activated_at=row.get("activated_at"),
    )


async def create_strategy(conn: asyncpg.Connection, data: StrategyCreate) -> StrategyResponse:
    row = await conn.fetchrow(
        """
        INSERT INTO strategies (name, description, min_profit_threshold, score_threshold,
            max_position_size, max_daily_trades, risk_limit_pct, capital_weight, account_id)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING *
        """,
        data.name, data.description, data.min_profit_threshold, data.score_threshold,
        data.max_position_size, data.max_daily_trades, data.risk_limit_pct,
        data.capital_weight, data.account_id,
    )
    return _row_to_response(row)


async def get_strategy(conn: asyncpg.Connection, strategy_id: str) -> Optional[StrategyResponse]:
    row = await conn.fetchrow("SELECT * FROM strategies WHERE id = $1::uuid", strategy_id)
    if row is None:
        return None
    return _row_to_response(row)


async def list_strategies(
    conn: asyncpg.Connection,
    status: Optional[str] = None,
    account_id: Optional[str] = None,
    limit: int = 50,
    offset: int = 0,
) -> tuple[list[StrategyResponse], int]:
    conditions = []
    params = []
    idx = 1

    if status:
        conditions.append(f"status = ${idx}")
        params.append(status)
        idx += 1
    if account_id:
        conditions.append(f"account_id = ${idx}::uuid")
        params.append(account_id)
        idx += 1

    where = f"WHERE {' AND '.join(conditions)}" if conditions else ""

    count_row = await conn.fetchrow(f"SELECT COUNT(*) as total FROM strategies {where}", *params)
    total = count_row["total"]

    rows = await conn.fetch(
        f"SELECT * FROM strategies {where} ORDER BY created_at DESC LIMIT ${idx} OFFSET ${idx + 1}",
        *params, limit, offset,
    )
    return [_row_to_response(r) for r in rows], total


async def update_strategy(
    conn: asyncpg.Connection, strategy_id: str, data: StrategyUpdate, changed_by: Optional[str] = None
) -> Optional[StrategyResponse]:
    # Get current state before update
    current = await get_strategy(conn, strategy_id)
    if current is None:
        return None

    fields = []
    params = []
    idx = 1

    for field, value in data.model_dump(exclude_unset=True).items():
        if field not in ALLOWED_UPDATE_FIELDS:
            continue
        if value is not None:
            fields.append(f"{field} = ${idx}")
            params.append(value)
            idx += 1

    if not fields:
        return current

    # #2: Create version snapshot before applying changes
    old_params = {
        "name": current.name,
        "description": current.description,
        "min_profit_threshold": current.min_profit_threshold,
        "score_threshold": current.score_threshold,
        "max_position_size": current.max_position_size,
        "max_daily_trades": current.max_daily_trades,
        "risk_limit_pct": current.risk_limit_pct,
        "capital_weight": current.capital_weight,
        "status": current.status,
    }
    new_params = {**old_params, **data.model_dump(exclude_unset=True)}
    change_summary = version_repo.generate_change_summary(old_params, new_params)

    await version_repo.create_version(conn, strategy_id, old_params, change_summary, changed_by)

    fields.append(f"updated_at = ${idx}")
    params.append(datetime.now(timezone.utc))
    idx += 1

    params.append(strategy_id)
    row = await conn.fetchrow(
        f"UPDATE strategies SET {', '.join(fields)} WHERE id = ${idx}::uuid RETURNING *",
        *params,
    )
    if row is None:
        return None
    return _row_to_response(row)


async def delete_strategy(conn: asyncpg.Connection, strategy_id: str) -> bool:
    # #2: Prevent deleting active strategies — must deactivate first
    row = await conn.fetchrow("SELECT status FROM strategies WHERE id = $1::uuid", strategy_id)
    if row is None:
        return False
    if row["status"] == "active":
        raise ValueError("Cannot delete an active strategy. Deactivate it first.")

    # #15: Use RETURNING for definitive result
    deleted = await conn.fetchrow("DELETE FROM strategies WHERE id = $1::uuid RETURNING id", strategy_id)
    return deleted is not None


async def activate_strategy(conn: asyncpg.Connection, strategy_id: str) -> Optional[StrategyResponse]:
    # #3: Conditional guard — only activate if not already active
    row = await conn.fetchrow(
        """
        UPDATE strategies SET status = 'active', activated_at = NOW(), updated_at = NOW()
        WHERE id = $1::uuid AND status != 'active'
        RETURNING *
        """,
        strategy_id,
    )
    if row is None:
        # Check if strategy exists but is already active
        existing = await conn.fetchrow("SELECT id FROM strategies WHERE id = $1::uuid", strategy_id)
        if existing is None:
            return None
        # Already active — return current state without event
        return await get_strategy(conn, strategy_id)
    return _row_to_response(row)


async def deactivate_strategy(conn: asyncpg.Connection, strategy_id: str) -> Optional[StrategyResponse]:
    # #3: Conditional guard — only deactivate if not already inactive
    row = await conn.fetchrow(
        """
        UPDATE strategies SET status = 'inactive', updated_at = NOW()
        WHERE id = $1::uuid AND status != 'inactive'
        RETURNING *
        """,
        strategy_id,
    )
    if row is None:
        existing = await conn.fetchrow("SELECT id FROM strategies WHERE id = $1::uuid", strategy_id)
        if existing is None:
            return None
        return await get_strategy(conn, strategy_id)
    return _row_to_response(row)
