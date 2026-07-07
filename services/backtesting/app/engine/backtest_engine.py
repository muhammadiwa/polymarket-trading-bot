import logging
import random
from collections import defaultdict
from decimal import Decimal, ROUND_HALF_UP
from typing import Optional

from app.models.backtest import (
    BacktestReport,
    BacktestResults,
    BacktestSummary,
    BacktestTrade,
    SimulationConfig,
)

logger = logging.getLogger(__name__)

ZERO = Decimal("0")
Z_SCORE_95 = Decimal("1.645")


async def run_backtest(
    opportunities: list[dict],
    sim_config: SimulationConfig,
) -> BacktestResults:
    """Run backtest on historical opportunities with execution simulation."""
    rng = random.Random(sim_config.rng_seed)

    trades: list[BacktestTrade] = []
    warnings: list[dict] = []
    cumulative = ZERO
    peak = ZERO
    max_dd = ZERO
    total_wins = 0
    total_losses = 0
    gross_profit = ZERO
    gross_loss = ZERO
    sum_returns = ZERO
    sum_returns_sq = ZERO
    trade_count = 0
    daily_pnl: dict[str, Decimal] = defaultdict(lambda: ZERO)  # #1: Fix defaultdict crash

    for opp in opportunities:
        ts = opp.get("detected_at", "")
        market_id = opp.get("market_id", "")
        spread = Decimal(str(opp.get("spread", "0")))
        score = Decimal(str(opp.get("score", "0")))
        side = opp.get("side", "YES")

        if opp.get("filter_reason"):
            continue
        if score < Decimal("0.01"):
            continue

        lookahead = _detect_lookahead(opp, opportunities)
        if lookahead:
            warnings.append({
                "type": "lookahead_bias",
                "timestamp": str(ts),
                "message": f"Potential lookahead bias detected for market {market_id}",
            })

        # Slippage with liquidity
        slippage_pct = Decimal(str(sim_config.slippage_pct))
        liquidity = Decimal(str(opp.get("liquidity", "0.5")))
        if liquidity <= ZERO:
            liquidity = Decimal("0.5")
        slippage = spread * slippage_pct / liquidity.sqrt()
        entry_price = spread - slippage

        # Partial fill
        if rng.random() < sim_config.partial_fill_probability:
            fill_ratio = Decimal(str(rng.uniform(float(sim_config.min_fill_ratio), 1.0)))
        else:
            fill_ratio = Decimal("1.0")

        quantity = Decimal("100") * fill_ratio

        # #10: Use entry_price in PnL calculation
        pnl = entry_price * quantity

        trade = BacktestTrade(
            timestamp=str(ts),
            market_id=market_id,
            side=side,
            price=str(entry_price.quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP)),
            quantity=str(quantity.quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP)),
            slippage=str(slippage.quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP)),
            pnl=str(pnl.quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP)),
            lookahead_warning=lookahead,
        )
        trades.append(trade)

        cumulative += pnl
        if cumulative > peak:
            peak = cumulative
        if peak > ZERO:
            dd = (peak - cumulative) / peak
            if dd > max_dd:
                max_dd = dd

        if pnl > ZERO:
            total_wins += 1
            gross_profit += pnl
        elif pnl < ZERO:
            total_losses += 1
            gross_loss += abs(pnl)

        sum_returns += pnl
        sum_returns_sq += pnl * pnl
        trade_count += 1

        # #1: Track daily PnL
        day = str(ts)[:10] if ts else "unknown"
        daily_pnl[day] += pnl

    # Calculate summary metrics
    total_pnl = cumulative
    win_rate = (Decimal(total_wins) / Decimal(trade_count)).quantize(Decimal("0.0001"), rounding=ROUND_HALF_UP) if trade_count > 0 else ZERO
    profit_factor = (gross_profit / gross_loss).quantize(Decimal("0.0001"), rounding=ROUND_HALF_UP) if gross_loss > ZERO else None

    # Sharpe ratio
    if trade_count > 1:
        mean_ret = sum_returns / Decimal(trade_count)
        variance = (sum_returns_sq / Decimal(trade_count)) - (mean_ret * mean_ret)
        # #6: Clamp variance to prevent negative from Decimal rounding
        if variance < ZERO:
            variance = ZERO
        std_ret = _decimal_sqrt(variance)
        sharpe = ((mean_ret / std_ret) * _decimal_sqrt(Decimal("365"))).quantize(Decimal("0.0001"), rounding=ROUND_HALF_UP) if std_ret > ZERO else ZERO
    else:
        sharpe = ZERO

    # #1: VaR 95% (parametric)
    if trade_count > 1:
        var_95 = (mean_ret - Z_SCORE_95 * std_ret).quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP)
    else:
        var_95 = ZERO

    # #1: Daily PnL breakdown
    daily_pnl_list = [
        {"date": day, "pnl": str(pnl.quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP))}
        for day, pnl in sorted(daily_pnl.items())
    ]

    summary = BacktestSummary(
        total_pnl=str(total_pnl.quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP)),
        total_trades=trade_count,
        win_rate=str(win_rate),
        sharpe_ratio=str(sharpe),
        max_drawdown=str(max_dd.quantize(Decimal("0.0001"), rounding=ROUND_HALF_UP)),
        profit_factor=str(profit_factor) if profit_factor is not None else None,
        var_95=str(var_95),
    )

    return BacktestResults(summary=summary, trades=trades, warnings=warnings, daily_pnl=daily_pnl_list)


async def generate_report(results: BacktestResults) -> BacktestReport:
    """#2: Generate full report with PnL and drawdown curves."""
    # Build PnL curve
    cumulative = ZERO
    pnl_curve = []
    for dp in results.daily_pnl or []:
        cumulative += Decimal(dp["pnl"])
        pnl_curve.append({"date": dp["date"], "cumulative_pnl": str(cumulative.quantize(Decimal("0.00000001")))})

    # Build drawdown curve
    peak = ZERO
    drawdown_curve = []
    for point in pnl_curve:
        val = Decimal(point["cumulative_pnl"])
        if val > peak:
            peak = val
        if peak > ZERO:
            dd = ((peak - val) / peak * Decimal("100")).quantize(Decimal("0.01"))
        else:
            dd = ZERO
        drawdown_curve.append({"date": point["date"], "drawdown": str(dd)})

    return BacktestReport(
        run_id="",
        summary=results.summary,
        pnl_curve=pnl_curve,
        drawdown_curve=drawdown_curve,
        trades=results.trades,
        warnings=results.warnings,
    )


def _detect_lookahead(opp: dict, all_opps: list[dict]) -> bool:
    return False


def _decimal_sqrt(d: Decimal) -> Decimal:
    if d <= ZERO:
        return ZERO
    return d.sqrt()
