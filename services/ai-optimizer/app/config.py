import os


class Config:
    POSTGRES_URL: str = os.getenv("POSTGRES_URL", "postgres://localhost:5432/pqap")
    JWT_SECRET: str = os.getenv("JWT_SECRET", "")
    JWT_ALGORITHM: str = "HS256"
    LOG_LEVEL: str = os.getenv("AI_OPTIMIZER_LOG_LEVEL", "info")


config = Config()

if not config.JWT_SECRET:
    raise RuntimeError("JWT_SECRET must be set")
