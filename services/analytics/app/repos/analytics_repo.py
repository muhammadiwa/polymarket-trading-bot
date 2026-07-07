import math
import json
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
    side: Optional[str] = None,
    pnl_sign: Optional[str] = None,
) -> list[dict]:
    """Fetch filled trades in date range with optional filters."""
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
    if side:
        conditions.append(f"side = ${idx}")
        params.append(side)
        idx += 1
    # pnl_sign: SQL filter for efficiency
    if pnl_sign == "positive":
        conditions.append("pnl > 0")
    elif pnl_sign == "negative":
        conditions.append("pnl < 0")
    elif pnl_sign == "zero":
        conditions.append("pnl = 0")

    where = " AND ".join(conditions)
    rows = await conn.fetch(
        f"SELECT pnl, strategy_id, market_id, market_slug, fill_timestamp, side, quantity, price, filled_quantity, fee, slippage_pct, fill_status, latency_ms FROM trades WHERE {where} ORDER BY fill_timestamp",
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
        variance = sum((p - mean_return) ** 2 for p in pnls) / Decimal(len(pnls) - 1)
        std_return = _decimal_sqrt(variance)  # #2: Use Decimal sqrt, not float
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


# --- Anomaly Detection (Story 4.3) ---

async def detect_anomalies(
    conn: asyncpg.Connection,
    thresholds: dict,
) -> list[dict]:
    """Detect performance anomalies by comparing current metrics against 7-day baseline."""
    from datetime import timedelta
    now = datetime.now(timezone.utc)
    day_ago = now - timedelta(days=1)
    week_ago = now - timedelta(days=7)

    anomalies = []

    # Get current (last 24h) and baseline (7d to 1d ago, NOT including current)
    current_perf = await calculate_performance_metrics(conn, day_ago, now)
    baseline_perf = await calculate_performance_metrics(conn, week_ago, day_ago)
    current_risk = await calculate_risk_metrics(conn, day_ago, now)
    baseline_risk = await calculate_risk_metrics(conn, week_ago, day_ago)

    # Rule 1: Win rate drop
    current_wr = Decimal(current_perf["win_rate"]) if current_perf["win_rate"] else ZERO
    baseline_wr = Decimal(baseline_perf["win_rate"]) if baseline_perf["win_rate"] else ZERO
    wr_drop = baseline_wr - current_wr
    if baseline_wr > Decimal("0.1") and wr_drop > Decimal(str(thresholds.get("win_rate_drop", 0.20))):
        anomalies.append({
            "anomaly_type": "win_rate_drop",
            "metric_name": "win_rate",
            "threshold_value": str(baseline_wr - Decimal(str(thresholds.get("win_rate_drop", 0.20)))),
            "actual_value": str(current_wr),
            "severity": "high",
            "confidence": "0.9",
            "context": {"baseline_7d": str(baseline_wr), "current_1d": str(current_wr), "drop": str(wr_drop)},
        })

    # Rule 2: Unusual drawdown
    current_dd = Decimal(current_risk["current_drawdown"]) if current_risk["current_drawdown"] else ZERO
    baseline_dd = Decimal(baseline_risk["max_drawdown"]) if baseline_risk["max_drawdown"] else ZERO
    dd_multiplier = Decimal(str(thresholds.get("drawdown_multiplier", 2.0)))
    if baseline_dd > Decimal("0.01") and current_dd > baseline_dd * dd_multiplier:
        anomalies.append({
            "anomaly_type": "unusual_drawdown",
            "metric_name": "current_drawdown",
            "threshold_value": str(baseline_dd * dd_multiplier),
            "actual_value": str(current_dd),
            "severity": "critical",
            "confidence": "0.85",
            "context": {"baseline_max_dd": str(baseline_dd), "current_dd": str(current_dd)},
        })

    # Rule 3: Consecutive losses
    recent_trades = await get_trades_in_range(conn, day_ago, now)
    max_consecutive = thresholds.get("consecutive_losses", 5)
    consecutive = 0
    max_seen = 0
    for t in recent_trades:
        # #4: Guard against NULL pnl
        pnl_val = t.get("pnl")
        if pnl_val is None:
            continue
        if Decimal(str(pnl_val)) < ZERO:
            consecutive += 1
            if consecutive > max_seen:
                max_seen = consecutive
        else:
            consecutive = 0
    if max_seen >= max_consecutive:
        anomalies.append({
            "anomaly_type": "consecutive_losses",
            "metric_name": "consecutive_loss_streak",
            "threshold_value": str(max_consecutive),
            "actual_value": str(max_seen),
            "severity": "medium",
            "confidence": "1.0",
            "context": {"streak_length": max_seen},
        })

    # Rule 4: Profit factor drop
    current_pf = Decimal(current_perf["profit_factor"]) if current_perf.get("profit_factor") else None
    baseline_pf = Decimal(baseline_perf["profit_factor"]) if baseline_perf.get("profit_factor") else None
    pf_low = Decimal(str(thresholds.get("profit_factor_low", 0.5)))
    if current_pf is not None and baseline_pf is not None and baseline_pf > Decimal("1.5") and current_pf < pf_low:
        anomalies.append({
            "anomaly_type": "profit_factor_drop",
            "metric_name": "profit_factor",
            "threshold_value": str(pf_low),
            "actual_value": str(current_pf),
            "severity": "high",
            "confidence": "0.85",
            "context": {"baseline_7d": str(baseline_pf), "current_1d": str(current_pf)},
        })

    # Rule 5: Sharpe ratio drop
    current_sharpe = Decimal(current_perf["sharpe_ratio"]) if current_perf["sharpe_ratio"] else ZERO
    baseline_sharpe = Decimal(baseline_perf["sharpe_ratio"]) if baseline_perf["sharpe_ratio"] else ZERO
    sharpe_low = Decimal(str(thresholds.get("sharpe_low", 0)))
    if baseline_sharpe > Decimal("1.0") and current_sharpe < sharpe_low:
        anomalies.append({
            "anomaly_type": "sharpe_drop",
            "metric_name": "sharpe_ratio",
            "threshold_value": str(sharpe_low),
            "actual_value": str(current_sharpe),
            "severity": "medium",
            "confidence": "0.8",
            "context": {"baseline_7d": str(baseline_sharpe), "current_1d": str(current_sharpe)},
        })

    return anomalies


async def log_anomaly(conn: asyncpg.Connection, anomaly: dict) -> Optional[str]:
    """Log anomaly to PostgreSQL. Returns ID if new, None if duplicate."""
    # #5: Suppress duplicates — check if same anomaly_type was logged in last 24h
    from datetime import timedelta
    day_ago = datetime.now(timezone.utc) - timedelta(hours=24)
    existing = await conn.fetchrow(
        "SELECT id FROM anomaly_events WHERE anomaly_type = $1 AND detected_at > $2 LIMIT 1",
        anomaly["anomaly_type"], day_ago,
    )
    if existing:
        return None  # Already logged recently — skip

    row = await conn.fetchrow(
        """
        INSERT INTO anomaly_events (anomaly_type, metric_name, threshold_value, actual_value, severity, confidence, context)
        VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
        RETURNING id
        """,
        anomaly["anomaly_type"],
        anomaly["metric_name"],
        Decimal(anomaly["threshold_value"]),
        Decimal(anomaly["actual_value"]),
        anomaly["severity"],
        Decimal(anomaly.get("confidence", "0.9")),
        json.dumps(anomaly.get("context", {})),
    )
    return str(row["id"])


async def get_anomalies(
    conn: asyncpg.Connection,
    limit: int = 50,
    severity: Optional[str] = None,
) -> list[dict]:
    """Get recent anomalies."""
    conditions = []
    params = []
    idx = 1
    if severity:
        conditions.append(f"severity = ${idx}")
        params.append(severity)
        idx += 1

    where = f"WHERE {' AND '.join(conditions)}" if conditions else ""
    rows = await conn.fetch(
        f"SELECT * FROM anomaly_events {where} ORDER BY detected_at DESC LIMIT ${idx}",
        *params, limit,
    )
    return [dict(r) for r in rows]
