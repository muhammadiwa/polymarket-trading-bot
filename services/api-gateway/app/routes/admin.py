import logging

import asyncpg
from fastapi import APIRouter, Depends, HTTPException, status
from passlib.context import CryptContext
from pydantic import BaseModel, Field

from app.config import config
from app.db import get_pool
from app.middleware.auth import extract_user, require_admin

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/admin", tags=["admin"])

pwd_context = CryptContext(
    schemes=["bcrypt"],
    deprecated="auto",
    bcrypt__rounds=config.BCRYPT_COST,
)


class CreateUserRequest(BaseModel):
    username: str = Field(min_length=1, max_length=64)
    password: str = Field(min_length=8)
    role: str = "viewer"


class UserResponse(BaseModel):
    id: str
    username: str
    role: str


@router.post("/users", response_model=UserResponse)
async def create_user(body: CreateUserRequest, request: dict = Depends(extract_user)):
    require_admin(request)

    if body.role not in ("admin", "viewer"):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Role must be 'admin' or 'viewer'",
        )

    password_hash = pwd_context.hash(body.password)

    pool = await get_pool()
    async with pool.acquire() as conn:
        try:
            row = await conn.fetchrow(
                "INSERT INTO users (username, password_hash, role) VALUES ($1, $2, $3) RETURNING id, username, role",
                body.username,
                password_hash,
                body.role,
            )
        except asyncpg.UniqueViolationError:
            raise HTTPException(
                status_code=status.HTTP_409_CONFLICT,
                detail="Username already exists",
            )
        except Exception as e:
            raise

    return UserResponse(id=str(row["id"]), username=row["username"], role=row["role"])


@router.get("/users", response_model=list[UserResponse])
async def list_users(request: dict = Depends(extract_user)):
    require_admin(request)

    pool = await get_pool()
    async with pool.acquire() as conn:
        rows = await conn.fetch(
            "SELECT id, username, role FROM users ORDER BY created_at DESC"
        )

    return [UserResponse(id=str(r["id"]), username=r["username"], role=r["role"]) for r in rows]
