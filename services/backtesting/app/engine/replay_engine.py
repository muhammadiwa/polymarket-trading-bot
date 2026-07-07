import asyncio
import json
import logging
from datetime import datetime, timezone
from decimal import Decimal, ROUND_HALF_UP
from typing import AsyncGenerator

from app.models.backtest import DecisionDisplay, ReplayEvent

logger = logging.getLogger(__name__)

ZERO = Decimal("0")


async def replay_events(
    opportunities: list[dict],
    speed: int = 1,
) -> AsyncGenerator[ReplayEvent, None]:
    """Yield replay events one at a time with speed-controlled delays."""
    delay = 1.0 / speed  # seconds between events

    for opp in opportunities:
        ts = str(opp.get("detected_at", ""))
        market_id = opp.get("market_id", "")
        spread = Decimal(str(opp.get("spread", "0")))
        score = Decimal(str(opp.get("score", "0")))
        filter_reason = opp.get("filter_reason")
        side = opp.get("side", "YES")

        # Market update event
        yield ReplayEvent(
            event_type="market_update",
            timestamp=ts,
            data={
                "market_id": market_id,
                "spread": str(spread.quantize(Decimal("0.00000001"))),
                "score": str(score.quantize(Decimal("0.00000001"))),
                "side": side,
            },
        )

        # Decision event
        if filter_reason:
            decision = "FILTER"
            reason = filter_reason
            risk_result = "N/A"
        elif score < Decimal("0.01"):
            decision = "SKIP"
            reason = "Score below threshold"
            risk_result = "N/A"
        else:
            decision = "EXECUTE"
            reason = "Score above threshold"
            risk_result = "ALLOWED"

        yield ReplayEvent(
            event_type="decision",
            timestamp=ts,
            data={
                "market_id": market_id,
                "detected": "YES+NO arbitrage" if side == "YES" else "Cross-market arbitrage",
                "decision": decision,
                "reason": reason,
                "score": str(score.quantize(Decimal("0.00000001"))),
                "risk_result": risk_result,
            },
        )

        # Speed-controlled delay
        await asyncio.sleep(delay)

    # Done event
    yield ReplayEvent(
        event_type="done",
        timestamp=datetime.now(timezone.utc).isoformat(),
        data={"total_events": len(opportunities) * 2, "total_decisions": len(opportunities)},
    )
