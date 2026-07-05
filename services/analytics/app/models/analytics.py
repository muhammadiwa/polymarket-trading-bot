from datetime import datetime
from typing import Optional

from pydantic import BaseModel, Field


class PnLByPeriod(BaseModel):
    date: str
    pnl: str  # Decimal string
    trade_count: int


class PnLByStrategy(BaseModel):
    strategy_id: str
    total_pnl: str
    trade_count: int


class PnLByMarket(BaseModel):
    market_id: str
    market_slug: str
    total_pnl: str
    trade_count: int


class PnLResponse(BaseModel):
    by_period: list[PnLByPeriod]
    by_strategy: list[PnLByStrategy]
    by_market: list[PnLByMarket]
    total_pnl: str
    total_trades: int


class PerformanceMetrics(BaseModel):
    win_rate: str
    average_win: str
    average_loss: str
    profit_factor: Optional[str]  # None when no losing trades
    sharpe_ratio: str
    total_trades: int
    winning_trades: int
    losing_trades: int


class RiskMetrics(BaseModel):
    max_drawdown: str
    current_drawdown: str
    var_95: str
    peak_equity: str
    current_equity: str


class AnalyticsSummary(BaseModel):
    pnl: PnLResponse
    performance: PerformanceMetrics
    risk: RiskMetrics
    date_range: dict
