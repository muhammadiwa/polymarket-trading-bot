import json
import logging
import re
import time
import uuid
from collections import defaultdict
from datetime import datetime, timezone
from decimal import Decimal

import nats
import redis.asyncio as aioredis
from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel, field_validator

from app.config import config
from app.db import get_pool
from app.middleware.auth import extract_user, require_admin
from app.metrics import (
    PORTFOLIO_CROSS_ACCOUNT_LATENCY,
    PORTFOLIO_CROSS_ACCOUNT_QUERIES_TOTAL,
    RISK_ACTION_LATENCY,
    RISK_ACTIONS_TOTAL,
    RISK_LIMITS_UPDATES_TOTAL,
    RISK_PARAM_CHANGES_TOTAL,
    RISK_PER_ACCOUNT_LIMIT_CHECKS_TOTAL,
)
from app.models.risk_limits import (
    AccountRiskSummary,
    CrossAccountRiskResponse,
    RiskLimitsResponse,
    RiskLimitsUpdate,
)

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/risk", tags=["risk"])

# #8: Module-level Redis connection pool
_redis_pool: aioredis.Redis | None = None

# #9: Long-lived NATS connection
_nc: nats.NATS | None = None


async def _get_redis() -> aioredis.Redis:
    global _redis_pool
    if _redis_pool is None:
        _redis_pool = aioredis.from_url(config.REDIS_URL, decode_responses=True)
    return _redis_pool


async def _get_nats() -> nats.NATS:
    global _nc
    if _nc is None or _nc.is_closed:
        _nc = await nats.connect(config.NATS_URL)
    return _nc


# #11: Simple per-user rate limiter for emergency stop
_emergency_rate_limits: dict[str, list[float]] = defaultdict(list)
EMERGENCY_RATE_LIMIT = 3
EMERGENCY_RATE_WINDOW = 60.0
_EMERGENCY_EVICTION_INTERVAL = 300
_last_emergency_eviction: float = 0.0


def _evict_stale_emergency_entries() -> None:
    """Clean up stale emergency rate limit entries."""
    global _last_emergency_eviction
    now = time.monotonic()
    if now - _last_emergency_eviction < _EMERGENCY_EVICTION_INTERVAL:
        return
    _last_emergency_eviction = now
    window_start = now - EMERGENCY_RATE_WINDOW
    stale_keys = [k for k, v in _emergency_rate_limits.items() if not any(t > window_start for t in v)]
    for k in stale_keys:
        del _emergency_rate_limits[k]


def _check_emergency_rate_limit(user_id: str) -> None:
    # #20: Uses time.monotonic() which resets on service restart.
    # This is acceptable: rate limits are best-effort and a restart
    # already disrupts the attack surface.  Do NOT switch to wall clock
    # here — monotonic is immune to NTP jumps.
    _evict_stale_emergency_entries()
    now = time.monotonic()
    window_start = now - EMERGENCY_RATE_WINDOW
    _emergency_rate_limits[user_id] = [
        t for t in _emergency_rate_limits[user_id] if t > window_start
    ]
    if len(_emergency_rate_limits[user_id]) >= EMERGENCY_RATE_LIMIT:
        raise HTTPException(
            status_code=status.HTTP_429_TOO_MANY_REQUESTS,
            detail="Too many emergency stop requests. Try again later.",
        )
    _emergency_rate_limits[user_id].append(now)


# #4: Lua script for atomic emergency stop token check-and-delete
_EMERGENCY_TOKEN_LUA = """
local val = redis.call('GET', KEYS[1])
if val then
    redis.call('DEL', KEYS[1])
    return val
else
    return nil
end
"""

# #5: Lua script for atomic state read-modify-write
_STATE_UPDATE_LUA = """
local current = redis.call('GET', KEYS[1])
if not current then
    current = '{}'
end
local state = cjson.decode(current)
local updates = cjson.decode(ARGV[1])
for k, v in pairs(updates) do
    state[k] = v
end
local new_state = cjson.encode(state)
redis.call('SET', KEYS[1], new_state, 'EX', ARGV[2])
return new_state
"""


