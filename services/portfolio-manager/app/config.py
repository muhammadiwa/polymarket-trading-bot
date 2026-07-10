import os


class Config:
    POSTGRES_URL: str = os.getenv("POSTGRES_URL", "postgres://localhost:5432/pqap")
    NATS_URL: str = os.getenv("NATS_URL", "nats://localhost:4222")
    JWT_SECRET: str = os.getenv("JWT_SECRET", "")
    JWT_ALGORITHM: str = "HS256"
    LOG_LEVEL: str = os.getenv("PORTFOLIO_LOG_LEVEL", "info")


config = Config()

# Fail fast if JWT_SECRET is empty — prevents token forgery
if not config.JWT_SECRET:
    raise RuntimeError("JWT_SECRET must be set — empty value allows token forgery")
