import json
import logging
from datetime import datetime, timezone
from uuid import uuid4

import nats

logger = logging.getLogger(__name__)


class StrategyEventPublisher:
    def __init__(self, nats_url: str):
        self._nats_url = nats_url
        self._nc = None

    async def connect(self):
        self._nc = await nats.connect(self._nats_url)
        logger.info("connected to NATS for strategy events")

    async def close(self):
        if self._nc:
            await self._nc.close()

    async def publish_strategy_updated(self, strategy_id: str, name: str, status: str, action: str, parameters: dict):
        if not self._nc:
            logger.warning("NATS not connected, skipping event publish")
            return

        event = {
            "event_id": str(uuid4()),
            "event_type": "StrategyUpdated",
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "source": "strategy-manager",
            "payload": {
                "strategy_id": strategy_id,
                "name": name,
                "status": status,
                "action": action,
                "parameters": parameters,
            },
        }

        try:
            await self._nc.publish("pqap.strategy.updated", json.dumps(event).encode())
            logger.info("published StrategyUpdated event", extra={"strategy_id": strategy_id, "action": action})
        except Exception as e:
            logger.error("failed to publish StrategyUpdated event", exc_info=e)