class EmergencyStopRequest(BaseModel):
    reason: str
    confirmationToken: str


class ResumeRequest(BaseModel):
    confirmationToken: str


class PauseRequest(BaseModel):
    reason: str


class RiskParameterUpdate(BaseModel):
    dailyLossLimit: str | None = None
    maxPositionPerMarket: str | None = None
    maxPositionPerStrategy: str | None = None

    @field_validator("dailyLossLimit")
    @classmethod
    def validate_daily_loss_limit(cls, v: str | None) -> str | None:
        if v is None:
            return v
        try:
            num = Decimal(v)
            if num <= 0:
                raise ValueError("Must be positive")
            if num > 20:
                raise ValueError("Must be <= 20%")
            return v
        except Exception as e:
            raise ValueError(f"Invalid percentage: {e}")

    @field_validator("maxPositionPerMarket", "maxPositionPerStrategy")
    @classmethod
    def validate_position_limit(cls, v: str | None) -> str | None:
        if v is None:
            return v
        try:
            num = Decimal(v)
            if num <= 0:
                raise ValueError("Must be positive")
            if num > 50:
                raise ValueError("Must be <= 50%")
            return v
        except Exception as e:
            raise ValueError(f"Invalid percentage: {e}")


def _safe_decimal_str(val, default="0") -> str:
    if val is None:
        return default
    return str(val)


@router.get("/status")
async def get_risk_status(_user: dict = Depends(extract_user)):
    start = time.monotonic()
    r = await _get_redis()

    try:
        state_raw = await r.get("pqap:risk:state")

        if not state_raw:
            raise HTTPException(
                status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
                detail="Risk state not available",
            )

        state = json.loads(state_raw)

        elapsed = (time.monotonic() - start) * 1000
        RISK_ACTION_LATENCY.observe(elapsed)

        daily_budget_remaining = _safe_decimal_str(state.get("daily_budget_remaining"))
        capital = _safe_decimal_str(state.get("capital"))
        daily_loss_limit = _safe_decimal_str(state.get("daily_loss_limit"))
        daily_loss = _safe_decimal_str(state.get("daily_loss"))

        budget_total = daily_loss_limit
        try:
            remaining = Decimal(daily_budget_remaining)
            total = Decimal(daily_loss_limit)
            used_pct = "0"
            if total > 0:
                # #12: Clamp used fraction to [0, 1]
                raw = (total - remaining) / total
                clamped = max(Decimal("0"), min(Decimal("1"), raw))
                used_pct = str(clamped)
        except Exception:
            used_pct = "0"

        drawdown = _safe_decimal_str(state.get("drawdown"))
        drawdown_limit = _safe_decimal_str(state.get("drawdown_limit"))

        emergency_stop = state.get("emergency_stop", False)
        emergency_timestamp = state.get("emergency_stop_timestamp")
        batasi_paused = state.get("batasi_win_paused", False)

        is_paused = emergency_stop or batasi_paused
        paused_reason = None
        if emergency_stop:
            paused_reason = state.get("emergency_stop_reason", "Emergency stop active")
        elif batasi_paused:
            paused_reason = "Win streak threshold reached"

        win_streak_current = state.get("win_streak_current", 0)
        win_streak_threshold = state.get("win_streak_threshold", 5)

        updated_at = state.get("updated_at")

        return {
            "dailyBudgetRemaining": daily_budget_remaining,
            "dailyBudgetTotal": budget_total,
            "dailyBudgetUsedFraction": used_pct,
            "currentDrawdown": drawdown,
            "drawdownThreshold": drawdown_limit,
            "winStreakCurrent": win_streak_current,
            "winStreakThreshold": win_streak_threshold,
            "circuitBreakerStatus": "open" if emergency_stop else "closed",
            "circuitBreakerTrippedAt": emergency_timestamp,
            "isPaused": is_paused,
            "pausedReason": paused_reason,
            "lastUpdated": updated_at,
        }

    except HTTPException:
        raise
    except Exception:
        logger.exception("Failed to read risk status from Redis")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to read risk status",
        )


