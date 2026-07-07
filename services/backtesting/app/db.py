import asyncio
import asyncpg
from app.config import config

_pg_pool: asyncpg.Pool = None
_ts_pool: asyncpg.Pool = None
_init_lock = asyncio.Lock()


async def init_pools():
    global _pg_pool, _ts_pool
    async with _init_lock:
        if _pg_pool is None:
            _pg_pool = await asyncpg.create_pool(config.POSTGRES_URL, min_size=2, max_size=10)
        if _ts_pool is None:
            _ts_pool = await asyncpg.create_pool(config.TIMESCALE_URL, min_size=2, max_size=10)


async def get_pg_pool() -> asyncpg.Pool:
    global _pg_pool
    if _pg_pool is None:
        await init_pools()
    return _pg_pool


async def get_ts_pool() -> asyncpg.Pool:
    global _ts_pool
    if _ts_pool is None:
        await init_pools()
    return _ts_pool


async def close_pools():
    global _pg_pool, _ts_pool
    async with _init_lock:
        if _pg_pool:
            await _pg_pool.close()
            _pg_pool = None
        if _ts_pool:
            await _ts_pool.close()
            _ts_pool = None
