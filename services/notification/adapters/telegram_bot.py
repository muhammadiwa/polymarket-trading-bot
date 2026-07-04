import asyncio
import logging
import time

from telegram import Bot
from telegram.error import RetryAfter, TelegramError

from app.formatter import format_message
from app.metrics import (
    NOTIFICATION_DELIVERY_LATENCY,
    NOTIFICATION_FAILED_TOTAL,
    NOTIFICATION_SENT_TOTAL,
)
from app.models import NotificationRecord
from ports.notify_port import NotifyPort

logger = logging.getLogger(__name__)

MAX_RETRIES = 3
BACKOFF_BASE = 1.0
SEND_TIMEOUT = 5.0
MAX_MESSAGE_LENGTH = 4000

# Circuit breaker state
_CIRCUIT_FAIL_THRESHOLD = 5
_CIRCUIT_RECOVERY_TIMEOUT = 30.0


class TelegramNotifier(NotifyPort):
    def __init__(self, bot_token: str, chat_id: str):
        self._bot = Bot(token=bot_token)
        self._chat_id = chat_id
        self._circuit_failures: int = 0
        self._circuit_open_until: float = 0.0

    def _is_circuit_open(self) -> bool:
        if self._circuit_failures >= _CIRCUIT_FAIL_THRESHOLD:
            if time.monotonic() < self._circuit_open_until:
                return True
            self._circuit_failures = 0
        return False

    def _record_circuit_failure(self) -> None:
        self._circuit_failures += 1
        if self._circuit_failures >= _CIRCUIT_FAIL_THRESHOLD:
            self._circuit_open_until = time.monotonic() + _CIRCUIT_RECOVERY_TIMEOUT
            logger.warning(
                "Circuit breaker opened for Telegram API, recovery in %.0fs",
                _CIRCUIT_RECOVERY_TIMEOUT,
            )

    def _record_circuit_success(self) -> None:
        self._circuit_failures = 0

    async def send(self, record: NotificationRecord) -> bool:
        if self._is_circuit_open():
            logger.warning("Circuit breaker open, skipping Telegram send")
            NOTIFICATION_FAILED_TOTAL.labels(
                channel="telegram", reason="circuit_open"
            ).inc()
            return False

        text = format_message(record)
        if len(text) > MAX_MESSAGE_LENGTH:
            text = text[:MAX_MESSAGE_LENGTH - 20] + "\n\n... (truncated)"

        text = f"<b>{record.title}</b>\n\n{text}"
        start = time.monotonic()

        for attempt in range(1, MAX_RETRIES + 1):
            try:
                await asyncio.wait_for(
                    self._bot.send_message(
                        chat_id=self._chat_id,
                        text=text,
                        parse_mode="HTML",
                    ),
                    timeout=SEND_TIMEOUT,
                )
                elapsed = time.monotonic() - start
                NOTIFICATION_SENT_TOTAL.labels(
                    channel="telegram", severity=record.severity.value
                ).inc()
                NOTIFICATION_DELIVERY_LATENCY.labels(channel="telegram").observe(
                    elapsed
                )
                self._record_circuit_success()
                logger.info(
                    "Notification sent: event_type=%s severity=%s latency=%.3fs",
                    record.event_type,
                    record.severity.value,
                    elapsed,
                )
                return True
            except RetryAfter as exc:
                retry_after = exc.retry_after + 1
                logger.warning(
                    "Telegram rate limited, sleeping %ds", retry_after
                )
                await asyncio.sleep(retry_after)
            except TelegramError as exc:
                logger.warning(
                    "Telegram send failed attempt=%d/%d: %s",
                    attempt,
                    MAX_RETRIES,
                    exc,
                )
                if attempt < MAX_RETRIES:
                    await asyncio.sleep(BACKOFF_BASE * (2 ** (attempt - 1)))

        self._record_circuit_failure()
        NOTIFICATION_FAILED_TOTAL.labels(
            channel="telegram", reason="api_error"
        ).inc()
        logger.error(
            "Telegram delivery failed after %d retries: event_type=%s",
            MAX_RETRIES,
            record.event_type,
        )
        return False
