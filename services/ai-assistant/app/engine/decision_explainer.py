import logging
import time
from decimal import Decimal

import asyncpg

from app.engine.llm_client import LLMClient
from app.repos import trade_repo

logger = logging.getLogger(__name__)


async def explain_trade(
    pool: asyncpg.Pool,
    llm: LLMClient,
    trade_id: str,
) -> dict:
    """Explain why a specific trade was made."""
    start_time = time.time()

    async with pool.acquire() as conn:
        context = await trade_repo.get_trade_context(conn, trade_id)

    trade = context.get("trade")
    if trade is None:
        return {"error": "Trade not found", "trade_id": trade_id}

    risk_events = context.get("risk_events", [])

    # Build decision context string
    decision_context = _build_decision_context(trade, risk_events)

    # Generate explanation
    explanation = await llm.explain_trade_decision(trade, decision_context)

    response_time_ms = int((time.time() - start_time) * 1000)

    return {
        "trade_id": trade_id,
        "market_id": trade.get("market_id"),
        "side": trade.get("side"),
        "entry_price": trade.get("entry_price"),
        "exit_price": trade.get("exit_price"),
        "pnl": trade.get("pnl"),
        "explanation": explanation,
        "decision_context": decision_context,
        "related_events": risk_events,
        "response_time_ms": response_time_ms,
    }


def _build_decision_context(trade: dict, risk_events: list[dict]) -> str:
    """Build a human-readable decision context string."""
    parts = []

    # Trade details
    parts.append(f"Strategy: {trade.get('strategy_id', 'unknown')}")
    parts.append(f"Market: {trade.get('market_id', 'unknown')}")
    parts.append(f"Side: {trade.get('side', 'unknown')}")
    parts.append(f"Entry: {trade.get('entry_price', 'N/A')}")
    parts.append(f"Exit: {trade.get('exit_price', 'N/A')}")

    # PnL
    pnl = trade.get("pnl")
    if pnl is not None:
        if pnl > 0:
            parts.append(f"Result: Profitable trade (+${pnl:.2f})")
        elif pnl < 0:
            parts.append(f"Result: Losing trade (${pnl:.2f})")
        else:
            parts.append("Result: Break-even trade")

    # Risk events
    if risk_events:
        parts.append("\nRisk events that influenced this trade:")
        for event in risk_events:
            event_type = event.get("event_type", "unknown")
            decision = event.get("decision", "unknown")
            reason = event.get("reason", "no reason provided")
            parts.append(f"- {event_type}: {decision} — {reason}")
    else:
        parts.append("\nNo risk events recorded for this trade.")

    return "\n".join(parts)
