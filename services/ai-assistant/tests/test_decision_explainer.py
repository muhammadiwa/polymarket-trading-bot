from decimal import Decimal

import pytest

from app.engine.decision_explainer import _build_decision_context


class TestBuildDecisionContext:
    def test_build_decision_context_profitable_trade(self):
        """Test context building for profitable trade."""
        trade = {
            "market_id": "test-market",
            "side": "YES",
            "entry_price": Decimal("0.60"),
            "exit_price": Decimal("0.80"),
            "pnl": Decimal("100.00"),
            "strategy_id": "strat-1",
        }
        risk_events = []

        result = _build_decision_context(trade, risk_events)

        assert "test-market" in result
        assert "YES" in result
        assert "Profitable trade" in result
        assert "+$100.00" in result

    def test_build_decision_context_losing_trade(self):
        """Test context building for losing trade."""
        trade = {
            "market_id": "test-market",
            "side": "NO",
            "entry_price": Decimal("0.40"),
            "exit_price": Decimal("0.60"),
            "pnl": Decimal("-50.00"),
            "strategy_id": "strat-2",
        }
        risk_events = []

        result = _build_decision_context(trade, risk_events)

        assert "Losing trade" in result
        assert "$-50.00" in result

    def test_build_decision_context_with_risk_events(self):
        """Test context building with risk events."""
        trade = {
            "market_id": "test-market",
            "side": "YES",
            "entry_price": Decimal("0.60"),
            "exit_price": Decimal("0.70"),
            "pnl": Decimal("50.00"),
            "strategy_id": "strat-1",
        }
        risk_events = [
            {
                "event_type": "pit_boss_check",
                "decision": "approved",
                "reason": "Within risk limits",
            },
            {
                "event_type": "slippage_check",
                "decision": "approved",
                "reason": "Slippage within tolerance",
            },
        ]

        result = _build_decision_context(trade, risk_events)

        assert "Risk events" in result
        assert "pit_boss_check" in result
        assert "Within risk limits" in result

    def test_build_decision_context_no_pnl(self):
        """Test context building with no PnL (open position)."""
        trade = {
            "market_id": "test-market",
            "side": "YES",
            "entry_price": Decimal("0.60"),
            "exit_price": None,
            "pnl": None,
            "strategy_id": "strat-1",
        }
        risk_events = []

        result = _build_decision_context(trade, risk_events)

        assert "test-market" in result
        assert "Profitable" not in result
        assert "Losing" not in result
