from datetime import datetime, timezone

import pytest

from app.formatter import format_message
from app.models import Channel, NotificationRecord, NotificationStatus, Severity


def _make_record(severity: Severity, title: str = "Test", message: str = "Body"):
    return NotificationRecord(
        event_type="TestEvent",
        severity=severity,
        title=title,
        message=message,
        channel=Channel.TELEGRAM,
        status=NotificationStatus.SENT,
    )


class TestFormatMessage:
    def test_critical_format(self):
        record = _make_record(Severity.CRITICAL, "Emergency Stop", "Trading halted")
        text = format_message(record)
        assert "\U0001f534" in text
        assert "CRITICAL" in text
        assert "EMERGENCY STOP" in text
        assert "Trading halted" in text
        assert "Event: TestEvent" in text

    def test_warning_format(self):
        record = _make_record(Severity.WARNING, "Budget Alert", "80% used")
        text = format_message(record)
        assert "\U0001f7e1" in text
        assert "Warning:" in text
        assert "Budget Alert" in text
        assert "80% used" in text

    def test_info_format(self):
        record = _make_record(Severity.INFO, "Order Filled", "Filled at 0.55")
        text = format_message(record)
        assert "\U0001f535" in text
        assert "Info:" in text
        assert "Order Filled" in text

    def test_debug_format(self):
        record = _make_record(Severity.DEBUG, "Health Check", "All systems nominal")
        text = format_message(record)
        assert "\u26aa" in text
        assert "Debug:" in text
        assert "Health Check" in text

    def test_critical_has_separator(self):
        record = _make_record(Severity.CRITICAL)
        text = format_message(record)
        assert "\u2501\u2501\u2501" in text
