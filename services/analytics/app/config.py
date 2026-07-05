import os


class Config:
    POSTGRES_URL: str = os.getenv("POSTGRES_URL", "postgres://localhost:5432/pqap")
    JWT_SECRET: str = os.getenv("RISK_JWT_SECRET", "")
    JWT_ALGORITHM: str = "HS256"
    LOG_LEVEL: str = os.getenv("ANALYTICS_LOG_LEVEL", "info")
    SHARPE_RISK_FREE_RATE: float = float(os.getenv("ANALYTICS_RISK_FREE_RATE", "0"))


config = Config()

if not config.JWT_SECRET:
    raise RuntimeError("RISK_JWT_SECRET must be set")
