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
from app.metrics import NOTIFICATION_QUEUE_SIZE, NOTIFICATION_THROTTLED_TOTAL
from app.models import (
    Channel,
    NotificationPreferences,
    NotificationRecord,
    NotificationRequest,
    NotificationStatus,
)
from app.throttler import Throttler

logging.basicConfig(
    level=getattr(logging, config.LOG_LEVEL.upper(), logging.INFO),
    format='{"level":"%(levelname)s","service":"notification","msg":"%(message)s"}',
)
logger = logging.getLogger(__name__)


class NotificationService:
    def __init__(self) -> None:
        self._preferences = NotificationPreferences(
            critical_enabled=config.CRITICAL_ENABLED,
            warning_enabled=config.WARNING_ENABLED,
            info_enabled=config.INFO_ENABLED,
            debug_enabled=config.DEBUG_ENABLED,
        )
        self._throttler = Throttler(max_per_minute=config.NOTIFICATION_MAX_PER_MINUTE)
        self._nats = NATSSubscriber(config.NATS_URL)
        self._telegram = TelegramNotifier(config.TELEGRAM_BOT_TOKEN, config.TELEGRAM_CHAT_ID)
        self._postgres = PostgresRepo(config.DATABASE_URL)
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
        await self._postgres.init_preferences_table()
        await self._load_preferences()
        await self._nats.connect()
        await self._nats.subscribe(config.NATS_SUBJECT, self._handle_notification)
        self._throttle_drain_task = asyncio.create_task(self._drain_throttle_queue())

        logger.info("Notification service started")
        self._setup_signal_handlers()

    async def _load_preferences(self) -> None:
        prefs = await self._postgres.load_preferences()
        if prefs is not None:
            self._preferences = prefs
            logger.info("Loaded notification preferences from database")

    async def _save_preferences(self) -> None:
        await self._postgres.save_preferences(self._preferences)

    def _setup_signal_handlers(self) -> None:
        if platform.system() == "Windows":
            logger.info("Skipping signal handlers on Windows")
            return
        loop = asyncio.get_running_loop()
        for sig in (signal.SIGINT, signal.SIGTERM):
            loop.add_signal_handler(sig, self._shutdown_event.set)

    async def _drain_throttle_queue(self) -> None:
        while not self._shutdown_event.is_set():
            try:
                record = await asyncio.wait_for(
                    self._throttle_queue.get(), timeout=5.0
                )
            except asyncio.TimeoutError:
                NOTIFICATION_QUEUE_SIZE.set(self._throttle_queue.qsize())
                continue
            delivered = await self._telegram.send(record)
            if delivered:
                record.delivered_at = datetime.now(timezone.utc)
            await self._history.record_delivery(record, delivered)
            NOTIFICATION_QUEUE_SIZE.set(self._throttle_queue.qsize())

    async def _handle_notification(self, request: NotificationRequest) -> None:
        severity = classify(request.event_type, request.payload.severity, request.payload.priority)

        if not self._preferences.is_enabled(severity):
            logger.debug(
                "Notification disabled by preferences: event_type=%s severity=%s",
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
            status=NotificationStatus.DELIVERED,
            metadata=request.payload.metadata,
        )

        if not await self._throttler.should_allow(severity):
            NOTIFICATION_THROTTLED_TOTAL.labels(severity=severity.value).inc()
            logger.info(
                "Notification throttled: event_type=%s severity=%s",
                request.event_type,
                severity.value,
            )
            await self._history.record_throttled(record)
            await self._throttle_queue.put(record)
            NOTIFICATION_QUEUE_SIZE.set(self._throttle_queue.qsize())
            return

        delivered = await self._telegram.send(record)
        if delivered:
            record.delivered_at = datetime.now(timezone.utc)
        await self._history.record_delivery(record, delivered)

    async def run(self) -> None:
        await self.start()
        await self._shutdown_event.wait()
        await self.shutdown()

    async def shutdown(self) -> None:
        logger.info("Shutting down notification service")
        if self._throttle_drain_task:
            self._throttle_drain_task.cancel()
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
    # TODO(#17): Add circuit breaker monitoring for Redis/PostgreSQL connections.
    # Currently health checks only verify connection existence, not responsiveness.
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
