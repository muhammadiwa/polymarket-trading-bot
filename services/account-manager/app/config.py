import os


class Config:
    POSTGRES_URL: str = os.getenv("POSTGRES_URL", "")
    JWT_SECRET: str = os.getenv("RISK_JWT_SECRET", "")
    JWT_ALGORITHM: str = "HS256"
    LOG_LEVEL: str = os.getenv("ACCOUNT_MANAGER_LOG_LEVEL", "info")
    CORS_ORIGINS: str = os.getenv("CORS_ORIGINS", "http://localhost:3000")

    # Encryption
    ENCRYPTION_MASTER_KEY: str = os.getenv("ENCRYPTION_MASTER_KEY", "")


config = Config()

if not config.POSTGRES_URL:
    raise RuntimeError("POSTGRES_URL must be set")

if not config.JWT_SECRET:
    raise RuntimeError("RISK_JWT_SECRET must be set")

if not config.ENCRYPTION_MASTER_KEY:
    raise RuntimeError("ENCRYPTION_MASTER_KEY must be set")