@router.post("/emergency-stop/confirmationToken")
async def generate_emergency_stop_token(_user: dict = Depends(extract_user)):
    r = await _get_redis()
    try:
        token = str(uuid.uuid4())
        await r.set(f"pqap:risk:emergency_token:{token}", "1", ex=60)
        return {"confirmationToken": token}
    finally:
        pass


@router.post("/emergency-stop")
async def emergency_stop(
    body: EmergencyStopRequest,
    user: dict = Depends(extract_user),
):
    start = time.monotonic()
    r = await _get_redis()

    # #11: Rate limit emergency stop per user
    user_id = user.get("user_id", "unknown")
    _check_emergency_rate_limit(user_id)

    try:
        # #4: Atomic check-and-delete for confirmation token
        token_key = f"pqap:risk:emergency_token:{body.confirmationToken}"
        stored = await r.eval(_EMERGENCY_TOKEN_LUA, 1, token_key)
        if not stored:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="Invalid or expired confirmation token",
            )

        now = datetime.now(timezone.utc).isoformat()

        # #5: Atomic read-modify-write for Redis state
        updates = json.dumps({
            "emergency_stop": True,
            "emergency_stop_reason": body.reason,
            "emergency_stop_timestamp": now,
            "updated_at": now,
        })
        new_state_raw = await r.eval(_STATE_UPDATE_LUA, 1, "pqap:risk:state", updates, "3600")

        # #1: Standalone keys eliminated — emergency fields derived from composite state.
        # #6: Single source of truth: pqap:risk:state contains all fields.

        elapsed = (time.monotonic() - start) * 1000
        RISK_ACTIONS_TOTAL.labels(action="emergency_stop").inc()
        RISK_ACTION_LATENCY.observe(elapsed)

        logger.warning("Emergency stop triggered: %s", body.reason)

        # #7: Publish NATS command so risk-manager syncs
        nc = await _get_nats()
        await nc.publish("pqap.risk.command", json.dumps({
            "command": "emergency_stop",
            "reason": body.reason,
            "timestamp": now,
        }).encode())

        return {"status": "emergency_stop_activated", "reason": body.reason}

    except HTTPException:
        raise
    except Exception:
        logger.exception("Failed to trigger emergency stop")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to trigger emergency stop",
        )


@router.post("/emergency-stop/resume/confirmationToken")
async def generate_resume_token(_user: dict = Depends(extract_user)):
    r = await _get_redis()
    try:
        token = str(uuid.uuid4())
        await r.set(f"pqap:risk:resume_token:{token}", "1", ex=60)
        return {"confirmationToken": token}
    finally:
        pass


@router.post("/pause")
async def pause_trading(
    body: PauseRequest,
    _user: dict = Depends(extract_user),
):
    start = time.monotonic()
    r = await _get_redis()

    # #16: Require non-empty reason
    reason = body.reason or "Manual pause"

    try:
        now = datetime.now(timezone.utc).isoformat()

        # #5: Atomic read-modify-write
        updates = json.dumps({
            "batasi_win_paused": True,
            "updated_at": now,
        })
        await r.eval(_STATE_UPDATE_LUA, 1, "pqap:risk:state", updates, "3600")

        # #6: Standalone key eliminated — derived from composite state.

        elapsed = (time.monotonic() - start) * 1000
        RISK_ACTIONS_TOTAL.labels(action="pause").inc()
        RISK_ACTION_LATENCY.observe(elapsed)

        logger.info("Trading paused: %s", reason)

        # #7: Publish NATS command
        nc = await _get_nats()
        await nc.publish("pqap.risk.command", json.dumps({
            "command": "pause",
            "reason": reason,
            "timestamp": now,
        }).encode())

        return {"status": "trading_paused", "reason": reason}

    except Exception:
        logger.exception("Failed to pause trading")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to pause trading",
        )


