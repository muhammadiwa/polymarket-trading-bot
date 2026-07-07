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
    start_date: str = Field(pattern=r"^\d{4}-\d{2}-\d{2}$")
    end_date: str = Field(pattern=r"^\d{4}-\d{2}-\d{2}$")
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
    var_95: Optional[str] = None  # #1: Added VaR


class BacktestResults(BaseModel):
    summary: BacktestSummary
    trades: list[BacktestTrade]
    warnings: list[dict]
    daily_pnl: Optional[list[dict]] = None  # #1: Daily PnL breakdown


class BacktestReport(BaseModel):
    """#2: Full report with curves and breakdown."""
    run_id: str
    summary: BacktestSummary
    pnl_curve: list[dict]  # [{date, cumulative_pnl}]
    drawdown_curve: list[dict]  # [{date, drawdown}]
    trades: list[BacktestTrade]
    warnings: list[dict]


class SweepParameter(BaseModel):
    """#3: Parameter sweep configuration."""
    name: str  # e.g., "slippage_pct"
    min_value: float
    max_value: float
    step: float = Field(gt=0)  # #2: Prevent division by zero


class SweepRequest(BaseModel):
    """#3: Request for parameter sweep."""
    strategy_id: str
    start_date: str = Field(pattern=r"^\d{4}-\d{2}-\d{2}$")
    end_date: str = Field(pattern=r"^\d{4}-\d{2}-\d{2}$")
    parameters: list[SweepParameter]
    rank_by: str = Field(default="sharpe_ratio", pattern="^(sharpe_ratio|total_pnl|win_rate|max_drawdown)$")
    simulation: SimulationConfig = SimulationConfig()


class SweepResult(BaseModel):
    parameters: dict
    summary: BacktestSummary


class SweepResults(BaseModel):
    results: list[SweepResult]
    best: Optional[SweepResult] = None
    total_configs: int


# --- Replay Models (Story 5.5) ---

class ReplayRequest(BaseModel):
    strategy_id: str = Field(min_length=1, max_length=64)
    start_date: str = Field(pattern=r"^\d{4}-\d{2}-\d{2}$")
    end_date: str = Field(pattern=r"^\d{4}-\d{2}-\d{2}$")
    speed: int = Field(default=1, ge=1, le=10, description="Replay speed multiplier (1x, 2x, 5x, 10x)")


class ReplayEvent(BaseModel):
    event_type: str  # "market_update", "decision", "risk_event", "done"
    timestamp: str
    data: dict


class DecisionDisplay(BaseModel):
    timestamp: str
    market_id: str
    detected: str  # "YES+NO arbitrage", "Cross-market arbitrage"
    decision: str  # "EXECUTE", "SKIP", "FILTER"
    reason: str
    score: str
    risk_result: str  # "ALLOWED", "DENIED"
