import pytest
from httpx import AsyncClient, ASGITransport
from unittest.mock import patch

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


@pytest.mark.anyio
async def test_portfolio_overview_returns_200(mock_jwt):
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.get("/api/portfolio/overview", headers=mock_jwt)
        assert resp.status_code == 200
        data = resp.json()
        assert "totalCapital" in data
        assert "dailyPnL" in data
        assert "totalPnL" in data
        assert "utilizationRate" in data
        assert "lastUpdated" in data


@pytest.mark.anyio
async def test_positions_returns_200(mock_jwt):
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.get("/api/positions", headers=mock_jwt)
        assert resp.status_code == 200
        assert isinstance(resp.json(), list)


@pytest.mark.anyio
async def test_portfolio_overview_requires_auth():
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.get("/api/portfolio/overview")
        assert resp.status_code == 403


@pytest.mark.anyio
async def test_positions_requires_auth():
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.get("/api/positions")
        assert resp.status_code == 403
