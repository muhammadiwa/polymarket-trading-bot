from decimal import Decimal

import pytest

from app.engine.performance_qa import (
    _format_market_data,
    _format_pnl_summary,
    _format_strategy_data,
)


class TestFormatPnlSummary:
    def test_format_pnl_summary(self):
        """Test PnL summary formatting."""
        summary = {
            "period": "week",
            "total_pnl": Decimal("1234.56"),
            "trade_count": 100,
            "winning_trades": 65,
            "losing_trades": 35,
            "win_rate": 65.0,
            "avg_pnl": Decimal("12.35"),
            "best_trade": Decimal("150.00"),
            "worst_trade": Decimal("-50.00"),
        }

        result = _format_pnl_summary(summary)

        assert len(result) == 9
        assert any(dp["label"] == "Total PnL" and "$1234.56" in dp["value"] for dp in result)
        assert any(dp["label"] == "Win Rate" and "65.0%" in dp["value"] for dp in result)
        assert any(dp["label"] == "Trade Count" and "100" in dp["value"] for dp in result)


class TestFormatStrategyData:
    def test_format_strategy_data_with_strategies(self):
        """Test strategy data formatting with data."""
        strategies = [
            {"strategy_id": "strat-1", "total_pnl": Decimal("500.00"), "trade_count": 50, "win_rate": 70.0},
            {"strategy_id": "strat-2", "total_pnl": Decimal("300.00"), "trade_count": 30, "win_rate": 60.0},
        ]

        result = _format_strategy_data(strategies)

        assert len(result) == 2
        assert "strat-1" in result[0]["label"]
        assert "$500.00" in result[0]["value"]

    def test_format_strategy_data_empty(self):
        """Test strategy data formatting with no data."""
        result = _format_strategy_data([])

        assert len(result) == 1
        assert "No strategy data" in result[0]["value"]


class TestFormatMarketData:
    def test_format_market_data_with_trades(self):
        """Test market data formatting with trades."""
        trades = [
            {"id": "1", "pnl": Decimal("100.00")},
            {"id": "2", "pnl": Decimal("-50.00")},
            {"id": "3", "pnl": Decimal("75.00")},
        ]

        result = _format_market_data(trades)

        assert len(result) == 2
        assert any(dp["label"] == "Market Trade Count" and "3" in dp["value"] for dp in result)
        assert any(dp["label"] == "Market Total PnL" and "$125.00" in dp["value"] for dp in result)

    def test_format_market_data_empty(self):
        """Test market data formatting with no trades."""
        result = _format_market_data([])

        assert len(result) == 1
        assert "No trades found" in result[0]["value"]
