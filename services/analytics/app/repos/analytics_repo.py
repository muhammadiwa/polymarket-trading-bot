import math
import logging
from datetime import datetime, timezone
from decimal import Decimal, ROUND_HALF_UP
from typing import Optional

import asyncpg

from app.config import config

logger = logging.getLogger(__name__)

ZERO = Decimal("0")
HUNDRED = Decimal("100")
Z_SCORE_95 = Decimal("1.645")


def _decimal_sqrt(d: Decimal) -> Decimal:
    """#3: Compute square root using Decimal to avoid float precision loss."""
    if d <= ZERO:
        return ZERO
    # Use Python's built-in Decimal sqrt (available in Python 3.9+)
    return d.sqrt()


async def get_trades_in_range(
    conn: asyncpg.Connection,
    start_date: datetime,
    end_date: datetime,
    strategy_id: Optional[str] = None,
    market_id: Optional[str] = None,
) -> list[dict]:
    """Fetch filled trades in date range."""
    conditions = ["fill_timestamp BETWEEN $1 AND $2", "fill_status IN ('FILLED', 'PARTIAL_FILL')"]
    params: list = [start_date, end_date]
    idx = 3

    if strategy_id:
        conditions.append(f"strategy_id = ${idx}")
        params.append(strategy_id)
        idx += 1
    if market_id:
        conditions.append(f"market_id = ${idx}")
        params.append(market_id)
        idx += 1

    where = " AND ".join(conditions)
    rows = await conn.fetch(
        f"SELECT pnl, strategy_id, market_id, market_slug, fill_timestamp, side, quantity, price FROM trades WHERE {where} ORDER BY fill_timestamp",
        *params,
    )
    return [dict(r) for r in rows]


async def calculate_pnl(
    conn: asyncpg.Connection,
    start_date: datetime,
    end_date: datetime,
    group_by: str = "day",
    strategy_id: Optional[str] = None,
    market_id: Optional[str] = None,
) -> dict:
    """Calculate PnL aggregated by period, strategy, market."""
    trades = await get_trades_in_range(conn, start_date, end_date, strategy_id, market_id)

    if not trades:
        return {
            "by_period": [],
            "by_strategy": [],
            "by_market": [],
            "total_pnl": "0",
            "total_trades": 0,
        }

    # PnL by period
    by_period: dict[str, Decimal] = {}
    period_counts: dict[str, int] = {}
    for t in trades:
        ft = t["fill_timestamp"]
        if group_by == "day":
            key = ft.strftime("%Y-%m-%d")
        elif group_by == "week":
            # #8: Use ISO 8601 week format
            key = ft.strftime("%G-W%V")
        else:  # month
            key = ft.strftime("%Y-%m")
        pnl = Decimal(str(t["pnl"]))
        by_period[key] = by_period.get(key, ZERO) + pnl
        period_counts[key] = period_counts.get(key, 0) + 1

    period_list = [
        {"date": k, "pnl": str(by_period[k].quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP)), "trade_count": period_counts[k]}
        for k in sorted(by_period.keys())
    ]

    # PnL by strategy
    by_strategy: dict[str, Decimal] = {}
    strategy_counts: dict[str, int] = {}
    for t in trades:
        sid = t["strategy_id"]
        pnl = Decimal(str(t["pnl"]))
        by_strategy[sid] = by_strategy.get(sid, ZERO) + pnl
        strategy_counts[sid] = strategy_counts.get(sid, 0) + 1

    strategy_list = [
        {"strategy_id": k, "total_pnl": str(by_strategy[k].quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP)), "trade_count": strategy_counts[k]}
        for k in sorted(by_strategy.keys())
    ]

    # PnL by market
    by_market: dict[str, Decimal] = {}
    market_counts: dict[str, int] = {}
    market_slugs: dict[str, str] = {}
    for t in trades:
        mid = t["market_id"]
        pnl = Decimal(str(t["pnl"]))
        by_market[mid] = by_market.get(mid, ZERO) + pnl
        market_counts[mid] = market_counts.get(mid, 0) + 1
        market_slugs[mid] = t.get("market_slug", mid)

    market_list = [
        {"market_id": k, "market_slug": market_slugs.get(k, k), "total_pnl": str(by_market[k].quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP)), "trade_count": market_counts[k]}
        for k in sorted(by_market.keys())
    ]

    total_pnl = sum(Decimal(str(t["pnl"])) for t in trades)

    return {
        "by_period": period_list,
        "by_strategy": strategy_list,
        "by_market": market_list,
        "total_pnl": str(total_pnl.quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP)),
        "total_trades": len(trades),
    }


