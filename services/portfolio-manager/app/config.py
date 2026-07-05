import os


class Config:
    POSTGRES_URL: str = os.getenv("POSTGRES_URL", "postgres://localhost:5432/pqap")
    NATS_URL: str = os.getenv("NATS_URL", "nats://localhost:4222")
    JWT_SECRET: str = os.getenv("RISK_JWT_SECRET", "")
    JWT_ALGORITHM: str = "HS256"
    LOG_LEVEL: str = os.getenv("PORTFOLIO_LOG_LEVEL", "info")


config = Config()
