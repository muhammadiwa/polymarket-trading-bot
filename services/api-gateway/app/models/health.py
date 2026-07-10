from typing import Optional

from pydantic import BaseModel, ConfigDict
from pydantic.alias_generators import to_camel


class ServiceHealth(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    name: str
    status: str  # "up" | "down" | "degraded"
    ws_connected: bool = False
    cpu_percent: float = 0.0
    memory_mb: float = 0.0
    memory_limit_mb: float = 0.0
    error_rate: float = 0.0
    last_heartbeat: Optional[str] = None


class SystemHealthResponse(BaseModel):
    scanner: ServiceHealth
    arbEngine: ServiceHealth
    executionEngine: ServiceHealth
    riskManager: ServiceHealth
    positionManager: ServiceHealth
    backtesting: Optional[ServiceHealth] = None
    aiOptimizer: Optional[ServiceHealth] = None
    accountManager: Optional[ServiceHealth] = None
    overall: str  # "healthy" | "degraded" | "unhealthy"
    lastUpdated: str

    model_config = {"populate_by_name": True}
