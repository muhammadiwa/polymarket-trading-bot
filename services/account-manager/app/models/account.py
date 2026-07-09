from datetime import datetime
from typing import Optional

from pydantic import BaseModel, Field


class AccountCreate(BaseModel):
    name: str = Field(min_length=1, max_length=100)
    wallet_address: str = Field(min_length=42, max_length=42, pattern=r"^0x[0-9a-fA-F]{40}$")
    private_key: str = Field(min_length=1, max_length=256)


class AccountUpdate(BaseModel):
    name: Optional[str] = Field(None, min_length=1, max_length=100)


class AccountResponse(BaseModel):
    id: str
    name: str
    wallet_address: str
    is_active: bool
    created_at: datetime
    updated_at: datetime


class AccountListResponse(BaseModel):
    accounts: list[AccountResponse]
    total: int
