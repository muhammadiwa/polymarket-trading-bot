import os


class Config:
    TELEGRAM_BOT_TOKEN: str = os.getenv("TELEGRAM_BOT_TOKEN", "")
    TELEGRAM_CHAT_ID: str = os.getenv("TELEGRAM_CHAT_ID", "")

    NATS_URL: str = os.getenv("NATS_URL", "nats://localhost:4222")
    NATS_SUBJECT: str = os.getenv("NATS_SUBJECT", "pqap.notification.send")

    DATABASE_URL: str = os.getenv("DATABASE_URL", "")

    NOTIFICATION_MAX_PER_MINUTE: int = int(
        os.getenv("NOTIFICATION_MAX_PER_MINUTE", "10")
    )
    NOTIFICATION_HISTORY_LIMIT: int = int(
        os.getenv("NOTIFICATION_HISTORY_LIMIT", "1000")
    )

    CRITICAL_ENABLED: bool = os.getenv("NOTIFICATION_CRITICAL_ENABLED", "true").lower() == "true"
    WARNING_ENABLED: bool = os.getenv("NOTIFICATION_WARNING_ENABLED", "true").lower() == "true"
    INFO_ENABLED: bool = os.getenv("NOTIFICATION_INFO_ENABLED", "true").lower() == "true"
    DEBUG_ENABLED: bool = os.getenv("NOTIFICATION_DEBUG_ENABLED", "false").lower() == "true"

    METRICS_PORT: int = int(os.getenv("METRICS_PORT", "9090"))
    LOG_LEVEL: str = os.getenv("LOG_LEVEL", "info")


config = Config()
