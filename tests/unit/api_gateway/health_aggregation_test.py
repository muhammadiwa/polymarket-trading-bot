import json
import pytest
from httpx import AsyncClient, ASGITransport
from unittest.mock import patch, AsyncMock, MagicMock

from app.main import app


@pytest.fixture
def anyio_backend():
    return "asyncio"


@pytest.fixture
def mock_jwt():
    with patch("app.middleware.auth.config") as mock_config:
        mock_config.JWT_SECRET = "test-secret"
        mock_config.JWT_ALGORITHM = "HS256"
        from jose import jwt

        token = jwt.encode({"sub": "test-user"}, "test-secret", algorithm="HS256")
        yield {"Authorization": f"Bearer {token}"}


MOCK_HEALTH_RESPONSE = {
    "scanner": {
        "name": "Scanner",
        "status": "up",
        "wsConnected": True,
        "cpuPercent": 45.2,
        "memoryMB": 512,
        "errorRate": 0.3,
        "lastHeartbeat": "2026-07-04T12:00:00Z",
    },
    "arbEngine": {
        "name": "Arb Engine",
        "status": "up",
        "wsConnected": False,
        "cpuPercent": 72.5,
        "memoryMB": 256,
        "errorRate": 1.2,
        "lastHeartbeat": "2026-07-04T12:00:00Z",
    },
    "executionEngine": {
        "name": "Execution Engine",
        "status": "up",
        "wsConnected": False,
        "cpuPercent": 30.0,
        "memoryMB": 384,
        "errorRate": 0.1,
        "lastHeartbeat": "2026-07-04T12:00:00Z",
    },
    "riskManager": {
        "name": "Risk Manager",
        "status": "up",
        "wsConnected": False,
        "cpuPercent": 20.1,
        "memoryMB": 128,
        "errorRate": 0.0,
        "lastHeartbeat": "2026-07-04T12:00:00Z",
    },
    "positionManager": {
        "name": "Position Manager",
        "status": "up",
        "wsConnected": False,
        "cpuPercent": 15.0,
        "memoryMB": 96,
        "errorRate": 0.0,
        "lastHeartbeat": "2026-07-04T12:00:00Z",
    },
    "overall": "healthy",
    "lastUpdated": "2026-07-04T12:00:00Z",
}


@pytest.mark.anyio
async def test_system_health_requires_auth():
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.get("/api/system/health")
        assert resp.status_code == 403


@pytest.mark.anyio
async def test_system_health_returns_cached_data(mock_jwt):
    mock_redis = AsyncMock()
    mock_redis.get = AsyncMock(return_value=json.dumps(MOCK_HEALTH_RESPONSE))

    with patch("app.routes.health._get_redis", return_value=mock_redis):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            resp = await client.get("/api/system/health", headers=mock_jwt)
            assert resp.status_code == 200
            data = resp.json()
            assert data["overall"] == "healthy"
            assert "scanner" in data
            assert "arbEngine" in data
            assert "executionEngine" in data
            assert "riskManager" in data
            assert "positionManager" in data


@pytest.mark.anyio
async def test_system_health_returns_overall_status(mock_jwt):
    mock_redis = AsyncMock()
    mock_redis.get = AsyncMock(return_value=json.dumps(MOCK_HEALTH_RESPONSE))

    with patch("app.routes.health._get_redis", return_value=mock_redis):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            resp = await client.get("/api/system/health", headers=mock_jwt)
            data = resp.json()
            assert data["overall"] in ("healthy", "degraded", "unhealthy")


@pytest.mark.anyio
async def test_system_health_has_service_fields(mock_jwt):
    mock_redis = AsyncMock()
    mock_redis.get = AsyncMock(return_value=json.dumps(MOCK_HEALTH_RESPONSE))

    with patch("app.routes.health._get_redis", return_value=mock_redis):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            resp = await client.get("/api/system/health", headers=mock_jwt)
            data = resp.json()
            scanner = data["scanner"]
            assert "name" in scanner
            assert "status" in scanner
            assert "cpuPercent" in scanner
            assert "memoryMB" in scanner
            assert "errorRate" in scanner
            assert "lastHeartbeat" in scanner