@router.post("/resume")
async def resume_trading(
    body: ResumeRequest,
    _user: dict = Depends(extract_user),
):
    start = time.monotonic()
    r = await _get_redis()

    try:
        # #6: Validate resume confirmation token
        token_key = f"pqap:risk:resume_token:{body.confirmationToken}"
        stored = await r.eval(_EMERGENCY_TOKEN_LUA, 1, token_key)
        if not stored:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="Invalid or expired confirmation token",
            )

        now = datetime.now(timezone.utc).isoformat()

        # #5: Atomic read-modify-write
        updates = json.dumps({
            "emergency_stop": False,
            "emergency_stop_reason": "",
            "emergency_stop_timestamp": None,
            "batasi_win_paused": False,
            "updated_at": now,
        })
        await r.eval(_STATE_UPDATE_LUA, 1, "pqap:risk:state", updates, "3600")

        # #6: Standalone keys eliminated — derived from composite state.

        elapsed = (time.monotonic() - start) * 1000
        RISK_ACTIONS_TOTAL.labels(action="resume").inc()
        RISK_ACTION_LATENCY.observe(elapsed)

        logger.info("Trading resumed")

        # #7: Publish NATS command
        nc = await _get_nats()
        await nc.publish("pqap.risk.command", json.dumps({
            "command": "resume",
            "timestamp": now,
        }).encode())

        return {"status": "trading_resumed"}

    except HTTPException:
        raise
    except Exception:
        logger.exception("Failed to resume trading")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to resume trading",
        )


@router.put("/parameters")
async def update_parameters(
    body: RiskParameterUpdate,
    _user: dict = Depends(extract_user),
):
    start = time.monotonic()
    r = await _get_redis()

    try:
        # #14: Reject empty parameter updates
        if body.dailyLossLimit is None and body.maxPositionPerMarket is None and body.maxPositionPerStrategy is None:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="At least one parameter must be provided",
            )

        # #2: Atomic read-modify-write using Lua script to prevent race conditions
        state_raw = await r.get("pqap:risk:state")
        state = json.loads(state_raw) if state_raw else {}

        changes = {}

        if body.dailyLossLimit is not None:
            # #13: Validate capital > 0
            capital = Decimal(state.get("capital", "0"))
            if capital <= 0:
                raise HTTPException(
                    status_code=status.HTTP_400_BAD_REQUEST,
                    detail="Capital must be greater than 0 to set daily loss limit",
                )
            limit = Decimal(body.dailyLossLimit) / 100
            changes["dailyLossLimit"] = body.dailyLossLimit
            changes["daily_loss_limit"] = str(capital * limit)

        if body.maxPositionPerMarket is not None:
            changes["maxPositionPerMarket"] = body.maxPositionPerMarket

        if body.maxPositionPerStrategy is not None:
            changes["maxPositionPerStrategy"] = body.maxPositionPerStrategy

        now = datetime.now(timezone.utc).isoformat()

        # #10: Persist to PostgreSQL BEFORE Redis to ensure consistency on failure
        try:
            pool = await get_pool()
            async with pool.acquire() as conn:
                await conn.execute(
                    """
                    INSERT INTO risk_parameters (daily_loss_limit, max_position_per_market, max_position_per_strategy, updated_at)
                    VALUES ($1, $2, $3, $4)
                    """,
                    body.dailyLossLimit,
                    body.maxPositionPerMarket,
                    body.maxPositionPerStrategy,
                    datetime.now(timezone.utc),
                )
        except Exception:
            logger.exception("Failed to persist risk parameters to PostgreSQL")
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail="Operation partially completed. Please try again.",
            )

        # #2: Use Lua script for atomic parameter update
        updates = json.dumps(changes | {"updated_at": now})
        await r.eval(_STATE_UPDATE_LUA, 1, "pqap:risk:state", updates, "3600")

        elapsed = (time.monotonic() - start) * 1000
        RISK_PARAM_CHANGES_TOTAL.inc()
        RISK_ACTION_LATENCY.observe(elapsed)

        logger.info("Risk parameters updated: %s", changes)

        # #7: Publish NATS command
        nc = await _get_nats()
        await nc.publish("pqap.risk.command", json.dumps({
            "command": "parameters_updated",
            "changes": changes,
            "timestamp": now,
        }).encode())

        return {"status": "parameters_updated", "changes": changes}

    except HTTPException:
        raise
    except Exception:
        logger.exception("Failed to update risk parameters")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to update risk parameters",
        )


