import asyncio
import logging
import platform
import signal
import sys
from datetime import datetime, timezone

from aiohttp import web
from prometheus_client import generate_latest

from adapters.nats_subscriber import NATSSubscriber
from adapters.postgres_repo import PostgresRepo
from adapters.telegram_bot import TelegramNotifier
from app.categorizer import classify
from app.config import config
from app.history import HistoryManager
from app.metrics import (
    NOTIFICATION_QUEUE_SIZE,
    NOTIFICATION_SUPPRESSED_TOTAL,
    NOTIFICATION_THROTTLED_TOTAL,
)
from app.models import (
    Channel,
    NotificationPreferences,
    NotificationRecord,
    NotificationRequest,
    NotificationStatus,
)
from app.preferences import PreferencesManager
from app.throttler import Throttler

logging.basicConfig(
    level=getattr(logging, config.LOG_LEVEL.upper(), logging.INFO),
    format='{"level":"%(levelname)s","service":"notification","msg":"%(message)s"}',
)
logger = logging.getLogger(__name__)


class NotificationService:
    """Notification delivery service.

    Supported channels:
      - telegram: implemented via TelegramNotifier
      - email:    DEFERRED — no EmailNotifier adapter yet; requests routed to
                  email will fall through to telegram as a stop-gap.
    """

    def __init__(self) -> None:
        self._postgres = PostgresRepo(config.DATABASE_URL)
        self._preferences = PreferencesManager(self._postgres)
        self._throttler = Throttler(
            max_per_minute=config.NOTIFICATION_MAX_PER_MINUTE,
            redis_url=config.REDIS_URL,
        )
        self._nats = NATSSubscriber(config.NATS_URL)
        self._telegram = TelegramNotifier(config.TELEGRAM_BOT_TOKEN, config.TELEGRAM_CHAT_ID)
        self._history = HistoryManager(self._postgres, retention_limit=config.NOTIFICATION_HISTORY_LIMIT)
        self._shutdown_event = asyncio.Event()
        self._throttle_queue: asyncio.Queue[NotificationRecord] = asyncio.Queue()
        self._throttle_drain_task: asyncio.Task[None] | None = None

    async def start(self) -> None:
        if not config.TELEGRAM_BOT_TOKEN:
            logger.error("TELEGRAM_BOT_TOKEN is required")
            sys.exit(1)
        if not config.TELEGRAM_CHAT_ID:
            logger.error("TELEGRAM_CHAT_ID is required")
            sys.exit(1)
        if not config.DATABASE_URL:
            logger.error("DATABASE_URL is required")
            sys.exit(1)

        await self._postgres.connect()
        await self._postgres.init_tables()
        await self._preferences.load()
        await self._throttler.connect()
        await self._nats.connect()
        await self._nats.subscribe(config.NATS_SUBJECT, self._handle_notification)
        self._throttle_drain_task = asyncio.create_task(self._drain_throttle_queue())

        logger.info("Notification service started")
        self._setup_signal_handlers()

    async def _setup_signal_handlers(self) -> None:
        if platform.system() == "Windows":
            logger.info("Skipping POSIX signal handlers on Windows; using asyncio task watchdog for shutdown")
            # #7: On Windows, monitor from a background task that checks a file-based
            # shutdown flag or responds to Ctrl+C via the default handler.
            asyncio.create_task(self._windows_shutdown_watchdog())
            return
        loop = asyncio.get_running_loop()
        for sig in (signal.SIGINT, signal.SIGTERM):
            loop.add_signal_handler(sig, self._shutdown_event.set)

    async def _windows_shutdown_watchdog(self) -> None:
        """Windows-compatible shutdown: polls for KeyboardInterrupt via shared event."""
        # #7: On Windows, signal handlers don't work with asyncio.
        # The default Python Ctrl+C handler will raise KeyboardInterrupt
        # which will propagate to the main asyncio.run() caller.
        # This watchdog allows cooperative shutdown if run_service is
        # cancelled externally.
        try:
            while not self._shutdown_event.is_set():
                await asyncio.sleep(1.0)
        except asyncio.CancelledError:
            self._shutdown_event.set()

    async def _drain_throttle_queue(self) -> None:
        while not self._shutdown_event.is_set():
            try:
                record = await asyncio.wait_for(
                    self._throttle_queue.get(), timeout=5.0
                )
            except asyncio.TimeoutError:
                NOTIFICATION_QUEUE_SIZE.set(self._throttle_queue.qsize())
                continue
            try:
                delivered = await self._telegram.send(record)
                if delivered:
                    record.delivered_at = datetime.now(timezone.utc)
                await self._history.record_delivery(record, delivered)
            except Exception:
                logger.exception(
                    "Error draining throttle queue for event_type=%s",
                    record.event_type,
                )
            NOTIFICATION_QUEUE_SIZE.set(self._throttle_queue.qsize())

    async def _handle_notification(self, request: NotificationRequest) -> None:
        severity = classify(request.event_type, request.payload.severity, request.payload.priority)

        if not self._preferences.is_enabled(severity):
            NOTIFICATION_SUPPRESSED_TOTAL.labels(severity=severity.value).inc()
            record = NotificationRecord(
                event_type=request.event_type,
                severity=severity,
                title=request.payload.title,
                message=request.payload.message,
                channel=Channel.TELEGRAM,
                status=NotificationStatus.SUPPRESSED,
                metadata=request.payload.metadata,
            )
            await self._history.record_suppressed(record)
            logger.info(
                "Notification suppressed by preferences: event_type=%s severity=%s",
                request.event_type,
                severity.value,
            )
            return

        record = NotificationRecord(
            event_type=request.event_type,
            severity=severity,
            title=request.payload.title,
            message=request.payload.message,
            channel=Channel.TELEGRAM,
            status=NotificationStatus.SENT,
            metadata=request.payload.metadata,
        )

        if not await self._throttler.should_allow(severity):
            NOTIFICATION_THROTTLED_TOTAL.labels(severity=severity.value).inc()
            record.status = NotificationStatus.THROTTLED
            await self._history.record_throttled(record)
            await self._publish_event("pqap.notification.throttled", record)
            logger.info(
                "Notification throttled: event_type=%s severity=%s",
                request.event_type,
                severity.value,
            )
            return

        delivered = await self._telegram.send(record)
        if delivered:
            record.delivered_at = datetime.now(timezone.utc)
        await self._history.record_delivery(record, delivered)
        await self._publish_event("pqap.notification.sent", record)

    async def _publish_event(self, subject: str, record: NotificationRecord) -> None:
        try:
            await self._nats.publish(subject, {
                "notification_id": str(record.id),
                "event_type": record.event_type,
                "severity": record.severity.value,
                "status": record.status.value,
                "channel": record.channel.value,
                "timestamp": datetime.now(timezone.utc).isoformat(),
            })
        except Exception:
            logger.exception("Failed to publish event to %s", subject)

    @property
    def preferences_manager(self) -> PreferencesManager:
        return self._preferences

    @property
    def history_manager(self) -> HistoryManager:
        return self._history

    async def run(self) -> None:
        await self.start()
        await self._shutdown_event.wait()
        await self.shutdown()

    async def shutdown(self) -> None:
        logger.info("Shutting down notification service")
        if self._throttle_drain_task:
            self._throttle_drain_task.cancel()
        await self._throttler.close()
        await self._nats.close()
        await self._postgres.close()
        logger.info("Notification service stopped")


