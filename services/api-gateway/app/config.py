import os
import sys


class Config:
    POSTGRES_URL: str = os.getenv(
        "POSTGRES_URL", "postgres://localhost:5432/pqap"
    )
    JWT_SECRET: str = os.getenv("JWT_SECRET", "")
    JWT_ALGORITHM: str = "HS256"
    JWT_EXPIRY_HOURS: int = int(os.getenv("AUTH_JWT_EXPIRY", "24"))
    BCRYPT_COST: int = int(os.getenv("AUTH_BCRYPT_COST", "12"))
    CSRF_ENABLED: bool = os.getenv("AUTH_CSRF_ENABLED", "true").lower() == "true"
    API_HOST: str = os.getenv("API_HOST", "0.0.0.0")
    API_PORT: int = int(os.getenv("API_PORT", "8080"))
    TRADE_HISTORY_BATCH_SIZE: int = int(os.getenv("TRADE_HISTORY_BATCH_SIZE", "1000"))
    TRADE_HISTORY_MAX_PAGE_SIZE: int = int(os.getenv("TRADE_HISTORY_MAX_PAGE_SIZE", "100"))
    TRADE_HISTORY_DEFAULT_PAGE_SIZE: int = int(os.getenv("TRADE_HISTORY_DEFAULT_PAGE_SIZE", "50"))
    TRADE_HISTORY_EXPORT_TIMEOUT: int = int(os.getenv("TRADE_HISTORY_EXPORT_TIMEOUT", "30"))
    TRADE_HISTORY_RETENTION_YEARS: int = int(os.getenv("TRADE_HISTORY_RETENTION_YEARS", "7"))
    LOG_LEVEL: str = os.getenv("LOG_LEVEL", "info")
    NATS_URL: str = os.getenv("NATS_URL", "nats://localhost:4222")
    CORS_ORIGINS: list[str] = [o.strip() for o in os.getenv("CORS_ORIGINS", "http://localhost:3000").split(",") if o.strip()]
    PORTFOLIO_SERVICE_URL: str = os.getenv("PORTFOLIO_SERVICE_URL", "http://localhost:8081")
    POSITION_SERVICE_URL: str = os.getenv("POSITION_SERVICE_URL", "http://localhost:8082")
    RISK_MANAGER_URL: str = os.getenv("RISK_MANAGER_URL", "http://localhost:8083")
    REDIS_URL: str = os.getenv("REDIS_URL", "redis://localhost:6379")
    # #16: Service URLs for health polling (configurable, not hardcoded)
    SCANNER_URL: str = os.getenv("SCANNER_URL", "http://localhost:8091")
    ARB_ENGINE_URL: str = os.getenv("ARB_ENGINE_URL", "http://localhost:8092")
    EXECUTION_ENGINE_URL: str = os.getenv("EXECUTION_ENGINE_URL", "http://localhost:8093")
    POSITION_MANAGER_URL: str = os.getenv("POSITION_MANAGER_URL", "http://localhost:8095")
    BACKTESTING_URL: str = os.getenv("BACKTESTING_URL", "http://localhost:8096")
    AI_OPTIMIZER_URL: str = os.getenv("AI_OPTIMIZER_URL", "http://localhost:8097")
    ACCOUNT_MANAGER_URL: str = os.getenv("ACCOUNT_MANAGER_URL", "http://localhost:8098")
    PROMETHEUS_URL: str = os.getenv("PROMETHEUS_URL", "http://localhost:9090")
    # Internal API key for service-to-service communication
    INTERNAL_API_KEY: str = os.getenv("INTERNAL_API_KEY", "")
    # Backup storage directory
    BACKUP_DIR: str = os.getenv("BACKUP_DIR", "/var/backups/pqap")
    # Default risk limits (used when no per-account limits set)
    DEFAULT_DAILY_LOSS_LIMIT_PCT: str = os.getenv("DEFAULT_DAILY_LOSS_LIMIT_PCT", "2.0")
    DEFAULT_MAX_POSITION_PER_MARKET_PCT: str = os.getenv("DEFAULT_MAX_POSITION_PER_MARKET_PCT", "10.0")
    DEFAULT_MAX_POSITION_PER_STRATEGY_PCT: str = os.getenv("DEFAULT_MAX_POSITION_PER_STRATEGY_PCT", "20.0")
    DEFAULT_DRAWDOWN_THRESHOLD_PCT: str = os.getenv("DEFAULT_DRAWDOWN_THRESHOLD_PCT", "10.0")


config = Config()

if not config.JWT_SECRET:
    print("FATAL: JWT_SECRET environment variable is empty. Refusing to start.", file=sys.stderr)
    sys.exit(1)
