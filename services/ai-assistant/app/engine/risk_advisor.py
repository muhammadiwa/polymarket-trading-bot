import logging
from decimal import Decimal, ROUND_HALF_UP
from typing import Optional

from app.repos import trade_repo

logger = logging.getLogger(__name__)

ZERO = Decimal("0")


async def suggest_risk_parameters(
    conn,
    risk_state: dict,
    strategy_id: Optional[str] = None,
) -> list[dict]:
    """Generate conservative risk parameter suggestions based on current state."""
    suggestions = []

    # Get recent trade performance
    trades = await trade_repo.get_trades(conn, limit=500)
    if not trades:
        return suggestions

    wins = [t for t in trades if t.get("pnl") and Decimal(str(t["pnl"])) > ZERO]
    losses = [t for t in trades if t.get("pnl") and Decimal(str(t["pnl"])) < ZERO]
    total = len(trades)
    win_rate = Decimal(len(wins)) / Decimal(total) if total > 0 else ZERO

    # Current risk parameters from state
    current_drawdown = Decimal(str(risk_state.get("current_drawdown", "0")))
    daily_budget_remaining = Decimal(str(risk_state.get("daily_budget_remaining", "0")))
    circuit_breaker = risk_state.get("circuit_breaker_status", "open")

    # Rule 1: Daily loss limit — decrease if drawdown > 5%
    if current_drawdown > Decimal("0.05"):
        current_limit = Decimal(str(risk_state.get("daily_loss_limit_pct", "0.02")))
        suggested = max(Decimal("0.01"), current_limit - Decimal("0.005"))
        suggestions.append({
            "parameter": "daily_loss_limit_pct",
            "current_value": str(current_limit),
            "suggested_value": str(suggested.quantize(Decimal("0.001"), rounding=ROUND_HALF_UP)),
            "direction": "decrease",
            "rationale": f"Current drawdown is {current_drawdown * 100:.1f}%, approaching the circuit breaker threshold. Reducing daily loss limit to {suggested * 100:.1f}% limits exposure during volatile periods.",
            "confidence": "high",
            "data_points": [f"current_drawdown: {current_drawdown * 100:.1f}%", f"current_limit: {current_limit * 100:.1f}%"],
        })

    # Rule 2: Score threshold — increase if win rate < 60%
    if win_rate < Decimal("0.60") and total >= 20:
        current_threshold = Decimal(str(risk_state.get("score_threshold", "0.01")))
        suggested = current_threshold + Decimal("0.005")
        suggestions.append({
            "parameter": "score_threshold",
            "current_value": str(current_threshold),
            "suggested_value": str(suggested.quantize(Decimal("0.001"), rounding=ROUND_HALF_UP)),
            "direction": "increase",
            "rationale": f"Win rate is at {win_rate * 100:.1f}% over {total} trades. Raising the score threshold filters out lower-quality opportunities and improves overall win rate.",
            "confidence": "medium",
            "data_points": [f"win_rate: {win_rate * 100:.1f}%", f"total_trades: {total}", f"current_threshold: {current_threshold}"],
        })

    # Rule 3: Max position per market — decrease if any market near limit
    market_limits = risk_state.get("market_limits", {})
    for market_id, limit_info in market_limits.items():
        if isinstance(limit_info, dict):
            utilization = Decimal(str(limit_info.get("utilization", "0")))
            if utilization > Decimal("0.8"):
                current_limit_pct = Decimal(str(risk_state.get("market_limit_pct", "0.10")))
                suggested = max(Decimal("0.05"), current_limit_pct - Decimal("0.02"))
                suggestions.append({
                    "parameter": "market_limit_pct",
                    "current_value": str(current_limit_pct),
                    "suggested_value": str(suggested.quantize(Decimal("0.01"), rounding=ROUND_HALF_UP)),
                    "direction": "decrease",
                    "rationale": f"Market {market_id} is at {utilization * 100:.0f}% utilization. Reducing per-market limit to {suggested * 100:.0f}% provides more headroom.",
                    "confidence": "medium",
                    "data_points": [f"market: {market_id}", f"utilization: {utilization * 100:.0f}%"],
                })
            break  # Only suggest for the most utilized market

    # Rule 4: Slippage tolerance — decrease if avg slippage is high
    avg_slippage = ZERO
    slippage_trades = [t for t in trades if t.get("slippage_pct")]
    if slippage_trades:
        avg_slippage = sum(Decimal(str(t["slippage_pct"])) for t in slippage_trades) / len(slippage_trades)
    if avg_slippage > Decimal("0.005"):  # > 0.5%
        current_tolerance = Decimal(str(risk_state.get("slippage_tolerance", "0.01")))
        suggested = max(Decimal("0.005"), current_tolerance - Decimal("0.002"))
        suggestions.append({
            "parameter": "slippage_tolerance",
            "current_value": str(current_tolerance),
            "suggested_value": str(suggested.quantize(Decimal("0.001"), rounding=ROUND_HALF_UP)),
            "direction": "decrease",
            "rationale": f"Average slippage is {avg_slippage * 100:.2f}%. Tighter tolerance reduces execution cost.",
            "confidence": "medium",
            "data_points": [f"avg_slippage: {avg_slippage * 100:.2f}%", f"current_tolerance: {current_tolerance * 100:.1f}%"],
        })

    return suggestions
