import asyncio
import logging
import time
import uuid

from fastapi import APIRouter, Depends, HTTPException, Query
from prometheus_client import Counter

from app.config import config
from app.db import get_pool
from app.engine.encryption import WalletEncryption
from app.middleware.auth import verify_jwt
from app.models.account import (
    AccountCreate,
    AccountListResponse,
    AccountResponse,
    AccountUpdate,
)
from app.repos import account_repo

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/accounts", tags=["accounts"])

# Prometheus metrics
ACCOUNT_CREATED_TOTAL = Counter("pqap_account_created_total", "Total accounts created")
ACCOUNT_UPDATED_TOTAL = Counter("pqap_account_updated_total", "Total accounts updated")
ACCOUNT_DEACTIVATED_TOTAL = Counter("pqap_account_deactivated_total", "Total accounts deactivated")
ACCOUNT_ACTIVATED_TOTAL = Counter("pqap_account_activated_total", "Total accounts activated")

# Rate limiting with TTL eviction
_rate_limit_cache: dict[str, float] = {}
_rate_limit_lock = asyncio.Lock()
RATE_LIMIT_SECONDS = 5
RATE_LIMIT_TTL = 300  # 5 minutes
RATE_LIMIT_MAX_ENTRIES = 10000

encryption = WalletEncryption(config.ENCRYPTION_MASTER_KEY)


async def _check_rate_limit(user_id: str) -> bool:
    """Check if user is within rate limit."""
    now = time.time()
    async with _rate_limit_lock:
        # Evict stale entries if cache is large
        if len(_rate_limit_cache) > RATE_LIMIT_MAX_ENTRIES:
            cutoff = now - RATE_LIMIT_TTL
            stale = [k for k, v in _rate_limit_cache.items() if v < cutoff]
            for k in stale:
                del _rate_limit_cache[k]

        last_request = _rate_limit_cache.get(user_id, 0)
        if now - last_request < RATE_LIMIT_SECONDS:
            return False
        _rate_limit_cache[user_id] = now
        return True


def _rate_limit_error() -> HTTPException:
    return HTTPException(
        status_code=429,
        detail=f"Rate limit exceeded. Please wait {RATE_LIMIT_SECONDS} seconds.",
        headers={"Retry-After": str(RATE_LIMIT_SECONDS)},
    )


def _validate_uuid(account_id: str) -> uuid.UUID:
    """Validate UUID format and return UUID object."""
    try:
        return uuid.UUID(account_id)
    except ValueError:
        raise HTTPException(status_code=400, detail="Invalid account ID format")


def _sanitize_name(name: str) -> str:
    """Sanitize account name."""
    if not name:
        return name
    # Strip whitespace
    name = name.strip()
    # Remove null bytes
    name = name.replace("\x00", "")
    return name


@router.post("", response_model=AccountResponse)
async def create_account(
    request: AccountCreate,
    user: dict = Depends(verify_jwt),
):
    """Create a new account with encrypted private key."""
    user_id = user.get("user_id")
    if not user_id:
        raise HTTPException(status_code=400, detail="Missing user_id in token")
    if not await _check_rate_limit(user_id):
        raise _rate_limit_error()

    try:
        pool = await get_pool()
    except Exception as e:
        logger.error("failed to get database pool", extra={"error": str(e)})
        raise HTTPException(status_code=503, detail="Database unavailable")

    async with pool.acquire() as conn:
        # Check for duplicate wallet address
        existing = await account_repo.get_account_by_wallet(conn, request.wallet_address)
        if existing:
            raise HTTPException(status_code=409, detail="Wallet address already configured")

        # Sanitize name input
        sanitized_name = _sanitize_name(request.name)

        # Use transaction to ensure atomicity
        async with conn.transaction():
            # Step 1: Create account WITHOUT private key to get DB-generated UUID
            account = await account_repo.create_account(
                conn,
                name=sanitized_name,
                wallet_address=request.wallet_address,
                private_key_encrypted=b"",  # Placeholder
                private_key_iv=b"",  # Placeholder
                private_key_tag=b"",  # Placeholder
            )

            # Step 2: Encrypt private key using the ACTUAL account ID from DB
            account_id = account["id"]
            try:
                encrypted_data = encryption.encrypt_private_key(request.private_key, account_id)
            except Exception as e:
                logger.error("encryption failed", extra={"error": str(e), "account_id": account_id})
                raise HTTPException(status_code=500, detail="Failed to encrypt private key")

            # Step 3: Update account with encrypted private key
            account = await account_repo.update_encrypted_key(
                conn,
                account_id=account_id,
                private_key_encrypted=encrypted_data["encrypted"],
                private_key_iv=encrypted_data["iv"],
                private_key_tag=encrypted_data["tag"],
            )

    ACCOUNT_CREATED_TOTAL.inc()
    logger.info("account created", extra={"account_id": account["id"], "user": user.get("username")})
    return AccountResponse(**account)


