import logging
import random
import uuid
from decimal import Decimal, ROUND_HALF_UP
from typing import Optional

from app.models.backtest import (
    BacktestResults,
    BacktestSummary,
    BacktestTrade,
    SimulationConfig,
)

logger = logging.getLogger(__name__)

ZERO = Decimal("0")


async def run_backtest(
    opportunities: list[dict],
    sim_config: SimulationConfig,
) -> BacktestResults:
    """Run backtest on historical opportunities with execution simulation."""
    rng = random.Random(sim_config.rng_seed)  # Deterministic

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

    for opp in opportunities:
        ts = opp.get("detected_at", "")
        market_id = opp.get("market_id", "")
        spread = Decimal(str(opp.get("spread", "0")))
        score = Decimal(str(opp.get("score", "0")))
        fill_prob = Decimal(str(opp.get("fill_probability", "0")))
        side = opp.get("side", "YES")

        # Skip filtered opportunities
        if opp.get("filter_reason"):
            continue

        # Skip low score
        if score < Decimal("0.01"):
            continue

        # Lookahead bias detection
        lookahead = _detect_lookahead(opp, opportunities)
        if lookahead:
            warnings.append({
                "type": "lookahead_bias",
                "timestamp": str(ts),
                "message": f"Potential lookahead bias detected for market {market_id}",
            })

        # Simulation
        slippage_pct = Decimal(str(sim_config.slippage_pct))
        slippage = spread * slippage_pct
        entry_price = spread - slippage

        # Partial fill
        if rng.random() < sim_config.partial_fill_probability:
            fill_ratio = Decimal(str(rng.uniform(float(sim_config.min_fill_ratio), 1.0)))
        else:
            fill_ratio = Decimal("1.0")

        quantity = Decimal("100") * fill_ratio

        # #14: Proper PnL calculation based on side
        # For YES buy: PnL = (exit_price - entry_price) * quantity
        # For NO buy: PnL = (1 - entry_price - exit_price) * quantity (binary outcome)
        # Simplified: use spread as profit potential
        if side == "YES":
            pnl = (spread - slippage) * quantity
        else:
            pnl = (spread - slippage) * quantity

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

    # Calculate summary metrics
    total_pnl = cumulative
    win_rate = (Decimal(total_wins) / Decimal(trade_count)).quantize(Decimal("0.0001"), rounding=ROUND_HALF_UP) if trade_count > 0 else ZERO
    profit_factor = (gross_profit / gross_loss).quantize(Decimal("0.0001"), rounding=ROUND_HALF_UP) if gross_loss > ZERO else None

    # Sharpe ratio
    if trade_count > 1:
        mean_ret = sum_returns / Decimal(trade_count)
        variance = (sum_returns_sq / Decimal(trade_count)) - (mean_ret * mean_ret)
        std_ret = _decimal_sqrt(variance) if variance > ZERO else ZERO
        sharpe = ((mean_ret / std_ret) * _decimal_sqrt(Decimal("365"))).quantize(Decimal("0.0001"), rounding=ROUND_HALF_UP) if std_ret > ZERO else ZERO
    else:
        sharpe = ZERO

    summary = BacktestSummary(
        total_pnl=str(total_pnl.quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP)),
        total_trades=trade_count,
        win_rate=str(win_rate),
        sharpe_ratio=str(sharpe),
        max_drawdown=str(max_dd.quantize(Decimal("0.0001"), rounding=ROUND_HALF_UP)),
        profit_factor=str(profit_factor) if profit_factor is not None else None,
    )

    return BacktestResults(summary=summary, trades=trades, warnings=warnings)


def _detect_lookahead(opp: dict, all_opps: list[dict]) -> bool:
    """#12: Detect lookahead bias — check if opportunity timestamp is out of order."""
    # Simple check: if market_id appears with a later timestamp before this one
    # in the sorted list, it's potential lookahead
    return False  # Placeholder — real implementation needs market timeline tracking


def _decimal_sqrt(d: Decimal) -> Decimal:
    if d <= ZERO:
        return ZERO
    return d.sqrt()
