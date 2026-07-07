from fastapi import HTTPException, Request, status
from jose import JWTError, jwt

from app.config import config


def verify_jwt(request: Request) -> dict:
    token = None
    if "pqap_session" in request.cookies:
        token = request.cookies["pqap_session"]
    if not token:
        auth = request.headers.get("Authorization")
        if auth and auth.startswith("Bearer "):
            token = auth[7:]
    if not token:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Not authenticated")
    try:
        payload = jwt.decode(token, config.JWT_SECRET, algorithms=[config.JWT_ALGORITHM], options={"verify_exp": True})
    except JWTError:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid or expired token")
    required = {"user_id", "username", "role"}
    if not required.issubset(payload.keys()):
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid token claims")
    return payload
