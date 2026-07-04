import pytest
from datetime import datetime, timezone
from unittest.mock import AsyncMock

from app.history import HistoryManager
from app.models import (
    Channel,
    NotificationRecord,
    NotificationStatus,
    Severity,
)


def _make_record(severity: Severity = Severity.INFO) -> NotificationRecord:
    return NotificationRecord(
        event_type="TestEvent",
        severity=severity,
        title="Test Title",
        message="Test message",
        channel=Channel.TELEGRAM,
        status=NotificationStatus.SENT,
    )


class TestHistoryManager:
    @pytest.mark.asyncio
    async def test_record_delivery_sets_sent_status(self):
        mock_storage = AsyncMock()
        manager = HistoryManager(mock_storage, retention_limit=100)
        record = _make_record()

        await manager.record_delivery(record, delivered=True)

        assert record.status == NotificationStatus.SENT
        mock_storage.save.assert_called_once_with(record)

    @pytest.mark.asyncio
    async def test_record_delivery_sets_failed_status(self):
        mock_storage = AsyncMock()
        manager = HistoryManager(mock_storage, retention_limit=100)
        record = _make_record()

        await manager.record_delivery(record, delivered=False)

        assert record.status == NotificationStatus.FAILED
        mock_storage.save.assert_called_once_with(record)

    @pytest.mark.asyncio
    async def test_record_throttled(self):
        mock_storage = AsyncMock()
        manager = HistoryManager(mock_storage, retention_limit=100)
        record = _make_record()

        await manager.record_throttled(record)

        assert record.status == NotificationStatus.THROTTLED
        mock_storage.save.assert_called_once_with(record)

    @pytest.mark.asyncio
    async def test_record_suppressed(self):
        mock_storage = AsyncMock()
        manager = HistoryManager(mock_storage, retention_limit=100)
        record = _make_record()

        await manager.record_suppressed(record)

        assert record.status == NotificationStatus.SUPPRESSED
        mock_storage.save.assert_called_once_with(record)

    @pytest.mark.asyncio
    async def test_get_history_delegates_to_storage(self):
        mock_storage = AsyncMock()
        mock_storage.get_history.return_value = [_make_record()]
        manager = HistoryManager(mock_storage, retention_limit=100)

        result = await manager.get_history(
            limit=10, offset=0, category="warning", status="sent"
        )

        assert len(result) == 1
        mock_storage.get_history.assert_called_once_with(
            limit=10, offset=0, category="warning", status="sent",
            start_date=None, end_date=None,
        )

    @pytest.mark.asyncio
    async def test_get_by_id_delegates_to_storage(self):
        mock_storage = AsyncMock()
        record = _make_record()
        mock_storage.get_notification_by_id.return_value = record
        manager = HistoryManager(mock_storage, retention_limit=100)

        result = await manager.get_by_id(str(record.id))

        assert result == record
        mock_storage.get_notification_by_id.assert_called_once_with(str(record.id))

    @pytest.mark.asyncio
    async def test_get_by_id_returns_none(self):
        mock_storage = AsyncMock()
        mock_storage.get_notification_by_id.return_value = None
        manager = HistoryManager(mock_storage, retention_limit=100)

        result = await manager.get_by_id("nonexistent")

        assert result is None