@router.get("", response_model=AccountListResponse)
async def list_accounts(
    is_active: bool = None,
    limit: int = Query(default=50, ge=1, le=1000),
    offset: int = Query(default=0, ge=0),
    user: dict = Depends(verify_jwt),
):
    """List all accounts."""
    try:
        pool = await get_pool()
        async with pool.acquire() as conn:
            accounts, total = await account_repo.list_accounts(conn, is_active, limit, offset)
    except Exception as e:
        logger.error("failed to list accounts", extra={"error": str(e)})
        raise HTTPException(status_code=503, detail="Database unavailable")

    return AccountListResponse(accounts=[AccountResponse(**a) for a in accounts], total=total)


@router.get("/{account_id}", response_model=AccountResponse)
async def get_account(
    account_id: str,
    user: dict = Depends(verify_jwt),
):
    """Get account details."""
    validated_id = _validate_uuid(account_id)

    try:
        pool = await get_pool()
        async with pool.acquire() as conn:
            account = await account_repo.get_account_by_id(conn, str(validated_id))
    except Exception as e:
        logger.error("failed to get account", extra={"error": str(e), "account_id": account_id})
        raise HTTPException(status_code=503, detail="Database unavailable")

    if account is None:
        raise HTTPException(status_code=404, detail="Account not found")

    return AccountResponse(**account)


@router.put("/{account_id}", response_model=AccountResponse)
async def update_account(
    account_id: str,
    request: AccountUpdate,
    user: dict = Depends(verify_jwt),
):
    """Update account configuration."""
    validated_id = _validate_uuid(account_id)

    user_id = user.get("user_id")
    if not await _check_rate_limit(user_id):
        raise _rate_limit_error()

    pool = await get_pool()
    async with pool.acquire() as conn:
        account = await account_repo.update_account(conn, str(validated_id), name=request.name)

    if account is None:
        raise HTTPException(status_code=404, detail="Account not found")

    ACCOUNT_UPDATED_TOTAL.inc()
    logger.info("account updated", extra={"account_id": account_id, "user": user.get("username")})
    return AccountResponse(**account)


@router.delete("/{account_id}", response_model=AccountResponse)
async def deactivate_account(
    account_id: str,
    user: dict = Depends(verify_jwt),
):
    """Deactivate account (soft delete)."""
    validated_id = _validate_uuid(account_id)

    user_id = user.get("user_id")
    if not await _check_rate_limit(user_id):
        raise _rate_limit_error()

    pool = await get_pool()
    async with pool.acquire() as conn:
        account = await account_repo.set_account_active(conn, str(validated_id), is_active=False)

    if account is None:
        raise HTTPException(status_code=404, detail="Account not found")

    ACCOUNT_DEACTIVATED_TOTAL.inc()
    logger.info("account deactivated", extra={"account_id": account_id, "user": user.get("username")})
    return AccountResponse(**account)


@router.post("/{account_id}/activate", response_model=AccountResponse)
async def activate_account(
    account_id: str,
    user: dict = Depends(verify_jwt),
):
    """Activate account."""
    validated_id = _validate_uuid(account_id)

    user_id = user.get("user_id")
    if not await _check_rate_limit(user_id):
        raise _rate_limit_error()

    pool = await get_pool()
    async with pool.acquire() as conn:
        account = await account_repo.set_account_active(conn, str(validated_id), is_active=True)

    if account is None:
        raise HTTPException(status_code=404, detail="Account not found")

    ACCOUNT_ACTIVATED_TOTAL.inc()
    logger.info("account activated", extra={"account_id": account_id, "user": user.get("username")})
    return AccountResponse(**account)


@router.post("/{account_id}/deactivate", response_model=AccountResponse)
async def deactivate_account_alt(
    account_id: str,
    user: dict = Depends(verify_jwt),
):
    """Deactivate account (alternative endpoint)."""
    return await deactivate_account(account_id, user)