async def metrics_handler(request: web.Request) -> web.Response:
    return web.Response(
        body=generate_latest(),
        content_type="text/plain",
    )


async def health_handler(request: web.Request) -> web.Response:
    service: NotificationService = request.app["service"]
    checks = {
        "nats": service._nats._nc is not None and not service._nats._nc.is_closed,
        "postgres": service._postgres._pool is not None,
    }
    # #9: Add actual connectivity checks, not just object existence
    if checks["nats"] and service._nats._nc is not None:
        try:
            await service._nats._nc.flush(timeout=1.0)
        except Exception:
            checks["nats"] = False
    if checks["postgres"] and service._postgres._pool is not None:
        try:
            async with service._postgres._pool.acquire() as conn:
                await conn.execute("SELECT 1")
        except Exception:
            checks["postgres"] = False
    all_ok = all(checks.values())
    return web.json_response(
        {"status": "ok" if all_ok else "degraded", "checks": checks},
        status=200 if all_ok else 503,
    )


async def run_service() -> None:
    service = NotificationService()

    app = web.Application()
    app["service"] = service
    app.router.add_get("/metrics", metrics_handler)
    app.router.add_get("/health", health_handler)

    runner = web.AppRunner(app)
    await runner.setup()
    site = web.TCPSite(runner, "127.0.0.1", config.METRICS_PORT)
    await site.start()
    logger.info("Metrics server started on port %d", config.METRICS_PORT)

    try:
        await service.run()
    finally:
        await runner.cleanup()


def main() -> None:
    asyncio.run(run_service())


if __name__ == "__main__":
    main()
