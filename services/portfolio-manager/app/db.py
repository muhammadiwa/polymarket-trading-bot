import asyncio
import asyncpg
from app.config import config

_pool: asyncpg.Pool = None
_init_lock = asyncio.Lock()


async def init_pool():
    global _pool
    async with _init_lock:
        if _pool is not None:
            return
        # NOTE: Passing credentials via URL string can expose them in process listings
        # (e.g., `ps aux`). Prefer asyncpg.create_pool(host=, port=, user=, password=, database=)
        # with separate params in production.
        _pool = await asyncpg.create_pool(config.POSTGRES_URL, min_size=2, max_size=10)


async def get_pool() -> asyncpg.Pool:
    if _pool is None:
        await init_pool()
    return _pool


async def close_pool():
    global _pool
    async with _init_lock:
        if _pool:
            await _pool.close()
            _pool = None
