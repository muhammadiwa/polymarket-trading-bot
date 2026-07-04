from datetime import datetime, timezone
from unittest.mock import AsyncMock, patch

import pytest
from fastapi.testclient import TestClient

from app.models.notification import (
    CategoryConfig,
    ChannelConfig,
    NotificationHistoryItem,
    NotificationPreferencesResponse,
)


@pytest.fixture
def mock_jwt():
    with patch("app.middleware.auth.verify_jwt") as mock:
        mock.return_value = {"sub": "test-user"}
        yield mock


@pytest.fixture
def sample_preferences():
    return NotificationPreferencesResponse(
        categories=CategoryConfig(critical=True, warning=True, info=True, debug=False),
        channels=ChannelConfig(telegram=True, email=False, chat_id="123", email_to=""),
        updated_at=datetime.now(timezone.utc),
    )


@pytest.fixture
def sample_history_items():
    return [
        NotificationHistoryItem(
            id="test-id-1",
            category="warning",
            title="Test Warning",
            message="Warning body",
            channel="telegram",
            status="sent",
            sent_at=datetime.now(timezone.utc),
            created_at=datetime.now(timezone.utc),
        ),
    ]


class TestGetPreferences:
    def test_returns_preferences(self, mock_jwt, sample_preferences):
        with patch("app.routes.notifications.get_pool") as mock_pool:
            mock_conn = AsyncMock()
            mock_pool.return_value.acquire.return_value.__aenter__ = AsyncMock(return_value=mock_conn)
            mock_pool.return_value.acquire.return_value.__aexit__ = AsyncMock(return_value=False)

            with patch("app.repos.notification_repo.get_preferences", new_callable=AsyncMock) as mock_repo:
                mock_repo.return_value = sample_preferences
                from app.main import app
                client = TestClient(app)
                response = client.get("/api/notifications/preferences", headers={"Authorization": "Bearer test"})

                assert response.status_code == 200
                data = response.json()
                assert data["categories"]["critical"] is True
                assert data["channels"]["telegram"] is True


class TestUpdatePreferences:
    def test_updates_preferences(self, mock_jwt, sample_preferences):
        with patch("app.routes.notifications.get_pool") as mock_pool:
            mock_conn = AsyncMock()
            mock_pool.return_value.acquire.return_value.__aenter__ = AsyncMock(return_value=mock_conn)
            mock_pool.return_value.acquire.return_value.__aexit__ = AsyncMock(return_value=False)

            with patch("app.repos.notification_repo.update_preferences", new_callable=AsyncMock) as mock_repo:
                mock_repo.return_value = sample_preferences
                from app.main import app
                client = TestClient(app)
                response = client.put(
                    "/api/notifications/preferences",
                    json={"categories": {"debug": True}},
                    headers={"Authorization": "Bearer test"},
                )

                assert response.status_code == 200


class TestGetHistory:
    def test_returns_history(self, mock_jwt, sample_history_items):
        with patch("app.routes.notifications.get_pool") as mock_pool:
            mock_conn = AsyncMock()
            mock_pool.return_value.acquire.return_value.__aenter__ = AsyncMock(return_value=mock_conn)
            mock_pool.return_value.acquire.return_value.__aexit__ = AsyncMock(return_value=False)

            with patch("app.repos.notification_repo.get_history", new_callable=AsyncMock) as mock_repo:
                mock_repo.return_value = (sample_history_items, 1)
                from app.main import app
                client = TestClient(app)
                response = client.get("/api/notifications/history", headers={"Authorization": "Bearer test"})

                assert response.status_code == 200
                data = response.json()
                assert data["total"] == 1
                assert len(data["items"]) == 1

    def test_filters_by_category(self, mock_jwt):
        with patch("app.routes.notifications.get_pool") as mock_pool:
            mock_conn = AsyncMock()
            mock_pool.return_value.acquire.return_value.__aenter__ = AsyncMock(return_value=mock_conn)
            mock_pool.return_value.acquire.return_value.__aexit__ = AsyncMock(return_value=False)

            with patch("app.repos.notification_repo.get_history", new_callable=AsyncMock) as mock_repo:
                mock_repo.return_value = ([], 0)
                from app.main import app
                client = TestClient(app)
                response = client.get(
                    "/api/notifications/history?category=warning",
                    headers={"Authorization": "Bearer test"},
                )

                assert response.status_code == 200


class TestGetNotificationById:
    def test_returns_notification(self, mock_jwt, sample_history_items):
        with patch("app.routes.notifications.get_pool") as mock_pool:
            mock_conn = AsyncMock()
            mock_pool.return_value.acquire.return_value.__aenter__ = AsyncMock(return_value=mock_conn)
            mock_pool.return_value.acquire.return_value.__aexit__ = AsyncMock(return_value=False)

            with patch("app.repos.notification_repo.get_history_by_id", new_callable=AsyncMock) as mock_repo:
                mock_repo.return_value = sample_history_items[0]
                from app.main import app
                client = TestClient(app)
                response = client.get(
                    "/api/notifications/history/test-id-1",
                    headers={"Authorization": "Bearer test"},
                )

                assert response.status_code == 200
                data = response.json()
                assert data["id"] == "test-id-1"

    def test_returns_404(self, mock_jwt):
        with patch("app.routes.notifications.get_pool") as mock_pool:
            mock_conn = AsyncMock()
            mock_pool.return_value.acquire.return_value.__aenter__ = AsyncMock(return_value=mock_conn)
            mock_pool.return_value.acquire.return_value.__aexit__ = AsyncMock(return_value=False)

            with patch("app.repos.notification_repo.get_history_by_id", new_callable=AsyncMock) as mock_repo:
                mock_repo.return_value = None
                from app.main import app
                client = TestClient(app)
                response = client.get(
                    "/api/notifications/history/nonexistent",
                    headers={"Authorization": "Bearer test"},
                )

                assert response.status_code == 404
