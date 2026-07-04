import pytest
from datetime import datetime, timezone
from decimal import Decimal
from unittest.mock import AsyncMock, MagicMock

from app.repos.trade_repo import _row_to_trade, _build_where
from app.models.trade import TradeFilterParams, FillStatusEnum


def test_row_to_trade_maps_all_fields():
    row = {
        "id": "test-uuid",
        "client_order_id": "client-1",
        "strategy_id": "default",
        "market_id": "market-1",
        "market_slug": "test-market",
        "side": "YES",
        "order_type": "GTC",
        "price": Decimal("0.5500"),
        "quantity": Decimal("100.00000000"),
        "filled_quantity": Decimal("0"),
        "fill_status": "PLACED",
        "latency_ms": 150,
        "pnl": Decimal("0"),
        "fee": Decimal("0"),
        "slippage_pct": Decimal("0"),
        "signal_timestamp": datetime.now(timezone.utc),
        "order_timestamp": datetime.now(timezone.utc),
        "fill_timestamp": None,
        "opportunity_id": None,
        "risk_decision": "APPROVED",
        "failure_reason": None,
        "account_id": None,
        "created_at": datetime.now(timezone.utc),
    }
    trade = _row_to_trade(row)
    assert trade.client_order_id == "client-1"
    assert trade.fill_status == FillStatusEnum.PLACED
    assert trade.price == Decimal("0.5500")


def test_build_where_empty_filters():
    filters = TradeFilterParams()
    where, params = _build_where(filters)
    assert where == "1=1"
    assert params == []


def test_build_where_with_market_id():
    filters = TradeFilterParams(market_id="market-1")
    where, params = _build_where(filters)
    assert "market_id" in where
    assert params == ["market-1"]


def test_build_where_with_date_range():
    start = datetime(2025, 1, 1, tzinfo=timezone.utc)
    end = datetime(2025, 12, 31, tzinfo=timezone.utc)
    filters = TradeFilterParams(start_date=start, end_date=end)
    where, params = _build_where(filters)
    assert "created_at >=" in where
    assert "created_at <=" in where
    assert len(params) == 2


@pytest.mark.asyncio
async def test_list_trades_returns_cursor():
    pass


@pytest.mark.asyncio
async def test_export_trades_uses_server_side_cursor():
    pass
