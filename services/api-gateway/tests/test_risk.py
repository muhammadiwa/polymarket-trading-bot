import json
import pytest
from httpx import AsyncClient, ASGITransport
from unittest.mock import patch, AsyncMock

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


def _make_risk_state(**overrides):
    base = {
        "daily_budget_remaining": "500.00",
        "daily_loss_limit": "1000.00",
        "daily_loss": "500.00",
        "capital": "10000.00",
        "drawdown": "0.03",
        "drawdown_limit": "0.10",
        "emergency_stop": False,
        "emergency_stop_reason": "",
        "emergency_stop_timestamp": None,
        "batasi_win_paused": False,
        "updated_at": "2025-01-01T00:00:00+00:00",
        "win_streak_current": 3,
        "win_streak_threshold": 5,
    }
    base.update(overrides)
    return json.dumps(base)


@pytest.mark.anyio
async def test_risk_status_returns_200(mock_jwt):
    with patch("app.routes.risk._get_redis") as mock_get_redis:
        r = AsyncMock()
        r.get.return_value = _make_risk_state()
        r.aclose = AsyncMock()
        mock_get_redis.return_value = r

        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            resp = await client.get("/api/risk/status", headers=mock_jwt)
            assert resp.status_code == 200
            data = resp.json()
            assert data["dailyBudgetRemaining"] == "500.00"
            assert data["currentDrawdown"] == "0.03"
            assert data["circuitBreakerStatus"] == "closed"
            assert data["isPaused"] is False
            assert data["winStreakCurrent"] == 3
            assert data["winStreakThreshold"] == 5


@pytest.mark.anyio
async def test_risk_status_emergency_active(mock_jwt):
    with patch("app.routes.risk._get_redis") as mock_get_redis:
        r = AsyncMock()
        r.get.return_value = _make_risk_state(
            emergency_stop=True,
            emergency_stop_reason="Manual stop",
            emergency_stop_timestamp="2025-01-01T12:00:00+00:00",
        )
        r.aclose = AsyncMock()
        mock_get_redis.return_value = r

        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            resp = await client.get("/api/risk/status", headers=mock_jwt)
            assert resp.status_code == 200
            data = resp.json()
            assert data["circuitBreakerStatus"] == "open"
            assert data["isPaused"] is True
            assert data["pausedReason"] == "Manual stop"


@pytest.mark.anyio
async def test_risk_status_batasi_paused(mock_jwt):
    with patch("app.routes.risk._get_redis") as mock_get_redis:
        r = AsyncMock()
        r.get.return_value = _make_risk_state(batasi_win_paused=True)
        r.aclose = AsyncMock()
        mock_get_redis.return_value = r

        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            resp = await client.get("/api/risk/status", headers=mock_jwt)
            assert resp.status_code == 200
            data = resp.json()
            assert data["isPaused"] is True
            assert data["pausedReason"] == "Win streak threshold reached"


@pytest.mark.anyio
async def test_risk_status_no_state_returns_503(mock_jwt):
    with patch("app.routes.risk._get_redis") as mock_get_redis:
        r = AsyncMock()
        r.get.return_value = None
        r.aclose = AsyncMock()
        mock_get_redis.return_value = r

        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            resp = await client.get("/api/risk/status", headers=mock_jwt)
            assert resp.status_code == 503


@pytest.mark.anyio
async def test_risk_status_requires_auth():
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.get("/api/risk/status")
        assert resp.status_code == 403


@pytest.mark.anyio
async def test_emergency_stop_success(mock_jwt):
    with (
        patch("app.routes.risk._get_redis") as mock_get_redis,
        patch("app.routes.risk._publish_nats") as mock_nats,
    ):
        r = AsyncMock()
        r.get.side_effect = lambda key: {
            "pqap:risk:emergency_token:valid-token-uuid": "1",
            "pqap:risk:state": _make_risk_state(),
        }.get(key)
        r.set = AsyncMock()
        r.delete = AsyncMock()
        r.aclose = AsyncMock()
        mock_get_redis.return_value = r

        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            resp = await client.post(
                "/api/risk/emergency-stop",
                json={"reason": "Test stop", "confirmationToken": "valid-token-uuid"},
                headers=mock_jwt,
            )
            assert resp.status_code == 200
            data = resp.json()
            assert data["status"] == "emergency_stop_activated"
            assert r.set.called
            r.delete.assert_called_once_with("pqap:risk:emergency_token:valid-token-uuid")
            mock_nats.assert_called_once()


