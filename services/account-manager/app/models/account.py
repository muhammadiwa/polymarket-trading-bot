from datetime import datetime
from typing import Optional

from pydantic import BaseModel, ConfigDict, Field
from pydantic.alias_generators import to_camel


class AccountCreate(BaseModel):
    name: str = Field(min_length=1, max_length=100)
    wallet_address: str = Field(min_length=42, max_length=42, pattern=r"^0x[0-9a-fA-F]{40}$")
    private_key: str = Field(min_length=1, max_length=256)


class AccountUpdate(BaseModel):
    name: Optional[str] = Field(None, min_length=1, max_length=100)


class AccountResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    id: str
    name: str
    wallet_address: str
    is_active: bool
    created_at: datetime
    updated_at: datetime


class AccountListResponse(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)

    items: list[AccountResponse]
    total: int