async def calculate_performance_metrics(
    conn: asyncpg.Connection,
    start_date: datetime,
    end_date: datetime,
    strategy_id: Optional[str] = None,
    market_id: Optional[str] = None,
) -> dict:
    """Calculate performance metrics: win rate, avg win/loss, profit factor, Sharpe."""
    trades = await get_trades_in_range(conn, start_date, end_date, strategy_id, market_id)

    if not trades:
        return {
            "win_rate": "0", "average_win": "0", "average_loss": "0",
            "profit_factor": "0", "sharpe_ratio": "0",
            "total_trades": 0, "winning_trades": 0, "losing_trades": 0,
        }

    pnls = [Decimal(str(t["pnl"])) for t in trades]
    wins = [p for p in pnls if p > ZERO]
    losses = [p for p in pnls if p < ZERO]

    win_rate = (Decimal(len(wins)) / Decimal(len(pnls))).quantize(Decimal("0.0001"), rounding=ROUND_HALF_UP) if pnls else ZERO
    avg_win = (sum(wins) / Decimal(len(wins))).quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP) if wins else ZERO
    avg_loss = (sum(losses) / Decimal(len(losses))).quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP) if losses else ZERO

    gross_profit = sum(wins) if wins else ZERO
    gross_loss = abs(sum(losses)) if losses else ZERO
    # #5: Return None when profit factor is undefined (no losses)
    profit_factor = (gross_profit / gross_loss).quantize(Decimal("0.0001"), rounding=ROUND_HALF_UP) if gross_loss > ZERO else None

    # Sharpe ratio (annualized) — uses sample variance (N-1)
    risk_free = Decimal(str(config.SHARPE_RISK_FREE_RATE))
    if len(pnls) > 1:
        mean_return = sum(pnls) / Decimal(len(pnls))
        # #2: Use sample variance (N-1) for Bessel's correction
        variance = sum((p - mean_return) ** 2 for p in pnls) / Decimal(len(pnls) - 1)
        # #3: Keep in Decimal — use Decimal sqrt
        std_return = _decimal_sqrt(variance)
        sharpe = ((mean_return - risk_free) / std_return * _decimal_sqrt(Decimal("365"))).quantize(Decimal("0.0001"), rounding=ROUND_HALF_UP) if std_return > ZERO else ZERO
    else:
        sharpe = ZERO

    return {
        "win_rate": str(win_rate),
        "average_win": str(avg_win),
        "average_loss": str(avg_loss),
        "profit_factor": str(profit_factor) if profit_factor is not None else None,
        "sharpe_ratio": str(sharpe),
        "total_trades": len(pnls),
        "winning_trades": len(wins),
        "losing_trades": len(losses),
    }


async def calculate_risk_metrics(
    conn: asyncpg.Connection,
    start_date: datetime,
    end_date: datetime,
    strategy_id: Optional[str] = None,
    market_id: Optional[str] = None,
) -> dict:
    """Calculate risk metrics: max drawdown, current drawdown, VaR."""
    trades = await get_trades_in_range(conn, start_date, end_date, strategy_id, market_id)

    if not trades:
        return {
            "max_drawdown": "0", "current_drawdown": "0", "var_95": "0",
            "peak_equity": "0", "current_equity": "0",
        }

    pnls = [Decimal(str(t["pnl"])) for t in trades]

    # Max drawdown from peak equity
    # #1: Use initial equity of 1.0 as baseline for all-losing portfolios
    initial_equity = Decimal("1.0")
    peak = initial_equity
    cumulative = initial_equity
    max_dd = ZERO
    for pnl in pnls:
        cumulative += pnl
        if cumulative > peak:
            peak = cumulative
        if peak > ZERO:
            dd = (peak - cumulative) / peak
            if dd > max_dd:
                max_dd = dd

    current_drawdown = ((peak - cumulative) / peak).quantize(Decimal("0.0001"), rounding=ROUND_HALF_UP) if peak > ZERO else ZERO

    # VaR 95% (parametric) — uses sample variance (N-1)
    if len(pnls) > 1:
        mean_return = sum(pnls) / Decimal(len(pnls))
        # #2: Use sample variance (N-1) for Sharpe and VaR
        variance = sum((p - mean_return) ** 2 for p in pnls) / Decimal(len(pnls) - 1)
        std_return = Decimal(str(math.sqrt(float(variance))))
        var_95 = (mean_return - Z_SCORE_95 * std_return).quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP)
    else:
        var_95 = ZERO

    return {
        "max_drawdown": str(max_dd.quantize(Decimal("0.0001"), rounding=ROUND_HALF_UP)),
        "current_drawdown": str(current_drawdown),
        "var_95": str(var_95),
        "peak_equity": str(peak.quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP)),
        "current_equity": str(cumulative.quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP)),
    }
