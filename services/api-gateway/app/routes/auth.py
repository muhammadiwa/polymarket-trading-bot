import logging
import uuid
from datetime import datetime, timezone

from fastapi import APIRouter, HTTPException, Request, Response, status
from passlib.context import CryptContext

from app.config import config
from app.db import get_pool
from app.middleware.auth import (
    check_rate_limit_async,
    clear_session_cookie,
    create_csrf_token_for_response,
    create_jwt,
    extract_user,
    record_login_attempt,
    require_admin,
    set_csrf_cookie,
    set_session_cookie,
)
from app.models.user import LoginRequest, LoginResponse, UserResponse

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/auth", tags=["auth"])

pwd_context = CryptContext(
    schemes=["bcrypt"],
    deprecated="auto",
    bcrypt__rounds=config.BCRYPT_COST,
)


def _verify_password(plain: str, hashed: str) -> bool:
    return pwd_context.verify(plain, hashed)


def _hash_password(password: str) -> str:
    return pwd_context.hash(password)


@router.post("/login", response_model=LoginResponse)
async def login(body: LoginRequest, request: Request, response: Response):
    await check_rate_limit_async(body.username)

    pool = await get_pool()
    async with pool.acquire() as conn:
        row = await conn.fetchrow(
            "SELECT id, username, password_hash, role, created_at, last_login FROM users WHERE username = $1",
            body.username,
        )

    if not row:
        record_login_attempt(body.username)
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid credentials",
        )

    if not _verify_password(body.password, row["password_hash"]):
        record_login_attempt(body.username)
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid credentials",
        )

    token = create_jwt(str(row["id"]), row["username"], row["role"])

    pool_conn = await get_pool()
    async with pool_conn.acquire() as conn:
        await conn.execute(
            "UPDATE users SET last_login = $1 WHERE id = $2",
            datetime.now(timezone.utc),
            row["id"],
        )

    set_session_cookie(response, token)
    csrf_token = create_csrf_token_for_response(response)

    user = UserResponse(
        id=str(row["id"]),
        username=row["username"],
        role=row["role"],
        created_at=row["created_at"],
        last_login=row["last_login"],
    )

    return LoginResponse(access_token=token, user=user)


@router.post("/logout")
async def logout(response: Response):
    clear_session_cookie(response)
    return {"status": "ok"}


@router.get("/me", response_model=UserResponse)
async def me(request: Request):
    user = extract_user(request)

    pool = await get_pool()
    async with pool.acquire() as conn:
        row = await conn.fetchrow(
            "SELECT id, username, role, created_at, last_login FROM users WHERE id = $1::uuid",
            user["user_id"],
        )

    if not row:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="User not found",
        )

    return UserResponse(
        id=str(row["id"]),
        username=row["username"],
        role=row["role"],
        created_at=row["created_at"],
        last_login=row["last_login"],
    )


@router.get("/csrf")
async def get_csrf_token(response: Response):
    csrf_token = create_csrf_token_for_response(response)
    return {"csrf_token": csrf_token}


@router.post("/refresh")
async def refresh_token(request: Request, response: Response):
    user = extract_user(request)

    pool = await get_pool()
    async with pool.acquire() as conn:
        row = await conn.fetchrow(
            "SELECT id, username, role FROM users WHERE id = $1",
            user["user_id"],
        )

    if not row:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="User not found",
        )

    new_token = create_jwt(str(row["id"]), row["username"], row["role"])
    set_session_cookie(response, new_token)

    return {"access_token": new_token, "token_type": "bearer"}
