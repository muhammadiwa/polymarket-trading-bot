import hashlib
import logging
import os
import secrets
import time
from asyncio import Lock
from datetime import datetime, timedelta, timezone
from typing import List

from cachetools import TTLCache
from fastapi import HTTPException, Request, Response, status
from jose import JWTError, jwt

from app.config import config

logger = logging.getLogger(__name__)

CSRF_TOKEN_TTL_SECONDS = 3600
CSRF_TOKEN_LENGTH = 32

CSRF_HEADER_NAME = "X-CSRF-Token"
CSRF_COOKIE_NAME = "pqap_csrf"

SESSION_COOKIE_NAME = "pqap_session"
SESSION_COOKIE_HTTPONLY = True
SESSION_COOKIE_SECURE = os.getenv("ENVIRONMENT", "development") == "production"
SESSION_COOKIE_SAMESITE = "lax"

RATE_LIMIT_WINDOW_SECONDS = 60
RATE_LIMIT_MAX_ATTEMPTS = 5
RATE_LIMIT_MAX_ENTRIES = 10000

# #8: Rate limiter using TTLCache for automatic memory management.
# TTLCache automatically evicts entries after TTL expires.
# For single-worker deployments this is correct and avoids Redis dependency
# for the auth path.

# TTLCache with max 10000 entries and 60 second TTL
_rate_limit_store: TTLCache[str, List[float]] = TTLCache(maxsize=RATE_LIMIT_MAX_ENTRIES, ttl=RATE_LIMIT_WINDOW_SECONDS)
_rate_limit_locks: dict[str, Lock] = {}


def create_jwt(user_id: str, username: str, role: str) -> str:
    try:
        expiry_hours = int(os.getenv("AUTH_JWT_EXPIRY", "24"))
    except (ValueError, TypeError):
        expiry_hours = 24
    expire = datetime.now(timezone.utc) + timedelta(hours=expiry_hours)
    claims = {
        "user_id": user_id,
        "username": username,
        "role": role,
        "exp": expire,
        "iat": datetime.now(timezone.utc),
    }
    return jwt.encode(claims, config.JWT_SECRET, algorithm=config.JWT_ALGORITHM)


def decode_jwt(token: str) -> dict:
    try:
        # #14: No token revocation — stolen tokens remain valid until expiry.
        # To add revocation, maintain a Redis blacklist keyed by jti claim
        # and check it here. For now, keep JWT expiry short (see AUTH_JWT_EXPIRY).
        payload = jwt.decode(
            token, config.JWT_SECRET, algorithms=[config.JWT_ALGORITHM],
            options={"verify_exp": True}
        )
        return payload
    except JWTError as e:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid or expired token",
        ) from e


def validate_jwt_claims(payload: dict) -> dict:
    required = {"user_id", "username", "role"}
    if not required.issubset(payload.keys()):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid token claims",
        )
    return payload


def get_token_from_cookie(request: Request) -> str | None:
    return request.cookies.get(SESSION_COOKIE_NAME)


def get_token_from_header(request: Request) -> str | None:
    auth = request.headers.get("Authorization")
    if auth and auth.startswith("Bearer "):
        return auth[7:]
    return None


def extract_user(request: Request) -> dict:
    token = get_token_from_cookie(request) or get_token_from_header(request)
    if not token:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Not authenticated",
        )
    payload = decode_jwt(token)
    return validate_jwt_claims(payload)


async def verify_jwt(request: Request) -> dict:
    """FastAPI dependency for JWT verification. Alias for extract_user."""
    return extract_user(request)


def require_admin(user: dict) -> dict:
    if user.get("role") != "admin":
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Admin access required",
        )
    return user


def generate_csrf_token() -> str:
    return secrets.token_hex(CSRF_TOKEN_LENGTH)


def _hash_token(token: str) -> str:
    return hashlib.sha256(token.encode()).hexdigest()


def set_session_cookie(response: Response, token: str) -> None:
    try:
        expiry_hours = int(os.getenv("AUTH_JWT_EXPIRY", "24"))
    except (ValueError, TypeError):
        expiry_hours = 24
    max_age = expiry_hours * 3600
    response.set_cookie(
        key=SESSION_COOKIE_NAME,
        value=token,
        httponly=SESSION_COOKIE_HTTPONLY,
        secure=SESSION_COOKIE_SECURE,
        samesite=SESSION_COOKIE_SAMESITE,
        max_age=max_age,
        path="/",
    )


def clear_session_cookie(response: Response) -> None:
    response.delete_cookie(key=SESSION_COOKIE_NAME, path="/")
    response.delete_cookie(key=CSRF_COOKIE_NAME, path="/")


def set_csrf_cookie(response: Response, token: str) -> None:
    response.set_cookie(
        key=CSRF_COOKIE_NAME,
        value=token,
        httponly=False,
        secure=SESSION_COOKIE_SECURE,
        samesite=SESSION_COOKIE_SAMESITE,
        max_age=CSRF_TOKEN_TTL_SECONDS,
        path="/",
    )


def check_rate_limit(username: str) -> None:
    if not username or not username.strip():
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Username is required for rate limiting",
        )
    now = time.time()
    window_start = now - RATE_LIMIT_WINDOW_SECONDS

    if username not in _rate_limit_store:
        _rate_limit_store[username] = []

    # Filter attempts within window
    attempts = _rate_limit_store[username]
    _rate_limit_store[username] = [t for t in attempts if t > window_start]

    if len(_rate_limit_store[username]) >= RATE_LIMIT_MAX_ATTEMPTS:
        raise HTTPException(
            status_code=status.HTTP_429_TOO_MANY_REQUESTS,
            detail="Too many login attempts. Try again later.",
        )


def record_login_attempt(username: str) -> None:
    if username not in _rate_limit_store:
        _rate_limit_store[username] = []
    _rate_limit_store[username].append(time.time())


async def check_rate_limit_async(username: str) -> None:
    if not username or not username.strip():
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Username is required for rate limiting",
        )
    # #4: Use setdefault to avoid lock creation race
    lock = _rate_limit_locks.setdefault(username, Lock())
    async with lock:
        check_rate_limit(username)


def create_csrf_token_for_response(response: Response) -> str:
    token = generate_csrf_token()
    set_csrf_cookie(response, token)
    return token
