import logging
from collections.abc import Awaitable, Callable
from typing import Any

import nats
import nats.js
import orjson
from cachetools import TTLCache

from app.models import NotificationRequest
from ports.event_port import EventPort

logger = logging.getLogger(__name__)


class NATSSubscriber(EventPort):
    def __init__(self, nats_url: str):
        self._nats_url = nats_url
        self._nc: nats.NATS | None = None
        self._js: nats.js.JetStreamContext | None = None
        self._sub: nats.aio.subscription.Subscription | None = None
        self._seen_ids: TTLCache[str, bool] = TTLCache(maxsize=10000, ttl=300)

    async def connect(self) -> None:
        self._nc = await nats.connect(self._nats_url, connect_timeout=10)
        self._js = self._nc.jetstream()
        logger.info("Connected to NATS JetStream at %s", self._nats_url)

    async def subscribe(
        self,
        subject: str,
        handler: Callable[[NotificationRequest], Awaitable[None]],
    ) -> None:
        if self._nc is None or self._js is None:
            raise RuntimeError("NATS not connected")

        async def _on_message(msg: nats.aio.msg.Msg) -> None:
            try:
                data = orjson.loads(msg.data)
                event_id = data.get("event_id", "")

                if not event_id:
                    logger.warning("Rejecting message with missing/empty event_id")
                    await msg.term()
                    return

                if event_id in self._seen_ids:
                    logger.debug("Duplicate event_id=%s, skipping", event_id)
                    await msg.ack()
                    return
                self._seen_ids[event_id] = True

                request = NotificationRequest.model_validate(data)
                await handler(request)
                await msg.ack()
            except Exception:
                logger.exception("Error processing NATS message")
                await msg.term()

        self._sub = await self._js.subscribe(subject, cb=_on_message, durable="notification", manual_ack=True)
        logger.info("Subscribed to JetStream subject=%s", subject)

    async def publish(self, subject: str, data: dict[str, Any]) -> None:
        if self._nc is None:
            raise RuntimeError("NATS not connected")
        payload = orjson.dumps(data)
        await self._nc.publish(subject, payload)
        logger.debug("Published to subject=%s", subject)

    async def close(self) -> None:
        if self._sub:
            await self._sub.unsubscribe()
        if self._nc:
            await self._nc.drain()
            self._nc = None
            logger.info("NATS connection closed")
