import json
import logging
import re
from datetime import datetime, timezone
from typing import Optional

import asyncpg
from fastapi import APIRouter, Depends, HTTPException, Path, Query, status
from passlib.context import CryptContext
from pydantic import BaseModel, Field

from app.config import config
from app.db import get_pool
from app.middleware.auth import extract_user, require_admin
from app.metrics import ADMIN_CONFIG_CHANGES_TOTAL, ADMIN_CONFIG_VALIDATION_ERRORS_TOTAL
from app.models.config import (
    ConfigAuditLogListResponse,
    ConfigAuditLogResponse,
    SystemConfigListResponse,
    SystemConfigResponse,
    SystemConfigUpdate,
    mask_sensitive_value,
    validate_config_value,
)

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/admin", tags=["admin"])

pwd_context = CryptContext(
    schemes=["bcrypt"],
    deprecated="auto",
    bcrypt__rounds=config.BCRYPT_COST,
)


class CreateUserRequest(BaseModel):
    username: str = Field(min_length=1, max_length=64)
    password: str = Field(min_length=12, max_length=128)
    role: str = "viewer"


class UserResponse(BaseModel):
    id: str
    username: str
    role: str


@router.post("/users", response_model=UserResponse)
async def create_user(body: CreateUserRequest, user: dict = Depends(extract_user)):
    require_admin(user)

    if body.role not in ("admin", "viewer"):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Role must be 'admin' or 'viewer'",
        )

    password_hash = pwd_context.hash(body.password)

    pool = await get_pool()
    async with pool.acquire() as conn:
        try:
            row = await conn.fetchrow(
                "INSERT INTO users (username, password_hash, role) VALUES ($1, $2, $3) RETURNING id, username, role",
                body.username,
                password_hash,
                body.role,
            )
        except asyncpg.UniqueViolationError:
            raise HTTPException(
                status_code=status.HTTP_409_CONFLICT,
                detail="Username already exists",
            )

    return UserResponse(id=str(row["id"]), username=row["username"], role=row["role"])


@router.get("/users", response_model=list[UserResponse])
async def list_users(user: dict = Depends(extract_user)):
    require_admin(user)

    pool = await get_pool()
    async with pool.acquire() as conn:
        rows = await conn.fetch(
            "SELECT id, username, role FROM users ORDER BY created_at DESC"
        )

    return [UserResponse(id=str(r["id"]), username=r["username"], role=r["role"]) for r in rows]


# ─────────────────────────────────────────────────────────────────────────────
# System Configuration Endpoints
# ─────────────────────────────────────────────────────────────────────────────

# Config key validation pattern: lowercase letters, numbers, underscores
CONFIG_KEY_PATTERN = r"^[a-z][a-z0-9_]{0,99}$"


def _validate_config_key(key: str) -> str:
    """Validate config key format."""
    if not re.match(CONFIG_KEY_PATTERN, key):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Invalid config key format: '{key}'. Must be lowercase alphanumeric with underscores, starting with a letter.",
        )
    return key


@router.get("/config", response_model=SystemConfigListResponse)
async def list_configs(
    category: Optional[str] = Query(None, pattern=r"^(api_keys|risk_defaults|notification_settings)$"),
    user: dict = Depends(extract_user),
):
    """List all system configurations. Sensitive values are masked."""
    try:
        pool = await get_pool()
    except Exception as e:
        logger.error("failed to get database pool", extra={"error": str(e)})
        raise HTTPException(status_code=503, detail="Database unavailable")
    async with pool.acquire() as conn:
        if category:
            rows = await conn.fetch(
                "SELECT * FROM system_config WHERE category = $1 ORDER BY config_key",
                category,
            )
        else:
            rows = await conn.fetch("SELECT * FROM system_config ORDER BY category, config_key")

    configs = []
    for row in rows:
        config_value = row["config_value"]
        # Mask sensitive values for non-admin or default view
        if row["is_sensitive"] and user.get("role") != "admin":
            if isinstance(config_value, str):
                config_value = mask_sensitive_value(config_value)

        configs.append(
            SystemConfigResponse(
                id=row["id"],
                config_key=row["config_key"],
                config_value=config_value,
                category=row["category"],
                description=row["description"],
                is_sensitive=row["is_sensitive"],
                created_at=row["created_at"],
                updated_at=row["updated_at"],
                updated_by=row["updated_by"],
            )
        )

    return SystemConfigListResponse(configs=configs, total=len(configs))


@router.get("/config/{key}")
async def get_config(
    key: str = Path(..., min_length=1, max_length=100),
    unmask: bool = Query(False),
    user: dict = Depends(extract_user),
):
    """Get a specific config value. Sensitive values masked unless unmask=true (admin only)."""
    _validate_config_key(key)

    try:
        pool = await get_pool()
    except Exception as e:
        logger.error("failed to get database pool", extra={"error": str(e)})
        raise HTTPException(status_code=503, detail="Database unavailable")
    async with pool.acquire() as conn:
        row = await conn.fetchrow(
            "SELECT * FROM system_config WHERE config_key = $1", key
        )

    if not row:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Config key '{key}' not found",
        )

    config_value = row["config_value"]

    # Only admins can unmask sensitive values
    if row["is_sensitive"] and unmask:
        require_admin(user)
    elif row["is_sensitive"] and not unmask:
        if isinstance(config_value, str):
            config_value = mask_sensitive_value(config_value)

    return SystemConfigResponse(
        id=row["id"],
        config_key=row["config_key"],
        config_value=config_value,
        category=row["category"],
        description=row["description"],
        is_sensitive=row["is_sensitive"],
        created_at=row["created_at"],
        updated_at=row["updated_at"],
        updated_by=row["updated_by"],
    )