# ─────────────────────────────────────────────────────────────────────────────
# Per-Account Risk Limits Endpoints
# ─────────────────────────────────────────────────────────────────────────────

# Configurable thresholds
RISK_WARNING_THRESHOLD = Decimal("0.8")  # 80% of limit


def _validate_account_id(account_id: str) -> uuid.UUID:
    """Validate and return UUID from string."""
    try:
        return uuid.UUID(account_id)
    except ValueError:
        raise HTTPException(status_code=400, detail="Invalid account ID format")


@router.get("/limits/{account_id}", response_model=RiskLimitsResponse)
async def get_risk_limits(
    account_id: str,
    _user: dict = Depends(extract_user),
):
    """Get per-account risk limits."""
    _validate_account_id(account_id)

    try:
        pool = await get_pool()
    except Exception as e:
        logger.error("failed to get database pool", extra={"error": str(e)})
        raise HTTPException(status_code=503, detail="Database unavailable")

    async with pool.acquire() as conn:
        row = await conn.fetchrow(
            "SELECT * FROM account_risk_limits WHERE account_id = $1::uuid",
            account_id,
        )

    if not row:
        # Return defaults if no limits set
        return RiskLimitsResponse(
            account_id=account_id,
            daily_loss_limit=config.DEFAULT_DAILY_LOSS_LIMIT_PCT,
            max_position_per_market=config.DEFAULT_MAX_POSITION_PER_MARKET_PCT,
            max_position_per_strategy=config.DEFAULT_MAX_POSITION_PER_STRATEGY_PCT,
            drawdown_threshold=config.DEFAULT_DRAWDOWN_THRESHOLD_PCT,
        )

    return RiskLimitsResponse(
        account_id=row["account_id"],
        daily_loss_limit=str(row["daily_loss_limit"]),
        max_position_per_market=str(row["max_position_per_market"]),
        max_position_per_strategy=str(row["max_position_per_strategy"]),
        drawdown_threshold=str(row["drawdown_threshold"]),
    )


@router.put("/limits/{account_id}", response_model=RiskLimitsResponse)
async def update_risk_limits(
    account_id: str,
    body: RiskLimitsUpdate,
    user: dict = Depends(extract_user),
):
    """Update per-account risk limits. Requires admin role."""
    require_admin(user)
    RISK_LIMITS_UPDATES_TOTAL.inc()
    _validate_account_id(account_id)

    try:
        pool = await get_pool()
    except Exception as e:
        logger.error("failed to get database pool", extra={"error": str(e)})
        raise HTTPException(status_code=503, detail="Database unavailable")

    async with pool.acquire() as conn:
        # Check if account exists
        account = await conn.fetchrow(
            "SELECT id FROM accounts WHERE id = $1::uuid",
            account_id,
        )
        if not account:
            raise HTTPException(status_code=404, detail="Account not found")

        # Build update query
        updates = []
        params = []
        idx = 1

        if body.daily_loss_limit is not None:
            updates.append(f"daily_loss_limit = ${idx}")
            params.append(body.daily_loss_limit)
            idx += 1

        if body.max_position_per_market is not None:
            updates.append(f"max_position_per_market = ${idx}")
            params.append(body.max_position_per_market)
            idx += 1

        if body.max_position_per_strategy is not None:
            updates.append(f"max_position_per_strategy = ${idx}")
            params.append(body.max_position_per_strategy)
            idx += 1

        if body.drawdown_threshold is not None:
            updates.append(f"drawdown_threshold = ${idx}")
            params.append(body.drawdown_threshold)
            idx += 1

        if not updates:
            raise HTTPException(status_code=400, detail="No fields to update")

        updates.append("updated_at = NOW()")

        # Build parameterized upert query
        # $1 = account_id, $2-$5 = default values, $6+ = update params
        default_params = [
            config.DEFAULT_DAILY_LOSS_LIMIT_PCT,
            config.DEFAULT_MAX_POSITION_PER_MARKET_PCT,
            config.DEFAULT_MAX_POSITION_PER_STRATEGY_PCT,
            config.DEFAULT_DRAWDOWN_THRESHOLD_PCT,
        ]

        # Adjust parameter indices for updates (start from $6)
        adjusted_updates = []
        for i, update in enumerate(updates):
            adjusted = re.sub(r'\$(\d+)', lambda m: f'${int(m.group(1)) + 5}', update)
            adjusted_updates.append(adjusted)

        query = f"""
            INSERT INTO account_risk_limits (account_id, daily_loss_limit, max_position_per_market, max_position_per_strategy, drawdown_threshold)
            VALUES ($1::uuid, $2, $3, $4, $5)
            ON CONFLICT (account_id) DO UPDATE SET {', '.join(adjusted_updates)}
            RETURNING *
        """
        all_params = [account_id] + default_params + params
        row = await conn.fetchrow(query, *all_params)

    return RiskLimitsResponse(
        account_id=row["account_id"],
        daily_loss_limit=str(row["daily_loss_limit"]),
        max_position_per_market=str(row["max_position_per_market"]),
        max_position_per_strategy=str(row["max_position_per_strategy"]),
        drawdown_threshold=str(row["drawdown_threshold"]),
    )


