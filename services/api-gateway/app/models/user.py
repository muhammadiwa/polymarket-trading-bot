import uuid
from datetime import datetime, timezone

from pydantic import BaseModel, Field


class User(BaseModel):
    id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    username: str
    password_hash: str = Field(exclude=True)
    role: str = "viewer"
    created_at: datetime = Field(default_factory=lambda: datetime.now(timezone.utc))
    last_login: datetime | None = None


class UserResponse(BaseModel):
    id: str
    username: str
    role: str
    created_at: datetime
    last_login: datetime | None


class LoginRequest(BaseModel):
    username: str = Field(min_length=1, max_length=64)
    password: str = Field(min_length=1)


class LoginResponse(BaseModel):
    access_token: str
    token_type: str = "bearer"
    user: UserResponse


class CSRFToken(BaseModel):
    token: str
    expires_at: datetime


class JWTClaims(BaseModel):
    user_id: str
    username: str
    role: str
