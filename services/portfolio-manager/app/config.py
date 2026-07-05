import os
import sys


class Config:
    POSTGRES_URL: str = os.getenv("POSTGRES_URL", "postgres://localhost:5432/pqap")
    NATS_URL: str = os.getenv("NATS_URL", "nats://localhost:4222")
    JWT_SECRET: str = os.getenv("RISK_JWT_SECRET", "")
    JWT_ALGORITHM: str = "HS256"
    LOG_LEVEL: str = os.getenv("PORTFOLIO_LOG_LEVEL", "info")


config = Config()

# #2: Fail fast if JWT_SECRET is empty
if not config.JWT_SECRET:
    print("WARNING: RISK_JWT_SECRET is empty — JWT tokens can be forged!", file=sys.stderr)
