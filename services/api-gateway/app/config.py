import os
import sys


class Config:
    POSTGRES_URL: str = os.getenv(
        "POSTGRES_URL", "postgres://localhost:5432/pqap"
    )
    JWT_SECRET: str = os.getenv("JWT_SECRET", "")
    JWT_ALGORITHM: str = "HS256"
    API_HOST: str = os.getenv("API_HOST", "0.0.0.0")
    API_PORT: int = int(os.getenv("API_PORT", "8080"))
    TRADE_HISTORY_BATCH_SIZE: int = int(os.getenv("TRADE_HISTORY_BATCH_SIZE", "1000"))
    TRADE_HISTORY_MAX_PAGE_SIZE: int = int(os.getenv("TRADE_HISTORY_MAX_PAGE_SIZE", "100"))
    TRADE_HISTORY_DEFAULT_PAGE_SIZE: int = int(os.getenv("TRADE_HISTORY_DEFAULT_PAGE_SIZE", "50"))
    TRADE_HISTORY_EXPORT_TIMEOUT: int = int(os.getenv("TRADE_HISTORY_EXPORT_TIMEOUT", "30"))
    TRADE_HISTORY_RETENTION_YEARS: int = int(os.getenv("TRADE_HISTORY_RETENTION_YEARS", "7"))
    LOG_LEVEL: str = os.getenv("LOG_LEVEL", "info")


config = Config()

if not config.JWT_SECRET:
    print("FATAL: JWT_SECRET environment variable is empty. Refusing to start.", file=sys.stderr)
    sys.exit(1)
