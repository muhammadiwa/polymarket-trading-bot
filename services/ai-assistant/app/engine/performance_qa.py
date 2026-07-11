import logging
import time
from decimal import Decimal

import asyncpg

from app.engine.llm_client import LLMClient
from app.repos import trade_repo

logger = logging.getLogger(__name__)

INTENT_HANDLERS = {
    "pnl": "_handle_pnl_question",
    "win_rate": "_handle_win_rate_question",
    "trade_count": "_handle_trade_count_question",
    "strategy": "_handle_strategy_question",
    "market": "_handle_market_question",
}


async def answer_question(
    pool: asyncpg.Pool,
    llm: LLMClient,
    question: str,
    user_id: str = "",
) -> dict:
    """Answer a performance question with verified data."""
    if not user_id or not user_id.strip():
        raise ValueError("user_id is required")

    start_time = time.time()

    # Detect intent
    intent = await llm.detect_intent(question)
    category = intent.get("category", "general")
    period = intent.get("period", "week")

    # Gather data based on intent
    data_points = []
    source_trade_ids = []

    async with pool.acquire() as conn:
        if category in ("pnl", "win_rate", "trade_count"):
            summary = await trade_repo.get_pnl_summary(conn, user_id, period)
            data_points = _format_pnl_summary(summary)
        elif category == "strategy":
            strategies = await trade_repo.get_pnl_by_strategy(conn, period)
            data_points = _format_strategy_data(strategies)
        elif category == "market":
            market_id = intent.get("market_id", "")
            if market_id:
                trades = await trade_repo.get_trades_by_market(conn, market_id)
                data_points = _format_market_data(trades)
                source_trade_ids = [t["id"] for t in trades]
            else:
                data_points = [{"label": "Note", "value": "No specific market identified in question"}]
        else:
            summary = await trade_repo.get_pnl_summary(conn, user_id, period)
            data_points = _format_pnl_summary(summary)

    # Generate answer
    answer = await llm.ask_performance_question(question, data_points)

    response_time_ms = int((time.time() - start_time) * 1000)

    return {
        "answer": answer,
        "data_points": data_points,
        "source_trade_ids": source_trade_ids,
        "response_time_ms": response_time_ms,
    }


def _format_pnl_summary(summary: dict) -> list[dict]:
    """Format PnL summary as data points."""
    return [
        {"label": "Total PnL", "value": f"${summary['total_pnl']:.2f}"},
        {"label": "Period", "value": summary["period"]},
        {"label": "Trade Count", "value": str(summary["trade_count"])},
        {"label": "Win Rate", "value": f"{summary['win_rate']:.1f}%"},
        {"label": "Winning Trades", "value": str(summary["winning_trades"])},
        {"label": "Losing Trades", "value": str(summary["losing_trades"])},
        {"label": "Average PnL per Trade", "value": f"${summary['avg_pnl']:.2f}"},
        {"label": "Best Trade", "value": f"${summary['best_trade']:.2f}"},
        {"label": "Worst Trade", "value": f"${summary['worst_trade']:.2f}"},
    ]


def _format_strategy_data(strategies: list[dict]) -> list[dict]:
    """Format strategy data as data points."""
    points = []
    for s in strategies:
        points.append({
            "label": f"Strategy {s['strategy_id']}",
            "value": f"PnL: ${s['total_pnl']:.2f}, Trades: {s['trade_count']}, Win Rate: {s['win_rate']:.1f}%",
        })
    return points if points else [{"label": "Note", "value": "No strategy data available"}]


def _format_market_data(trades: list[dict]) -> list[dict]:
    """Format market trade data as data points."""
    if not trades:
        return [{"label": "Note", "value": "No trades found for this market"}]

    total_pnl = sum(t["pnl"] for t in trades if t["pnl"] is not None)
    trade_count = len(trades)

    return [
        {"label": "Market Trade Count", "value": str(trade_count)},
        {"label": "Market Total PnL", "value": f"${total_pnl:.2f}"},
    ]
