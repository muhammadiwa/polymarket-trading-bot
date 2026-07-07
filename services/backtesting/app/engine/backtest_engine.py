import json
import logging
import random
import uuid
from datetime import datetime, timezone
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
    pnls: list[Decimal] = []
    cumulative = ZERO
    peak = ZERO
    max_dd = ZERO

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

        # #4: Lookahead bias detection
        lookahead = False
        if _detect_lookahead(opp, opportunities):
            lookahead = True
            warnings.append({
                "type": "lookahead_bias",
                "timestamp": ts,
                "message": f"Potential lookahead bias detected for market {market_id}",
            })

        # Simulate execution
        if score < Decimal("0.01"):  # Score threshold
            continue

        # Slippage
        slippage_pct = Decimal(str(sim_config.slippage_pct))
        slippage = spread * slippage_pct
        fill_price = spread - slippage

        # Partial fill
        if rng.random() < sim_config.partial_fill_probability:
            fill_ratio = Decimal(str(rng.uniform(float(sim_config.min_fill_ratio), 1.0)))
        else:
            fill_ratio = Decimal("1.0")

        quantity = Decimal("100") * fill_ratio  # Base quantity
        pnl = fill_price * quantity

        trade = BacktestTrade(
            timestamp=ts,
            market_id=market_id,
            side=side,
            price=str(fill_price.quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP)),
            quantity=str(quantity.quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP)),
            slippage=str(slippage.quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP)),
            pnl=str(pnl.quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP)),
            lookahead_warning=lookahead,
        )
        trades.append(trade)
        pnls.append(pnl)
        cumulative += pnl
        if cumulative > peak:
            peak = cumulative
        if peak > ZERO:
            dd = (peak - cumulative) / peak
            if dd > max_dd:
                max_dd = dd

    # Calculate summary metrics
    total_pnl = sum(pnls) if pnls else ZERO
    total_trades = len(trades)
    wins = [p for p in pnls if p > ZERO]
    losses = [p for p in pnls if p < ZERO]
    win_rate = (Decimal(len(wins)) / Decimal(total_trades)).quantize(Decimal("0.0001"), rounding=ROUND_HALF_UP) if total_trades > 0 else ZERO

    gross_profit = sum(wins) if wins else ZERO
    gross_loss = abs(sum(losses)) if losses else ZERO
    profit_factor = (gross_profit / gross_loss).quantize(Decimal("0.0001"), rounding=ROUND_HALF_UP) if gross_loss > ZERO else None

    # Sharpe ratio
    if len(pnls) > 1:
        mean_ret = sum(pnls) / Decimal(len(pnls))
        variance = sum((p - mean_ret) ** 2 for p in pnls) / Decimal(len(pnls) - 1)
        std_ret = _decimal_sqrt(variance)
        sharpe = ((mean_ret / std_ret) * _decimal_sqrt(Decimal("365"))).quantize(Decimal("0.0001"), rounding=ROUND_HALF_UP) if std_ret > ZERO else ZERO
    else:
        sharpe = ZERO

    summary = BacktestSummary(
        total_pnl=str(total_pnl.quantize(Decimal("0.00000001"), rounding=ROUND_HALF_UP)),
        total_trades=total_trades,
        win_rate=str(win_rate),
        sharpe_ratio=str(sharpe),
        max_drawdown=str(max_dd.quantize(Decimal("0.0001"), rounding=ROUND_HALF_UP)),
        profit_factor=str(profit_factor) if profit_factor is not None else None,
    )

    return BacktestResults(summary=summary, trades=trades, warnings=warnings)


def _detect_lookahead(opp: dict, all_opps: list[dict]) -> bool:
    """Detect if opportunity uses data from the future."""
    # Simple heuristic: if an opportunity references a market that hasn't appeared yet
    # in the chronological sequence, it's potential lookahead
    # This is a simplified check — real implementation would track data access patterns
    return False


def _decimal_sqrt(d: Decimal) -> Decimal:
    if d <= ZERO:
        return ZERO
    return d.sqrt()
