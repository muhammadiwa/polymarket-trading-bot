import json
import logging
from typing import Optional

import nats
from nats.aio.client import Client as NATSClient

from app.config import config
from app.models.events import CapitalUpdated

logger = logging.getLogger(__name__)

_nats_client: Optional[NATSClient] = None


async def init_nats() -> None:
    """Initialize NATS connection."""
    global _nats_client
    try:
        _nats_client = await nats.connect(config.NATS_URL)
        logger.info("connected to NATS", extra={"url": config.NATS_URL})
    except Exception as e:
        logger.error("failed to connect to NATS", extra={"error": str(e)})
        _nats_client = None


async def close_nats() -> None:
    """Close NATS connection."""
    global _nats_client
    if _nats_client:
        await _nats_client.close()
        _nats_client = None


async def publish_capital_updated(event: CapitalUpdated) -> bool:
    """Publish CapitalUpdated event to NATS."""
    if not _nats_client:
        logger.warning("NATS not connected, cannot publish CapitalUpdated")
        return False

    try:
        data = json.dumps(event.to_dict()).encode()
        await _nats_client.publish("pqap.portfolio.capital_updated", data)
        logger.info("published CapitalUpdated event", extra={"event_id": event.event_id})
        return True
    except Exception as e:
        logger.error("failed to publish CapitalUpdated", extra={"error": str(e)})
        return False