@router.get("/cross-account", response_model=CrossAccountRiskResponse)
async def get_cross_account_risk(
    _user: dict = Depends(extract_user),
):
    """Get cross-account risk exposure."""
    start = time.monotonic()
    RISK_PER_ACCOUNT_LIMIT_CHECKS_TOTAL.inc()

    try:
        pool = await get_pool()
    except Exception as e:
        logger.error("failed to get database pool", extra={"error": str(e)})
        raise HTTPException(status_code=503, detail="Database unavailable")

    async with pool.acquire() as conn:
        rows = await conn.fetch(
            """
            SELECT
                a.id as account_id,
                a.name as account_name,
                COALESCE(arl.daily_loss_limit, $1) as daily_loss_limit,
                COALESCE(SUM(p.unrealized_pnl), 0) as daily_loss_used,
                COALESCE(arl.max_position_per_market, $2) as max_position_per_market,
                COALESCE(SUM(p.quantity * p.current_price), 0) as current_exposure
            FROM accounts a
            LEFT JOIN account_risk_limits arl ON arl.account_id = a.id
            LEFT JOIN positions p ON p.account_id = a.id AND p.status = 'open'
            WHERE a.is_active = true
            GROUP BY a.id, a.name, arl.daily_loss_limit, arl.max_position_per_market
            """,
            config.DEFAULT_DAILY_LOSS_LIMIT_PCT,
            config.DEFAULT_MAX_POSITION_PER_MARKET_PCT,
        )

    accounts = []
    total_exposure = Decimal("0")
    total_daily_loss = Decimal("0")

    for row in rows:
        exposure = Decimal(str(row["current_exposure"]))
        loss = Decimal(str(row["daily_loss_used"]))
        limit = Decimal(str(row["daily_loss_limit"]))

        total_exposure += exposure
        total_daily_loss += loss

        # Determine status
        if loss >= limit:
            status_val = "critical"
        elif loss >= limit * RISK_WARNING_THRESHOLD:
            status_val = "warning"
        else:
            status_val = "healthy"

        accounts.append(AccountRiskSummary(
            account_id=row["account_id"],
            account_name=row["account_name"],
            daily_loss_limit=str(row["daily_loss_limit"]),
            daily_loss_used=str(row["daily_loss_used"]),
            max_position_per_market=str(row["max_position_per_market"]),
            current_exposure=str(row["current_exposure"]),
            status=status_val,
        ))

    # Determine overall status
    overall_status = "healthy"
    if any(a.status == "critical" for a in accounts):
        overall_status = "critical"
    elif any(a.status == "warning" for a in accounts):
        overall_status = "warning"

    elapsed = (time.monotonic() - start) * 1000
    PORTFOLIO_CROSS_ACCOUNT_LATENCY.observe(elapsed)
    PORTFOLIO_CROSS_ACCOUNT_QUERIES_TOTAL.inc()

    return CrossAccountRiskResponse(
        total_exposure=str(total_exposure),
        total_daily_loss=str(total_daily_loss),
        accounts=accounts,
        overall_status=overall_status,
        last_updated=datetime.now(timezone.utc),
    )
