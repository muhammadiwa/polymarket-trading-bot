import os


class Config:
    POSTGRES_URL: str = os.getenv("POSTGRES_URL", "")
    JWT_SECRET: str = os.getenv("RISK_JWT_SECRET", "")
    JWT_ALGORITHM: str = "HS256"
    LOG_LEVEL: str = os.getenv("AI_ASSISTANT_LOG_LEVEL", "info")

    # LLM Configuration
    LLM_API_KEY: str = os.getenv("LLM_API_KEY", "")
    LLM_BASE_URL: str = os.getenv("LLM_BASE_URL", "https://api.openai.com/v1")
    LLM_MODEL: str = os.getenv("LLM_MODEL", "gpt-4o-mini")

    # Risk Manager API
    RISK_MANAGER_URL: str = os.getenv("RISK_MANAGER_URL", "http://risk-manager:8080")


config = Config()

if not config.POSTGRES_URL:
    raise RuntimeError("POSTGRES_URL must be set")

if not config.JWT_SECRET:
    raise RuntimeError("RISK_JWT_SECRET must be set")

if not config.LLM_API_KEY:
    raise RuntimeError("LLM_API_KEY must be set")

if not config.LLM_BASE_URL:
    raise RuntimeError("LLM_BASE_URL must be set")
