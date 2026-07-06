import os


class Config:
    POSTGRES_URL: str = os.getenv("POSTGRES_URL", "postgres://localhost:5432/pqap")
    NATS_URL: str = os.getenv("NATS_URL", "nats://localhost:4222")
    JWT_SECRET: str = os.getenv("RISK_JWT_SECRET", "")
    JWT_ALGORITHM: str = "HS256"
    LOG_LEVEL: str = os.getenv("ANALYTICS_LOG_LEVEL", "info")
    SHARPE_RISK_FREE_RATE: float = float(os.getenv("ANALYTICS_RISK_FREE_RATE", "0"))

    # Anomaly detection thresholds
    ANOMALY_WIN_RATE_DROP: float = float(os.getenv("ANOMALY_WIN_RATE_DROP", "0.20"))
    ANOMALY_DRAWDOWN_MULTIPLIER: float = float(os.getenv("ANOMALY_DRAWDOWN_MULTIPLIER", "2.0"))
    ANOMALY_CONSECUTIVE_LOSSES: int = int(os.getenv("ANOMALY_CONSECUTIVE_LOSSES", "5"))
    ANOMALY_PROFIT_FACTOR_LOW: float = float(os.getenv("ANOMALY_PROFIT_FACTOR_LOW", "0.5"))
    ANOMALY_SHARPE_LOW: float = float(os.getenv("ANOMALY_SHARPE_LOW", "0"))
    ANOMALY_CHECK_INTERVAL: int = int(os.getenv("ANOMALY_CHECK_INTERVAL_SECONDS", "3600"))


config = Config()

if not config.JWT_SECRET:
    raise RuntimeError("RISK_JWT_SECRET must be set")
