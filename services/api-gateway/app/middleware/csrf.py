import hmac
import logging
import os

from fastapi import HTTPException, Request, status
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.responses import Response

logger = logging.getLogger(__name__)

CSRF_HEADER_NAME = "X-CSRF-Token"
CSRF_COOKIE_NAME = "pqap_csrf"

SAFE_METHODS = {"GET", "HEAD", "OPTIONS"}

# Auth endpoints exempt from CSRF — they establish the session/CSRF token itself
CSRF_EXEMPT_PATHS = {
    "/api/auth/login",
    "/api/auth/logout",
    "/api/auth/csrf",
    "/api/auth/refresh",
}


class CSRFMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next) -> Response:
        if os.getenv("AUTH_CSRF_ENABLED", "true").lower() != "true":
            # #12: Log warning when CSRF is disabled
            if os.getenv("ENVIRONMENT", "development") == "production":
                logger.critical("CSRF protection is DISABLED in production environment!")
            return await call_next(request)

        if request.method.upper() in SAFE_METHODS:
            return await call_next(request)

        if not request.url.path.startswith("/api/"):
            return await call_next(request)

        # Exempt auth endpoints from CSRF (they establish the session)
        if request.url.path in CSRF_EXEMPT_PATHS:
            return await call_next(request)

        header_token = request.headers.get(CSRF_HEADER_NAME)
        cookie_token = request.cookies.get(CSRF_COOKIE_NAME)

        if not header_token or not cookie_token:
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="CSRF token missing",
            )

        if not hmac.compare_digest(header_token, cookie_token):
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="CSRF token mismatch",
            )

        return await call_next(request)