@pytest.mark.anyio
async def test_emergency_stop_invalid_token(mock_jwt):
    with patch("app.routes.risk._get_redis") as mock_get_redis:
        r = AsyncMock()
        r.get.return_value = None
        r.aclose = AsyncMock()
        mock_get_redis.return_value = r

        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            resp = await client.post(
                "/api/risk/emergency-stop",
                json={"reason": "Test stop", "confirmationToken": "bad-token"},
                headers=mock_jwt,
            )
            assert resp.status_code == 400
            assert "confirmation token" in resp.json()["detail"].lower()


@pytest.mark.anyio
async def test_generate_confirmation_token(mock_jwt):
    with patch("app.routes.risk._get_redis") as mock_get_redis:
        r = AsyncMock()
        r.set = AsyncMock()
        r.aclose = AsyncMock()
        mock_get_redis.return_value = r

        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            resp = await client.post(
                "/api/risk/emergency-stop/confirmationToken",
                headers=mock_jwt,
            )
            assert resp.status_code == 200
            data = resp.json()
            assert "confirmationToken" in data
            assert len(data["confirmationToken"]) > 0


@pytest.mark.anyio
async def test_pause_success(mock_jwt):
    with (
        patch("app.routes.risk._get_redis") as mock_get_redis,
        patch("app.routes.risk._publish_nats") as mock_nats,
    ):
        r = AsyncMock()
        r.get.return_value = _make_risk_state()
        r.set = AsyncMock()
        r.aclose = AsyncMock()
        mock_get_redis.return_value = r

        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            resp = await client.post(
                "/api/risk/pause",
                json={"reason": "Manual pause"},
                headers=mock_jwt,
            )
            assert resp.status_code == 200
            assert resp.json()["status"] == "trading_paused"
            mock_nats.assert_called_once()


@pytest.mark.anyio
async def test_resume_success(mock_jwt):
    with (
        patch("app.routes.risk._get_redis") as mock_get_redis,
        patch("app.routes.risk._publish_nats") as mock_nats,
    ):
        r = AsyncMock()
        r.get.return_value = _make_risk_state(emergency_stop=True)
        r.set = AsyncMock()
        r.aclose = AsyncMock()
        mock_get_redis.return_value = r

        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            resp = await client.post("/api/risk/resume", headers=mock_jwt)
            assert resp.status_code == 200
            assert resp.json()["status"] == "trading_resumed"
            mock_nats.assert_called_once()


@pytest.mark.anyio
async def test_update_parameters_success(mock_jwt):
    with (
        patch("app.routes.risk._get_redis") as mock_get_redis,
        patch("app.routes.risk._publish_nats") as mock_nats,
        patch("app.routes.risk.get_pool") as mock_pool,
    ):
        r = AsyncMock()
        r.get.return_value = _make_risk_state()
        r.set = AsyncMock()
        r.aclose = AsyncMock()
        mock_get_redis.return_value = r

        conn = AsyncMock()
        conn.execute = AsyncMock()

        class MockAcquire:
            async def __aenter__(self):
                return conn
            async def __aexit__(self, *args):
                pass

        pool = AsyncMock()
        pool.acquire = MockAcquire
        mock_pool.return_value = pool

        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            resp = await client.put(
                "/api/risk/parameters",
                json={"dailyLossLimit": "5"},
                headers=mock_jwt,
            )
            assert resp.status_code == 200
            data = resp.json()
            assert data["status"] == "parameters_updated"
            assert "dailyLossLimit" in data["changes"]
            mock_nats.assert_called_once()
            conn.execute.assert_called_once()


@pytest.mark.anyio
async def test_update_parameters_invalid_range(mock_jwt):
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.put(
            "/api/risk/parameters",
            json={"dailyLossLimit": "25"},
            headers=mock_jwt,
        )
        assert resp.status_code == 422


@pytest.mark.anyio
async def test_update_parameters_negative_value(mock_jwt):
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.put(
            "/api/risk/parameters",
            json={"dailyLossLimit": "-5"},
            headers=mock_jwt,
        )
        assert resp.status_code == 422


@pytest.mark.anyio
async def test_emergency_stop_requires_auth():
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.post(
            "/api/risk/emergency-stop",
            json={"reason": "Test", "confirmationToken": "tok"},
        )
        assert resp.status_code == 403


@pytest.mark.anyio
async def test_pause_requires_auth():
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.post("/api/risk/pause", json={})
        assert resp.status_code == 403


@pytest.mark.anyio
async def test_resume_requires_auth():
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.post("/api/risk/resume")
        assert resp.status_code == 403


@pytest.mark.anyio
async def test_update_parameters_requires_auth():
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as client:
        resp = await client.put(
            "/api/risk/parameters",
            json={"dailyLossLimit": "5"},
        )
        assert resp.status_code == 403
