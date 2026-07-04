from datetime import datetime, timezone

import pytest

from app.models import (
    CategoryConfig,
    Channel,
    ChannelConfig,
    NotificationPreferences,
    Severity,
)


class TestNotificationPreferences:
    def test_critical_always_enabled(self):
        prefs = NotificationPreferences(
            categories=CategoryConfig(critical=False, warning=False, info=False, debug=False)
        )
        assert prefs.is_enabled(Severity.CRITICAL) is True

    def test_warning_can_be_disabled(self):
        prefs = NotificationPreferences(
            categories=CategoryConfig(warning=False)
        )
        assert prefs.is_enabled(Severity.WARNING) is False

    def test_info_can_be_disabled(self):
        prefs = NotificationPreferences(
            categories=CategoryConfig(info=False)
        )
        assert prefs.is_enabled(Severity.INFO) is False

    def test_debug_disabled_by_default(self):
        prefs = NotificationPreferences()
        assert prefs.is_enabled(Severity.DEBUG) is False

    def test_all_enabled(self):
        prefs = NotificationPreferences(
            categories=CategoryConfig(critical=True, warning=True, info=True, debug=True)
        )
        assert prefs.is_enabled(Severity.CRITICAL) is True
        assert prefs.is_enabled(Severity.WARNING) is True
        assert prefs.is_enabled(Severity.INFO) is True
        assert prefs.is_enabled(Severity.DEBUG) is True

    def test_channel_telegram_enabled_by_default(self):
        prefs = NotificationPreferences()
        assert prefs.is_channel_enabled(Channel.TELEGRAM) is True

    def test_channel_email_disabled_by_default(self):
        prefs = NotificationPreferences()
        assert prefs.is_channel_enabled(Channel.EMAIL) is False

    def test_channel_disable_telegram(self):
        prefs = NotificationPreferences(
            channels=ChannelConfig(telegram=False)
        )
        assert prefs.is_channel_enabled(Channel.TELEGRAM) is False

    def test_channel_enable_email(self):
        prefs = NotificationPreferences(
            channels=ChannelConfig(email=True)
        )
        assert prefs.is_channel_enabled(Channel.EMAIL) is True

    def test_to_row_roundtrip(self):
        prefs = NotificationPreferences(
            categories=CategoryConfig(critical=True, warning=False, info=True, debug=False),
            channels=ChannelConfig(telegram=True, email=False, chat_id="123", email_to="a@b.com"),
        )
        row = prefs.to_row()
        assert row == (True, False, True, False, True, False, "123", "a@b.com")

    def test_from_row(self):
        row = {
            "warning": True,
            "info": False,
            "debug": True,
            "telegram_enabled": True,
            "email_enabled": False,
            "telegram_chat_id": "456",
            "email_address": None,
            "updated_at": datetime.now(timezone.utc),
        }
        prefs = NotificationPreferences.from_row(row)
        assert prefs.categories.critical is True
        assert prefs.categories.warning is True
        assert prefs.categories.info is False
        assert prefs.categories.debug is True
        assert prefs.channels.telegram is True
        assert prefs.channels.email is False
        assert prefs.channels.chat_id == "456"
        assert prefs.channels.email_to == ""
