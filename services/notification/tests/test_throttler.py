import time
from unittest.mock import patch

import pytest

from app.models import Severity
from app.throttler import Throttler


class TestThrottler:
    @pytest.mark.asyncio
    async def test_critical_always_bypasses(self):
        throttler = Throttler(max_per_minute=1)
        for _ in range(100):
            assert await throttler.should_allow(Severity.CRITICAL) is True

    @pytest.mark.asyncio
    async def test_non_critical_respects_limit(self):
        throttler = Throttler(max_per_minute=3)
        assert await throttler.should_allow(Severity.WARNING) is True
        assert await throttler.should_allow(Severity.WARNING) is True
        assert await throttler.should_allow(Severity.WARNING) is True
        assert await throttler.should_allow(Severity.WARNING) is False

    @pytest.mark.asyncio
    async def test_mixed_severities_count_correctly(self):
        throttler = Throttler(max_per_minute=2)
        assert await throttler.should_allow(Severity.WARNING) is True
        assert await throttler.should_allow(Severity.INFO) is True
        assert await throttler.should_allow(Severity.DEBUG) is False

    @pytest.mark.asyncio
    async def test_critical_does_not_consume_quota(self):
        throttler = Throttler(max_per_minute=2)
        assert await throttler.should_allow(Severity.CRITICAL) is True
        assert await throttler.should_allow(Severity.CRITICAL) is True
        assert await throttler.should_allow(Severity.WARNING) is True
        assert await throttler.should_allow(Severity.WARNING) is True
        assert await throttler.should_allow(Severity.WARNING) is False

    @pytest.mark.asyncio
    async def test_window_reset(self):
        throttler = Throttler(max_per_minute=1)
        assert await throttler.should_allow(Severity.INFO) is True
        assert await throttler.should_allow(Severity.INFO) is False

        throttler._window_start = time.monotonic() - 61
        assert await throttler.should_allow(Severity.INFO) is True

    @pytest.mark.asyncio
    async def test_current_count(self):
        throttler = Throttler(max_per_minute=5)
        assert throttler.current_count == 0
        await throttler.should_allow(Severity.INFO)
        assert throttler.current_count == 1
        await throttler.should_allow(Severity.WARNING)
        assert throttler.current_count == 2
