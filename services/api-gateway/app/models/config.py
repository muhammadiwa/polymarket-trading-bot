from datetime import datetime
from typing import Any, Optional
from uuid import UUID

from pydantic import BaseModel, ConfigDict, Field
from pydantic.alias_generators import to_camel


# Validation patterns per category
VALIDATION_RULES = {
    "api_keys": {
        "min_length": 10,
        "max_length": 500,
        "type": "string",
    },
    "risk_defaults": {
        "min_value": 0,
        "max_value": 100,
        "type": "number",
    },
    "notification_settings": {
        "type": "mixed",  # number or boolean
    },
}

# Keys that should be treated as boolean
BOOLEAN_KEYS = {"critical_bypass_throttle", "enable_telegram", "enable_email"}


def validate_config_value(category: str, key: str, value: Any) -> Any:
    """Validate config value based on category and key rules."""
    rules = VALIDATION_RULES.get(category)
    if not rules:
        raise ValueError(f"Unknown category: {category}")

    if category == "api_keys":
        if not isinstance(value, str):
            raise ValueError(f"API key '{key}' must be a string")
        if len(value) < rules["min_length"] and value != "":
            raise ValueError(f"API key '{key}' must be at least {rules['min_length']} characters")
        if len(value) > rules["max_length"]:
            raise ValueError(f"API key '{key}' must be at most {rules['max_length']} characters")
        return value

    elif category == "risk_defaults":
        try:
            num_value = float(value)
        except (ValueError, TypeError):
            raise ValueError(f"Risk default '{key}' must be a number")
        if num_value < rules["min_value"]:
            raise ValueError(f"Risk default '{key}' must be >= {rules['min_value']}")
        if num_value > rules["max_value"]:
            raise ValueError(f"Risk default '{key}' must be <= {rules['max_value']}")
        return num_value

    elif category == "notification_settings":
        if key in BOOLEAN_KEYS:
            if isinstance(value, bool):
                return value
            if isinstance(value, str):
                if value.lower() in ("true", "1", "yes"):
                    return True
                if value.lower() in ("false", "0", "no"):
                    return False
            raise ValueError(f"Notification setting '{key}' must be a boolean")
        else:
            try:
                int_value = int(value)
            except (ValueError, TypeError):
                raise ValueError(f"Notification setting '{key}' must be an integer")
            if int_value < 0:
                raise ValueError(f"Notification setting '{key}' must be >= 0")
            return int_value

    return value


def mask_sensitive_value(value: str) -> str:
    """Mask sensitive value, showing only first 3 and last 4 characters."""
    if not value or value == '""' or len(value) < 8:
        return "****"
    return f"{value[:3]}...{value[-4:]}"


class SystemConfigBase(BaseModel):
    config_key: str = Field(..., max_length=100)
    config_value: Any
    category: str = Field(..., pattern=r"^(api_keys|risk_defaults|notification_settings)$")
    description: Optional[str] = None
    is_sensitive: bool = False


class SystemConfigCreate(SystemConfigBase):
    pass


class SystemConfigUpdate(BaseModel):
    config_value: Any
    reason: Optional[str] = Field(None, max_length=500)
    expected_updated_at: Optional[datetime] = None


class SystemConfigResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    id: UUID
    config_key: str
    config_value: Any
    category: str
    description: Optional[str] = None
    is_sensitive: bool = False
    created_at: datetime
    updated_at: datetime
    updated_by: Optional[UUID] = None


class SystemConfigListResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    configs: list[SystemConfigResponse]
    total: int


class ConfigAuditLogResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    id: UUID
    config_key: str
    old_value: Optional[Any] = None
    new_value: Any
    changed_by: Optional[UUID] = None
    changed_at: datetime
    reason: Optional[str] = None


class ConfigAuditLogListResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    logs: list[ConfigAuditLogResponse]
    total: int
