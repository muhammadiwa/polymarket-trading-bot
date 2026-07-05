import asyncpg
from app.config import config

_pool: asyncpg.Pool = None


async def init_pool():
    global _pool
    _pool = await asyncpg.create_pool(config.POSTGRES_URL, min_size=2, max_size=10)


async def get_pool() -> asyncpg.Pool:
    if _pool is None:
        await init_pool()
    return _pool


async def close_pool():
    if _pool:
        await _pool.close()
