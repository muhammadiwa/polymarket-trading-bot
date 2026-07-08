import logging
import secrets
import time
from datetime import datetime, timedelta, timezone

from fastapi import APIRouter, Depends, HTTPException, Path, status

from app.db import get_pool
from app.middleware.auth import extract_user, require_admin
from app.metrics import (
    ADMIN_BACKUP_TOTAL,
    ADMIN_BACKUP_DURATION,
    ADMIN_RESTORE_TOTAL,
    ADMIN_CLEANUP_TOTAL,
    ADMIN_CLEANUP_ROWS_DELETED,
)
from app.models.logs import (
    BackupInfoResponse,
    BackupListResponse,
    CleanupRequest,
    CleanupResponse,
    DatabaseStatsResponse,
    RestoreRequest,
)
from app.services.database import database_service

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/admin/database", tags=["admin-database"])

# In-memory store for confirmation tokens (in production, use Redis)
_restore_tokens: dict[str, dict] = {}


@router.post("/backup", response_model=BackupInfoResponse)
async def create_backup(user: dict = Depends(extract_user)):
    """Create a database backup. Requires admin role."""
    require_admin(user)
    ADMIN_BACKUP_TOTAL.inc()

    start_time = time.monotonic()
    try:
        result = await database_service.create_backup(triggered_by="manual")
        duration = (time.monotonic() - start_time) * 1000
        ADMIN_BACKUP_DURATION.observe(duration)

        return BackupInfoResponse(
            id=result["id"],
            filename=result["filename"],
            file_path=result["file_path"],
            size_bytes=result["size_bytes"],
            created_at=datetime.now(timezone.utc),
            status=result["status"],
            duration_ms=result["duration_ms"],
            triggered_by=result["triggered_by"],
        )
    except Exception as e:
        logger.error(f"Backup failed: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Backup failed: {str(e)}",
        )


@router.get("/backups", response_model=BackupListResponse)
async def list_backups(user: dict = Depends(extract_user)):
    """List all backups. Requires admin role."""
    require_admin(user)

    backups = await database_service.list_backups()
    return BackupListResponse(
        backups=[
            BackupInfoResponse(
                id=b["id"],
                filename=b["filename"],
                file_path=b["file_path"],
                size_bytes=b["size_bytes"],
                created_at=b["created_at"],
                status=b["status"],
                duration_ms=b["duration_ms"],
                triggered_by=b["triggered_by"],
                error_message=b["error_message"],
            )
            for b in backups
        ],
        total=len(backups),
    )


@router.post("/restore/{backup_id}/confirm-token")
async def get_restore_confirm_token(
    backup_id: str = Path(...),
    user: dict = Depends(extract_user),
):
    """Generate confirmation token for restore operation. Requires admin role."""
    require_admin(user)

    # Verify backup exists
    from app.db import get_pool
    pool = await get_pool()
    async with pool.acquire() as conn:
        backup = await conn.fetchrow(
            "SELECT * FROM database_backups WHERE id = $1 AND status = 'completed'",
            backup_id,
        )

    if not backup:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Backup not found or not completed",
        )

    # Generate token
    token = secrets.token_urlsafe(32)
    _restore_tokens[token] = {
        "backup_id": backup_id,
        "user_id": user.get("user_id"),
        "expires_at": datetime.now(timezone.utc) + timedelta(minutes=5),
    }

    return {"confirmationToken": token}


@router.post("/restore/{backup_id}")
async def restore_backup(
    backup_id: str = Path(...),
    body: RestoreRequest = ...,
    user: dict = Depends(extract_user),
):
    """Restore database from backup. Requires admin role and confirmation token."""
    require_admin(user)
    ADMIN_RESTORE_TOTAL.inc()

    # Validate token
    token_data = _restore_tokens.get(body.confirmation_token)
    if not token_data:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Invalid or expired confirmation token",
        )

    if token_data["backup_id"] != backup_id:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Token does not match backup ID",
        )

    if datetime.now(timezone.utc) > token_data["expires_at"]:
        del _restore_tokens[body.confirmation_token]
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Confirmation token expired",
        )

    # Remove used token
    del _restore_tokens[body.confirmation_token]

    try:
        result = await database_service.restore_backup(backup_id)
        return {"status": "restored", "backup_id": backup_id}
    except Exception as e:
        logger.error(f"Restore failed: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Restore failed: {str(e)}",
        )


@router.post("/cleanup", response_model=CleanupResponse)
async def cleanup_database(
    body: CleanupRequest,
    user: dict = Depends(extract_user),
):
    """Clean up old data based on retention policy. Requires admin role."""
    require_admin(user)
    ADMIN_CLEANUP_TOTAL.inc()

    try:
        result = await database_service.cleanup_old_data(
            retention_days=body.retention_days,
            tables=body.tables,
        )

        # Track deleted rows
        for table, count in result["deleted_rows"].items():
            ADMIN_CLEANUP_ROWS_DELETED.labels(table=table).inc(count)

        return CleanupResponse(
            deleted_rows=result["deleted_rows"],
            freed_bytes=result["freed_bytes"],
        )
    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e),
        )
    except Exception as e:
        logger.error(f"Cleanup failed: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Cleanup failed: {str(e)}",
        )


@router.get("/stats", response_model=DatabaseStatsResponse)
async def get_database_stats(user: dict = Depends(extract_user)):
    """Get database statistics. Requires admin role."""
    require_admin(user)

    try:
        stats = await database_service.get_database_stats()
        return DatabaseStatsResponse(
            total_size_bytes=stats["total_size_bytes"],
            table_sizes=stats["table_sizes"],
            oldest_log_timestamp=stats["oldest_log_timestamp"],
            newest_log_timestamp=stats["newest_log_timestamp"],
            total_log_entries=stats["total_log_entries"],
            total_trades=stats["total_trades"],
            total_positions=stats["total_positions"],
        )
    except Exception as e:
        logger.error(f"Failed to get database stats: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get database stats: {str(e)}",
        )
