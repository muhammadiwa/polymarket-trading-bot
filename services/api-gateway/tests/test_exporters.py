import pytest
from datetime import datetime, timezone
from decimal import Decimal

from app.export.csv_exporter import (
    generate_csv_header,
    generate_csv_rows,
    get_csv_filename,
    _format_value,
)
from app.export.json_exporter import (
    trade_to_bytes,
    get_json_filename,
    _trade_to_dict,
)
from app.models.trade import FillStatusEnum, TradeResponse


def _make_trade() -> TradeResponse:
    return TradeResponse(
        id="test-id",
        client_order_id="client-1",
        strategy_id="default",
        market_id="market-1",
        market_slug="test-market",
        side="YES",
        order_type="GTC",
        price=Decimal("0.5500"),
        quantity=Decimal("100.00000000"),
        filled_quantity=Decimal("0"),
        fill_status=FillStatusEnum.PLACED,
        latency_ms=150,
        pnl=Decimal("0"),
        fee=Decimal("0"),
        slippage_pct=Decimal("0"),
        signal_timestamp=datetime(2025, 1, 1, tzinfo=timezone.utc),
        order_timestamp=datetime(2025, 1, 1, tzinfo=timezone.utc),
        risk_decision="APPROVED",
        created_at=datetime(2025, 1, 1, tzinfo=timezone.utc),
    )


def test_generate_csv_header():
    header = generate_csv_header()
    assert "client_order_id" in header
    assert "fill_status" in header


def test_generate_csv_rows():
    trade = _make_trade()
    rows = generate_csv_rows([trade])
    assert "client-1" in rows
    assert "PLACED" in rows


def test_csv_filename_uses_utc():
    filename = get_csv_filename()
    assert filename.startswith("trades_export_")
    assert filename.endswith(".csv")


def test_format_value_none():
    assert _format_value(None) == ""


def test_format_value_decimal():
    assert _format_value(Decimal("0.5500")) == "0.5500"


def test_trade_to_dict():
    trade = _make_trade()
    d = _trade_to_dict(trade)
    assert d["client_order_id"] == "client-1"
    assert d["price"] == "0.5500"


def test_trade_to_bytes():
    trade = _make_trade()
    data = trade_to_bytes(trade)
    assert isinstance(data, bytes)
    assert b"client-1" in data


def test_json_filename_uses_utc():
    filename = get_json_filename()
    assert filename.startswith("trades_export_")
    assert filename.endswith(".json")
