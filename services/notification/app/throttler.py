import asyncio
import logging
import time
from collections import deque

import redis.asyncio as redis

from app.models import Severity

logger = logging.getLogger(__name__)

KEY_PREFIX = "pqap:throttle"
WINDOW_SECONDS = 60


class Throttler:
    def __init__(self, max_per_minute: int = 10, redis_url: str = ""):
        self.max_per_minute = max_per_minute
        self._redis_url = redis_url
        self._redis: redis.Redis | None = None
        self._timestamps: deque[float] = deque()
        self._lock = asyncio.Lock()

    async def connect(self) -> None:
        if self._redis_url:
            try:
                self._redis = redis.from_url(
                    self._redis_url, decode_responses=True
                )
                await self._redis.ping()
                logger.info("Connected to Redis for throttle state")
            except redis.RedisError:
                logger.warning(
                    "Redis unavailable, falling back to in-memory throttle"
                )
                self._redis = None

    async def close(self) -> None:
        if self._redis:
            await self._redis.aclose()
            self._redis = None

    async def should_allow(self, severity: Severity) -> bool:
        if severity == Severity.CRITICAL:
            return True

        if self._redis:
            return await self._redis_allow(severity)
        return await self._memory_allow()

    async def _redis_allow(self, severity: Severity) -> bool:
        key = f"{KEY_PREFIX}:{severity.value}"
        now = time.time()
        window_start = now - WINDOW_SECONDS

        try:
            pipe = self._redis.pipeline()
            pipe.zremrangebyscore(key, 0, window_start)
            pipe.zadd(key, {str(now): now})
            pipe.zcard(key)
            pipe.expire(key, WINDOW_SECONDS)
            results = await pipe.execute()

            count = results[2]
            return count <= self.max_per_minute
        except redis.RedisError:
            logger.warning("Redis throttle check failed, using in-memory fallback")
            return await self._memory_allow()

    async def _memory_allow(self) -> bool:
        async with self._lock:
            # #13: Use time.time() (wall clock) for consistency with Redis path.
            # Both paths now use the same time source to avoid drift.
            now = time.time()
            cutoff = now - WINDOW_SECONDS
            while self._timestamps and self._timestamps[0] < cutoff:
                self._timestamps.popleft()

            if len(self._timestamps) >= self.max_per_minute:
                return False

            self._timestamps.append(now)
            return True

    async def get_current_count(self) -> int:
        async with self._lock:
            now = time.time()
            cutoff = now - WINDOW_SECONDS
            return sum(1 for ts in self._timestamps if ts >= cutoff)
