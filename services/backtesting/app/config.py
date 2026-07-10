import os


class Config:
    POSTGRES_URL: str = os.getenv("POSTGRES_URL", "postgres://localhost:5432/pqap")
    TIMESCALE_URL: str = os.getenv("TIMESCALE_URL", "postgres://localhost:5432/pqap")
    JWT_SECRET: str = os.getenv("JWT_SECRET", "")
    JWT_ALGORITHM: str = "HS256"
    LOG_LEVEL: str = os.getenv("BACKTESTING_LOG_LEVEL", "info")

    # Simulation defaults
    DEFAULT_SLIPPAGE_PCT: float = float(os.getenv("BACKTEST_SLIPPAGE_PCT", "0.01"))
    DEFAULT_PARTIAL_FILL_PROB: float = float(os.getenv("BACKTEST_PARTIAL_FILL_PROB", "0.1"))
    DEFAULT_LATENCY_MS: int = int(os.getenv("BACKTEST_LATENCY_MS", "100"))
    DEFAULT_MIN_FILL_RATIO: float = float(os.getenv("BACKTEST_MIN_FILL_RATIO", "0.5"))
    DEFAULT_RNG_SEED: int = int(os.getenv("BACKTEST_RNG_SEED", "42"))


config = Config()

if not config.JWT_SECRET:
    raise RuntimeError("JWT_SECRET must be set")
