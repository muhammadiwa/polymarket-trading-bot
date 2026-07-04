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


MOCK_OPPORTUNITIES = [
    {
        "id": "opp-1",
        "market": "Will Trump win 2024?",
        "market_slug": "trump-2024",
        "score": "0.8500",
        "spread": "2.50",
        "fill_probability": "0.9200",
        "timestamp": "2026-07-04T12:00:00Z",
        "status": "detected",
        "filter_reason": None,
        "execution_latency_ms": None,
    },
    {
        "id": "opp-2",
        "market": "BTC above 100k EOY",
        "market_slug": "btc-100k-eoy",
        "score": "0.7200",
        "spread": "1.25",
        "fill_probability": "0.8500",
        "timestamp": "2026-07-04T11:59:30Z",
        "status": "executed",
        "filter_reason": None,
        "execution_latency_ms": 45,
    },
]


@pytest.mark.anyio
async def test_opportunities_requires_auth():
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.get("/api/opportunities")
        assert resp.status_code == 403


@pytest.mark.anyio
async def test_opportunities_returns_paginated_list(mock_jwt):
    mock_conn = AsyncMock()
    mock_pool = AsyncMock()
    mock_pool.acquire.return_value.__aenter__ = AsyncMock(return_value=mock_conn)
    mock_pool.acquire.return_value.__aexit__ = AsyncMock(return_value=False)

    with patch("app.routes.opportunities.get_pool", return_value=mock_pool), \
         patch("app.routes.opportunities.list_opportunities", return_value=(MOCK_OPPORTUNITIES, 2, None)):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            resp = await client.get("/api/opportunities", headers=mock_jwt)
            assert resp.status_code == 200
            data = resp.json()
            assert "opportunities" in data
            assert "total_count" in data
            assert "next_cursor" in data
            assert len(data["opportunities"]) == 2


@pytest.mark.anyio
async def test_opportunities_has_correct_fields(mock_jwt):
    mock_conn = AsyncMock()
    mock_pool = AsyncMock()
    mock_pool.acquire.return_value.__aenter__ = AsyncMock(return_value=mock_conn)
    mock_pool.acquire.return_value.__aexit__ = AsyncMock(return_value=False)

    with patch("app.routes.opportunities.get_pool", return_value=mock_pool), \
         patch("app.routes.opportunities.list_opportunities", return_value=(MOCK_OPPORTUNITIES, 2, None)):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            resp = await client.get("/api/opportunities", headers=mock_jwt)
            data = resp.json()
            opp = data["opportunities"][0]
            assert "id" in opp
            assert "market" in opp
            assert "market_slug" in opp
            assert "score" in opp
            assert "spread" in opp
            assert "fill_probability" in opp
            assert "timestamp" in opp
            assert "status" in opp


@pytest.mark.anyio
async def test_opportunities_accepts_status_filter(mock_jwt):
    mock_conn = AsyncMock()
    mock_pool = AsyncMock()
    mock_pool.acquire.return_value.__aenter__ = AsyncMock(return_value=mock_conn)
    mock_pool.acquire.return_value.__aexit__ = AsyncMock(return_value=False)

    filtered = [o for o in MOCK_OPPORTUNITIES if o["status"] == "executed"]

    with patch("app.routes.opportunities.get_pool", return_value=mock_pool), \
         patch("app.routes.opportunities.list_opportunities", return_value=(filtered, 1, None)):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            resp = await client.get("/api/opportunities?status=executed", headers=mock_jwt)
            assert resp.status_code == 200
            data = resp.json()
            assert len(data["opportunities"]) == 1
            assert data["opportunities"][0]["status"] == "executed"


@pytest.mark.anyio
async def test_opportunities_accepts_pagination_params(mock_jwt):
    mock_conn = AsyncMock()
    mock_pool = AsyncMock()
    mock_pool.acquire.return_value.__aenter__ = AsyncMock(return_value=mock_conn)
    mock_pool.acquire.return_value.__aexit__ = AsyncMock(return_value=False)

    with patch("app.routes.opportunities.get_pool", return_value=mock_pool), \
         patch("app.routes.opportunities.list_opportunities", return_value=([MOCK_OPPORTUNITIES[0]], 2, "2026-07-04T12:00:00Z")):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            resp = await client.get("/api/opportunities?page_size=1", headers=mock_jwt)
            assert resp.status_code == 200
            data = resp.json()
            assert len(data["opportunities"]) == 1
            assert data["next_cursor"] is not None


@pytest.mark.anyio
async def test_opportunities_handles_empty_result(mock_jwt):
    mock_conn = AsyncMock()
    mock_pool = AsyncMock()
    mock_pool.acquire.return_value.__aenter__ = AsyncMock(return_value=mock_conn)
    mock_pool.acquire.return_value.__aexit__ = AsyncMock(return_value=False)

    with patch("app.routes.opportunities.get_pool", return_value=mock_pool), \
         patch("app.routes.opportunities.list_opportunities", return_value=([], 0, None)):
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            resp = await client.get("/api/opportunities", headers=mock_jwt)
            assert resp.status_code == 200
            data = resp.json()
            assert data["opportunities"] == []
            assert data["total_count"] == 0
            assert data["next_cursor"] is None
