import pytest
import os
from unittest.mock import patch, MagicMock
from datetime import datetime, timezone

os.environ["JWT_SECRET"] = "test-secret-key-for-testing-only"
os.environ["AUTH_CSRF_ENABLED"] = "false"

from app.middleware.auth import (
    create_jwt,
    decode_jwt,
    validate_jwt_claims,
    generate_csrf_token,
    check_rate_limit,
    record_login_attempt,
    _rate_limit_store,
)


class TestJWT:
    def test_create_jwt_returns_valid_token(self):
        token = create_jwt("user-123", "testuser", "admin")
        assert isinstance(token, str)
        assert len(token) > 0

    def test_decode_jwt_valid_token(self):
        token = create_jwt("user-123", "testuser", "admin")
        payload = decode_jwt(token)
        assert payload["user_id"] == "user-123"
        assert payload["username"] == "testuser"
        assert payload["role"] == "admin"

    def test_decode_jwt_expired_token(self):
        from jose import jwt

        claims = {
            "user_id": "user-123",
            "username": "testuser",
            "role": "admin",
            "exp": datetime(2020, 1, 1, tzinfo=timezone.utc),
            "iat": datetime(2019, 12, 31, tzinfo=timezone.utc),
        }
        token = jwt.encode(claims, "test-secret-key-for-testing-only", algorithm="HS256")

        from fastapi import HTTPException
        with pytest.raises(HTTPException) as exc_info:
            decode_jwt(token)
        assert exc_info.value.status_code == 401

    def test_decode_jwt_invalid_token(self):
        from fastapi import HTTPException
        with pytest.raises(HTTPException) as exc_info:
            decode_jwt("invalid-token")
        assert exc_info.value.status_code == 401

    def test_validate_jwt_claims_valid(self):
        payload = {"user_id": "u1", "username": "test", "role": "viewer"}
        result = validate_jwt_claims(payload)
        assert result == payload

    def test_validate_jwt_claims_missing_field(self):
        from fastapi import HTTPException
        payload = {"user_id": "u1", "username": "test"}
        with pytest.raises(HTTPException) as exc_info:
            validate_jwt_claims(payload)
        assert exc_info.value.status_code == 401

    def test_jwt_roundtrip_different_roles(self):
        for role in ["admin", "viewer"]:
            token = create_jwt("user-123", "testuser", role)
            payload = decode_jwt(token)
            assert payload["role"] == role


class TestCSRF:
    def test_generate_csrf_token_returns_string(self):
        token = generate_csrf_token()
        assert isinstance(token, str)
        assert len(token) > 0

    def test_generate_csrf_token_unique(self):
        tokens = {generate_csrf_token() for _ in range(10)}
        assert len(tokens) == 10


class TestRateLimit:
    def setup_method(self):
        _rate_limit_store.clear()

    def test_rate_limit_allows_first_attempt(self):
        check_rate_limit("testuser")

    def test_rate_limit_blocks_after_max_attempts(self):
        from fastapi import HTTPException
        for _ in range(5):
            record_login_attempt("testuser")

        with pytest.raises(HTTPException) as exc_info:
            check_rate_limit("testuser")
        assert exc_info.value.status_code == 429

    def test_rate_limit_different_users_independent(self):
        for _ in range(5):
            record_login_attempt("user1")

        check_rate_limit("user2")

    def test_rate_limit_window_expiry(self):
        import time
        for _ in range(5):
            record_login_attempt("testuser")

        _rate_limit_store["testuser"] = [t - 61 for t in _rate_limit_store["testuser"]]
        check_rate_limit("testuser")
