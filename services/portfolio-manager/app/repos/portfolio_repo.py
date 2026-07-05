import json
import logging
from datetime import datetime, timezone
from typing import Optional

import asyncpg

from app.models.portfolio import (
    PROMOTION_DAYS_REQUIRED,
    PortfolioOverview,
    RebalanceResponse,
    TierTransition,
    get_tier_for_capital,
)

logger = logging.getLogger(__name__)


async def get_or_create_tier(conn: asyncpg.Connection, account_id: Optional[str] = None) -> dict:
    if account_id:
        row = await conn.fetchrow(
            "SELECT * FROM portfolio_tiers WHERE account_id = $1::uuid", account_id
        )
    else:
        row = await conn.fetchrow("SELECT * FROM portfolio_tiers WHERE account_id IS NULL LIMIT 1")

    if row is None:
        row = await conn.fetchrow(
            """
            INSERT INTO portfolio_tiers (account_id, current_tier, total_capital, deployed_capital, utilization_rate)
            VALUES ($1, 1, 0, 0, 0) RETURNING *
            """,
            account_id,
        )
    return dict(row)


async def update_capital(
    conn: asyncpg.Connection,
    total_capital: float,
    deployed_capital: float,
    account_id: Optional[str] = None,
) -> PortfolioOverview:
    tier_info = get_tier_for_capital(total_capital)
    utilization = deployed_capital / total_capital if total_capital > 0 else 0

    current = await get_or_create_tier(conn, account_id)
    current_tier = current["current_tier"]
    days_above = current["days_above_threshold"]

    # Check promotion/demotion
    new_tier = current_tier
    reason = None

    if tier_info["tier"] > current_tier:
        # Capital is above next tier — check consecutive days
        days_above += 1
        threshold = TIER_THRESHOLDS[current_tier]["max"] if current_tier < len(TIER_THRESHOLDS) else float("inf")
        if days_above >= PROMOTION_DAYS_REQUIRED:
            new_tier = tier_info["tier"]
            reason = "promotion"
            days_above = 0
    elif tier_info["tier"] < current_tier:
        # Immediate demotion
        new_tier = tier_info["tier"]
        reason = "demotion"
        days_above = 0

    # Update tier record
    now = datetime.now(timezone.utc)
    await conn.execute(
        """
        UPDATE portfolio_tiers SET
            current_tier = $1, total_capital = $2, deployed_capital = $3,
            utilization_rate = $4, days_above_threshold = $5,
            promotion_threshold = $6, updated_at = $7,
            promoted_at = CASE WHEN $8 = 'promotion' THEN $7 ELSE promoted_at END,
            demoted_at = CASE WHEN $8 = 'demotion' THEN $7 ELSE demoted_at END
        WHERE id = $9
        """,
        new_tier, total_capital, deployed_capital, utilization,
        days_above, TIER_THRESHOLDS[min(new_tier, len(TIER_THRESHOLDS)-1)]["max"],
        now, reason, current["id"],
    )

    # Log transition if tier changed
    if reason and new_tier != current_tier:
        await conn.execute(
            """
            INSERT INTO tier_transitions (account_id, from_tier, to_tier, capital_at_transition, reason)
            VALUES ($1, $2, $3, $4, $5)
            """,
            account_id, current_tier, new_tier, total_capital, reason,
        )
        logger.info("tier transition", extra={
            "from_tier": current_tier, "to_tier": new_tier,
            "capital": total_capital, "reason": reason,
        })

    tier_limits = get_tier_for_capital(total_capital)

    return PortfolioOverview(
        total_capital=total_capital,
        deployed_capital=deployed_capital,
        utilization_rate=round(utilization, 4),
        current_tier=new_tier,
        tier_limits=tier_limits,
        days_above_threshold=days_above,
        promotion_threshold=TIER_THRESHOLDS[min(new_tier, len(TIER_THRESHOLDS)-1)]["max"],
    )


async def get_overview(conn: asyncpg.Connection, account_id: Optional[str] = None) -> PortfolioOverview:
    current = await get_or_create_tier(conn, account_id)
    tier_limits = get_tier_for_capital(float(current["total_capital"]))
    return PortfolioOverview(
        total_capital=float(current["total_capital"]),
        deployed_capital=float(current["deployed_capital"]),
        utilization_rate=float(current["utilization_rate"]),
        current_tier=current["current_tier"],
        tier_limits=tier_limits,
        days_above_threshold=current["days_above_threshold"],
        promotion_threshold=float(current["promotion_threshold"]) if current["promotion_threshold"] else None,
    )


async def log_rebalance(
    conn: asyncpg.Connection,
    old_weights: dict,
    new_weights: dict,
    initiated_by: Optional[str] = None,
    account_id: Optional[str] = None,
) -> None:
    await conn.execute(
        """
        INSERT INTO rebalance_log (account_id, old_weights, new_weights, initiated_by)
        VALUES ($1, $2, $3, $4)
        """,
        account_id, json.dumps(old_weights), json.dumps(new_weights), initiated_by,
    )


async def get_tier_transitions(
    conn: asyncpg.Connection, account_id: Optional[str] = None, limit: int = 50
) -> list[TierTransition]:
    if account_id:
        rows = await conn.fetch(
            "SELECT * FROM tier_transitions WHERE account_id = $1::uuid ORDER BY created_at DESC LIMIT $2",
            account_id, limit,
        )
    else:
        rows = await conn.fetch(
            "SELECT * FROM tier_transitions ORDER BY created_at DESC LIMIT $1", limit
        )
    return [
        TierTransition(
            from_tier=r["from_tier"], to_tier=r["to_tier"],
            capital=float(r["capital_at_transition"]), reason=r["reason"],
            created_at=r["created_at"],
        )
        for r in rows
    ]
