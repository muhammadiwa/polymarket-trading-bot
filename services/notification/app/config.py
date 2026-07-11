import os


def _int_env(name: str, default: str) -> int:
    raw = os.getenv(name, default)
    try:
        return int(raw)
    except ValueError:
        raise ValueError(f"Environment variable {name} must be an integer, got {raw!r}")


class Config:
    TELEGRAM_BOT_TOKEN: str = os.getenv("TELEGRAM_BOT_TOKEN", "")
    TELEGRAM_CHAT_ID: str = os.getenv("TELEGRAM_CHAT_ID", "")

    NATS_URL: str = os.getenv("NATS_URL", "nats://localhost:4222")
    NATS_SUBJECT: str = os.getenv("NATS_SUBJECT", "pqap.notification.request")

    DATABASE_URL: str = os.getenv("POSTGRES_URL", os.getenv("DATABASE_URL", ""))
    REDIS_URL: str = os.getenv("REDIS_URL", "redis://localhost:6379")

    NOTIFICATION_MAX_PER_MINUTE: int = _int_env("NOTIFICATION_MAX_PER_MINUTE", "10")
    NOTIFICATION_HISTORY_LIMIT: int = _int_env("NOTIFICATION_HISTORY_LIMIT", "1000")

    CRITICAL_ENABLED: bool = os.getenv("NOTIFICATION_CRITICAL_ENABLED", "true").lower() == "true"
    WARNING_ENABLED: bool = os.getenv("NOTIFICATION_WARNING_ENABLED", "true").lower() == "true"
    INFO_ENABLED: bool = os.getenv("NOTIFICATION_INFO_ENABLED", "true").lower() == "true"
    DEBUG_ENABLED: bool = os.getenv("NOTIFICATION_DEBUG_ENABLED", "false").lower() == "true"

    METRICS_PORT: int = _int_env("METRICS_PORT", "9090")
    LOG_LEVEL: str = os.getenv("LOG_LEVEL", "info")
    INTERNAL_API_KEY: str = os.getenv("INTERNAL_API_KEY", "")


config = Config()