@router.put("/config/{key}", response_model=SystemConfigResponse)
async def update_config(
    key: str = Path(..., min_length=1, max_length=100),
    body: SystemConfigUpdate = ...,
    user: dict = Depends(extract_user),
):
    """Update a config value. Requires admin role. Validates before save."""
    _validate_config_key(key)
    require_admin(user)

    try:
        pool = await get_pool()
    except Exception as e:
        logger.error("failed to get database pool", extra={"error": str(e)})
        raise HTTPException(status_code=503, detail="Database unavailable")
    async with pool.acquire() as conn:
        # Get current config
        row = await conn.fetchrow(
            "SELECT * FROM system_config WHERE config_key = $1", key
        )

        if not row:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Config key '{key}' not found",
            )

        # Optimistic locking check - always enforce if timestamp provided
        if body.expected_updated_at is not None:
            if row["updated_at"] != body.expected_updated_at:
                raise HTTPException(
                    status_code=status.HTTP_409_CONFLICT,
                    detail="Config was modified by another request. Please refresh and try again.",
                )

        # Validate the new value
        try:
            validated_value = validate_config_value(
                row["category"], key, body.config_value
            )
        except ValueError as e:
            ADMIN_CONFIG_VALIDATION_ERRORS_TOTAL.inc()
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=str(e),
            )

        # Truncate reason if too long
        reason = body.reason[:500] if body.reason else None

        # Update config and create audit log in transaction
        async with conn.transaction():
            # Update config with optimistic locking in WHERE clause
            updated_row = await conn.fetchrow(
                """
                UPDATE system_config
                SET config_value = $1, updated_at = NOW(), updated_by = $2
                WHERE config_key = $3 AND updated_at = $4
                RETURNING *
                """,
                json.dumps(validated_value),
                user.get("user_id"),
                key,
                row["updated_at"],
            )

            if not updated_row:
                raise HTTPException(
                    status_code=status.HTTP_409_CONFLICT,
                    detail="Config was modified by another request. Please refresh and try again.",
                )

            # Create audit log
            await conn.execute(
                """
                INSERT INTO config_audit_log (config_key, old_value, new_value, changed_by, reason)
                VALUES ($1, $2, $3, $4, $5)
                """,
                key,
                json.dumps(row["config_value"]),
                json.dumps(validated_value),
                user.get("user_id"),
                reason,
            )

        ADMIN_CONFIG_CHANGES_TOTAL.inc()

        return SystemConfigResponse(
            id=updated_row["id"],
            config_key=updated_row["config_key"],
            config_value=updated_row["config_value"],
            category=updated_row["category"],
            description=updated_row["description"],
            is_sensitive=updated_row["is_sensitive"],
            created_at=updated_row["created_at"],
            updated_at=updated_row["updated_at"],
            updated_by=updated_row["updated_by"],
        )


@router.get("/config/audit/logs", response_model=ConfigAuditLogListResponse)
async def get_config_audit_logs(
    key: Optional[str] = Query(None, max_length=100),
    limit: int = Query(50, ge=1, le=200),
    offset: int = Query(0, ge=0),
    user: dict = Depends(extract_user),
):
    """Get config change audit logs. Requires admin role."""
    require_admin(user)
    if key:
        _validate_config_key(key)

    try:
        pool = await get_pool()
    except Exception as e:
        logger.error("failed to get database pool", extra={"error": str(e)})
        raise HTTPException(status_code=503, detail="Database unavailable")
    async with pool.acquire() as conn:
        if key:
            rows = await conn.fetch(
                """
                SELECT * FROM config_audit_log
                WHERE config_key = $1
                ORDER BY changed_at DESC
                LIMIT $2 OFFSET $3
                """,
                key,
                limit,
                offset,
            )
            count_row = await conn.fetchrow(
                "SELECT COUNT(*) as total FROM config_audit_log WHERE config_key = $1",
                key,
            )
        else:
            rows = await conn.fetch(
                """
                SELECT * FROM config_audit_log
                ORDER BY changed_at DESC
                LIMIT $1 OFFSET $2
                """,
                limit,
                offset,
            )
            count_row = await conn.fetchrow(
                "SELECT COUNT(*) as total FROM config_audit_log"
            )

    logs = [
        ConfigAuditLogResponse(
            id=row["id"],
            config_key=row["config_key"],
            old_value=row["old_value"],
            new_value=row["new_value"],
            changed_by=row["changed_by"],
            changed_at=row["changed_at"],
            reason=row["reason"],
        )
        for row in rows
    ]

    return ConfigAuditLogListResponse(
        logs=logs, total=count_row["total"]
    )
