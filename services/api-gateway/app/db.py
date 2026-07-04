import asyncio

import asyncpg
from app.config import config

_pool: asyncpg.Pool | None = None
_pool_lock: asyncio.Lock = asyncio.Lock()


async def get_pool() -> asyncpg.Pool:
    global _pool
    if _pool is not None:
        return _pool
    async with _pool_lock:
        if _pool is None:
            _pool = await asyncpg.create_pool(
                config.POSTGRES_URL,
                min_size=2,
                max_size=10,
            )
    return _pool


async def close_pool() -> None:
    global _pool
    async with _pool_lock:
        if _pool is not None:
            await _pool.close()
            _pool = None
