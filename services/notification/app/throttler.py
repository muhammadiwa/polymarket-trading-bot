import asyncio
import time

from app.models import Severity


class Throttler:
    def __init__(self, max_per_minute: int = 10):
        self.max_per_minute = max_per_minute
        self._window_start: float = time.monotonic()
        self._count: int = 0
        self._lock = asyncio.Lock()

    async def should_allow(self, severity: Severity) -> bool:
        if severity == Severity.CRITICAL:
            return True

        async with self._lock:
            now = time.monotonic()
            if now - self._window_start >= 60:
                self._window_start += 60
                self._count = 0

            if self._count >= self.max_per_minute:
                return False

            self._count += 1
            return True

    @property
    def current_count(self) -> int:
        return self._count
