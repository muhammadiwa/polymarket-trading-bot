import pytest

from app.categorizer import classify, get_emoji, EVENT_SEVERITY_MAP
from app.models import Severity


class TestClassify:
    def test_critical_events(self):
        critical_events = [
            "EmergencyStop",
            "CircuitBreakerTripped",
            "APIFailure",
            "DrawdownBreach",
            "DailyBudgetExhausted",
        ]
        for event in critical_events:
            assert classify(event) == Severity.CRITICAL

    def test_warning_events(self):
        warning_events = [
            "DailyBudget80Percent",
            "DrawdownApproaching",
            "PositionLimitBreach",
            "WinStreak",
        ]
        for event in warning_events:
            assert classify(event) == Severity.WARNING

    def test_info_events(self):
        info_events = ["OrderFilled", "TradeExecuted", "StrategyOptimization"]
        for event in info_events:
            assert classify(event) == Severity.INFO

    def test_debug_events(self):
        debug_events = [
            "SystemHealth",
            "ReconnectionEvent",
            "ReconciliationComplete",
        ]
        for event in debug_events:
            assert classify(event) == Severity.DEBUG

    def test_unknown_event_defaults_to_info(self):
        assert classify("UnknownEventType") == Severity.INFO

    def test_all_14_event_types_mapped(self):
        assert len(EVENT_SEVERITY_MAP) == 15


class TestGetEmoji:
    def test_critical_emoji(self):
        assert get_emoji(Severity.CRITICAL) == "\U0001f534"

    def test_warning_emoji(self):
        assert get_emoji(Severity.WARNING) == "\U0001f7e1"

    def test_info_emoji(self):
        assert get_emoji(Severity.INFO) == "\U0001f535"

    def test_debug_emoji(self):
        assert get_emoji(Severity.DEBUG) == "\u26aa"
