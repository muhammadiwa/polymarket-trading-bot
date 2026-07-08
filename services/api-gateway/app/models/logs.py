from datetime import datetime
from enum import Enum
from typing import Any, Optional
from uuid import UUID

from pydantic import BaseModel, ConfigDict, Field
from pydantic.alias_generators import to_camel


class LogLevel(str, Enum):
    DEBUG = "debug"
    INFO = "info"
    WARN = "warn"
    ERROR = "error"
    FATAL = "fatal"


class SystemLogCreate(BaseModel):
    level: LogLevel
    service: str = Field(..., max_length=50)
    request_id: Optional[str] = Field(None, max_length=100)
    message: str = Field(..., max_length=10000)
    context: Optional[dict[str, Any]] = None


class SystemLogResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    id: UUID
    timestamp: datetime
    level: LogLevel
    service: str
    request_id: Optional[str] = None
    message: str
    context: Optional[dict[str, Any]] = None


class LogQueryResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    logs: list[SystemLogResponse]
    total: int
    has_more: bool


class BackupInfoResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    id: UUID
    filename: str
    file_path: str
    size_bytes: int
    created_at: datetime
    status: str
    duration_ms: Optional[int] = None
    triggered_by: str = "manual"
    error_message: Optional[str] = None


class BackupListResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    backups: list[BackupInfoResponse]
    total: int


class CleanupRequest(BaseModel):
    retention_days: int = Field(..., ge=1, le=365)
    tables: Optional[list[str]] = None


class CleanupResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    deleted_rows: dict[str, int]
    freed_bytes: int


class RestoreRequest(BaseModel):
    confirmation_token: str


class DatabaseStatsResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    total_size_bytes: int
    table_sizes: dict[str, int]
    oldest_log_timestamp: Optional[datetime] = None
    newest_log_timestamp: Optional[datetime] = None
    total_log_entries: int
    total_trades: int
    total_positions: int
