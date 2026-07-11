import asyncio
import gzip
import logging
import os
import time
from datetime import datetime, timezone
from pathlib import Path
from urllib.parse import urlparse

from app.config import config

logger = logging.getLogger(__name__)


class DatabaseService:
    """Service for database backup, restore, and cleanup operations."""

    def __init__(self):
        self.backup_dir = Path(config.BACKUP_DIR)
        self.backup_dir.mkdir(parents=True, exist_ok=True)

    def _get_backup_path(self, filename: str) -> Path:
        """Get full path for backup file."""
        return self.backup_dir / filename

    def _generate_filename(self) -> str:
        """Generate backup filename with timestamp."""
        now = datetime.now(timezone.utc)
        return f"pqap_backup_{now.strftime('%Y%m%d_%H%M%S')}.sql.gz"

    async def create_backup(self, triggered_by: str = "manual") -> dict:
        """Create a database backup using pg_dump."""
        from app.db import get_pool

        filename = self._generate_filename()
        file_path = self._get_backup_path(filename)
        start_time = time.monotonic()

        pool = await get_pool()
        async with pool.acquire() as conn:
            # Insert backup record
            backup_row = await conn.fetchrow(
                """
                INSERT INTO database_backups (filename, file_path, status, triggered_by)
                VALUES ($1, $2, 'in_progress', $3)
                RETURNING id
                """,
                filename,
                str(file_path),
                triggered_by,
            )
            backup_id = backup_row["id"]

        try:
            # Parse connection URL to extract components
            parsed = urlparse(config.POSTGRES_URL)
            db_name = parsed.path.lstrip("/")
            db_host = parsed.hostname or "localhost"
            db_port = str(parsed.port or 5432)
            db_user = parsed.username or "postgres"
            db_password = parsed.password or ""

            # Run pg_dump with PGPASSWORD env var (more secure than URL)
            pg_dump_cmd = [
                "pg_dump",
                "-h", db_host,
                "-p", db_port,
                "-U", db_user,
                db_name,
                "--format=custom",
                "--compress=6",
            ]

            # Set PGPASSWORD in environment
            env = os.environ.copy()
            env["PGPASSWORD"] = db_password

            # Run in subprocess
            process = await asyncio.create_subprocess_exec(
                *pg_dump_cmd,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                env=env,
            )

            stdout, stderr = await process.communicate()

            if process.returncode != 0:
                error_msg = stderr.decode() if stderr else "Unknown error"
                raise Exception(f"pg_dump failed: {error_msg}")

            # Compress and save
            with gzip.open(file_path, "wb") as f:
                f.write(stdout)

            # Get file size
            size_bytes = file_path.stat().st_size
            duration_ms = int((time.monotonic() - start_time) * 1000)

            # Update backup record
            pool = await get_pool()
            async with pool.acquire() as conn:
                await conn.execute(
                    """
                    UPDATE database_backups
                    SET status = 'completed', size_bytes = $1, duration_ms = $2, completed_at = NOW()
                    WHERE id = $3
                    """,
                    size_bytes,
                    duration_ms,
                    backup_id,
                )

            return {
                "id": str(backup_id),
                "filename": filename,
                "file_path": str(file_path),
                "size_bytes": size_bytes,
                "status": "completed",
                "duration_ms": duration_ms,
                "triggered_by": triggered_by,
            }

        except Exception as e:
            duration_ms = int((time.monotonic() - start_time) * 1000)
            error_msg = str(e)[:1000]

            # Update backup record with error
            pool = await get_pool()
            async with pool.acquire() as conn:
                await conn.execute(
                    """
                    UPDATE database_backups
                    SET status = 'failed', duration_ms = $1, error_message = $2, completed_at = NOW()
                    WHERE id = $3
                    """,
                    duration_ms,
                    error_msg,
                    backup_id,
                )

            # Clean up failed backup file
            if file_path.exists():
                file_path.unlink()

            raise

    async def restore_backup(self, backup_id: str) -> dict:
        """Restore database from backup."""
        from app.db import get_pool

        pool = await get_pool()
        async with pool.acquire() as conn:
            # Get backup info
            backup = await conn.fetchrow(
                "SELECT * FROM database_backups WHERE id = $1 AND status = 'completed'",
                backup_id,
            )

            if not backup:
                raise Exception("Backup not found or not completed")

            file_path = Path(backup["file_path"])
            if not file_path.exists():
                raise Exception("Backup file not found on disk")

        # Decompress and restore
        temp_file = file_path.with_suffix("")
        try:
            # Decompress
            with gzip.open(file_path, "rb") as f:
                dump_data = f.read()

            # Write to temporary file
            with open(temp_file, "wb") as f:
                f.write(dump_data)

            # Parse POSTGRES_URL to avoid exposing credentials in process list
            from urllib.parse import urlparse
            parsed = urlparse(config.POSTGRES_URL)
            pg_host = parsed.hostname or "localhost"
            pg_port = str(parsed.port or 5432)
            pg_user = parsed.username or "postgres"
            pg_dbname = parsed.path.lstrip("/") or "pqap"

            # Run pg_restore with PGPASSWORD env var
            pg_restore_cmd = [
                "pg_restore",
                "--clean",
                "--if-exists",
                "-h", pg_host,
                "-p", pg_port,
                "-U", pg_user,
                "-d", pg_dbname,
                str(temp_file),
            ]

            env = {"PGPASSWORD": parsed.password or ""}
            process = await asyncio.create_subprocess_exec(
                *pg_restore_cmd,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                env={**dict(__import__("os").environ), **env},
            )

            stdout, stderr = await process.communicate()

            if process.returncode != 0:
                error_msg = stderr.decode() if stderr else "Unknown error"
                raise Exception(f"pg_restore failed: {error_msg}")

            return {"status": "restored", "backup_id": backup_id}

        except Exception as e:
            raise
        finally:
            # Clean up temp file in all cases
            if temp_file.exists():
                temp_file.unlink()

    async def cleanup_old_data(self, retention_days: int, tables: list[str] = None) -> dict:
        """Clean up old data based on retention policy."""
        from app.db import get_pool

        eligible_tables = {
            "system_logs": "timestamp",
            "trades": "created_at",
            "opportunities": "timestamp",
            "risk_events": "created_at",
            "config_audit_log": "changed_at",
            "notifications": "created_at",
        }

        if tables:
            # Validate requested tables
            invalid_tables = set(tables) - set(eligible_tables.keys())
            if invalid_tables:
                raise ValueError(f"Invalid tables: {invalid_tables}")
            target_tables = {t: eligible_tables[t] for t in tables}
        else:
            target_tables = eligible_tables

        deleted_rows = {}
        pool = await get_pool()

        async with pool.acquire() as conn:
            for table, timestamp_col in target_tables.items():
                try:
                    # Table and column names are from our whitelist, not user input
                    result = await conn.execute(
                        f"""
                        DELETE FROM {table}
                        WHERE {timestamp_col} < NOW() - $1::interval
                        """,
                        f"{retention_days} days",
                    )
                    # Parse "DELETE N" response
                    count = int(result.split()[-1]) if result else 0
                    deleted_rows[table] = count
                except Exception as e:
                    logger.warning(f"Failed to cleanup {table}: {e}")
                    deleted_rows[table] = 0

        # Estimate freed bytes (rough estimate)
        freed_bytes = sum(deleted_rows.values()) * 1024  # Assume 1KB per row

        return {
            "deleted_rows": deleted_rows,
            "freed_bytes": freed_bytes,
        }

    async def get_database_stats(self) -> dict:
        """Get database statistics."""
        from app.db import get_pool

        pool = await get_pool()
        async with pool.acquire() as conn:
            # Get total database size
            size_row = await conn.fetchrow("SELECT pg_database_size(current_database()) as size")
            total_size = size_row["size"]

            # Get table sizes
            table_sizes = {}
            tables = ["system_logs", "trades", "positions", "opportunities", "risk_events"]
            for table in tables:
                try:
                    row = await conn.fetchrow(f"SELECT pg_total_relation_size('{table}') as size")
                    table_sizes[table] = row["size"]
                except Exception:
                    table_sizes[table] = 0

            # Get log stats
            log_stats = await conn.fetchrow(
                """
                SELECT
                    COUNT(*) as total,
                    MIN(timestamp) as oldest,
                    MAX(timestamp) as newest
                FROM system_logs
                """
            )

            # Get trade count
            trade_row = await conn.fetchrow("SELECT COUNT(*) as total FROM trades")
            position_row = await conn.fetchrow("SELECT COUNT(*) as total FROM positions")

        return {
            "total_size_bytes": total_size,
            "table_sizes": table_sizes,
            "oldest_log_timestamp": log_stats["oldest"],
            "newest_log_timestamp": log_stats["newest"],
            "total_log_entries": log_stats["total"],
            "total_trades": trade_row["total"],
            "total_positions": position_row["total"],
        }

    async def list_backups(self) -> list[dict]:
        """List all backups."""
        from app.db import get_pool

        pool = await get_pool()
        async with pool.acquire() as conn:
            rows = await conn.fetch(
                "SELECT * FROM database_backups ORDER BY created_at DESC"
            )

        return [
            {
                "id": str(row["id"]),
                "filename": row["filename"],
                "file_path": row["file_path"],
                "size_bytes": row["size_bytes"],
                "status": row["status"],
                "duration_ms": row["duration_ms"],
                "triggered_by": row["triggered_by"],
                "error_message": row["error_message"],
                "created_at": row["created_at"],
                "completed_at": row["completed_at"],
            }
            for row in rows
        ]


# Singleton instance
database_service = DatabaseService()
