from datetime import datetime
from typing import Optional

from pydantic import BaseModel, Field


class SimulationConfig(BaseModel):
    slippage_pct: float = Field(default=0.01, ge=0, le=0.1)
    partial_fill_probability: float = Field(default=0.1, ge=0, le=1)
    latency_ms: int = Field(default=100, ge=0, le=10000)
    min_fill_ratio: float = Field(default=0.5, ge=0, le=1)
    rng_seed: int = Field(default=42)


class BacktestRequest(BaseModel):
    strategy_id: str = Field(min_length=1, max_length=64)
    start_date: str = Field(pattern=r"^\d{4}-\d{2}-\d{2}")  # #18: Validate date format
    end_date: str = Field(pattern=r"^\d{4}-\d{2}-\d{2}")    # #18: Validate date format
    simulation: SimulationConfig = SimulationConfig()


class BacktestStatus(BaseModel):
    run_id: str
    status: str
    progress: Optional[str] = None
    started_at: Optional[datetime] = None
    completed_at: Optional[datetime] = None
    error_message: Optional[str] = None


class BacktestTrade(BaseModel):
    timestamp: str
    market_id: str
    side: str
    price: str
    quantity: str
    slippage: str
    pnl: str
    lookahead_warning: bool = False


class BacktestSummary(BaseModel):
    total_pnl: str
    total_trades: int
    win_rate: str
    sharpe_ratio: str
    max_drawdown: str
    profit_factor: Optional[str]


class BacktestResults(BaseModel):
    summary: BacktestSummary
    trades: list[BacktestTrade]
    warnings: list[dict]
