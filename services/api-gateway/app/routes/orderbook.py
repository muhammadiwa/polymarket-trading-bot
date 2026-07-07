import logging
import re
from typing import Optional

import httpx
from fastapi import APIRouter, Depends, HTTPException, Query

from app.config import config
from app.middleware.auth import verify_jwt

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/orderbook", tags=["orderbook"])

POLYMARKET_CLOB_URL = "https://clob.polymarket.com"
HTTP_TIMEOUT = 10.0
MARKET_ID_RE = re.compile(r"^[a-zA-Z0-9_\-]{1,128}$")

# Module-level client for connection pooling
_client = httpx.AsyncClient(timeout=HTTP_TIMEOUT, limits=httpx.Limits(max_connections=50))


def _validate_market_id(market_id: str) -> str:
    """#4: Validate market_id format to prevent SSRF and injection."""
    if not MARKET_ID_RE.match(market_id):
        raise HTTPException(status_code=400, detail="Invalid market_id format")
    return market_id


@router.get("/{market_id}")
async def get_orderbook(
    market_id: str,
    _user: dict = Depends(verify_jwt),
):
    """Fetch orderbook snapshot from Polymarket CLOB API."""
    _validate_market_id(market_id)

    try:
        resp = await _client.get(
            f"{POLYMARKET_CLOB_URL}/book",
            params={"token_id": market_id},
        )
        if resp.status_code != 200:
            raise HTTPException(status_code=resp.status_code, detail="Failed to fetch orderbook from Polymarket")

        data = resp.json()

        bids = []
        asks = []
        for level in data.get("bids", []):
            try:
                bids.append({
                    "price": str(level.get("price", "0")),
                    "size": str(level.get("size", "0")),
                })
            except (ValueError, TypeError):
                continue

        for level in data.get("asks", []):
            try:
                asks.append({
                    "price": str(level.get("price", "0")),
                    "size": str(level.get("size", "0")),
                })
            except (ValueError, TypeError):
                continue

        bids.sort(key=lambda x: float(x["price"]), reverse=True)
        asks.sort(key=lambda x: float(x["price"]))

        cum_bid = 0
        for b in bids:
            cum_bid += float(b["size"])
            b["cumulative"] = str(cum_bid)

        cum_ask = 0
        for a in asks:
            cum_ask += float(a["size"])
            a["cumulative"] = str(cum_ask)

        spread = "0"
        if bids and asks:
            spread = str(float(asks[0]["price"]) - float(bids[0]["price"]))

        return {
            "market_id": market_id,
            "bids": bids,
            "asks": asks,
            "spread": spread,
            "last_update": data.get("timestamp", ""),
        }

    except httpx.HTTPError as e:
        logger.error("failed to fetch orderbook", exc_info=e)
        raise HTTPException(status_code=502, detail="Failed to fetch orderbook from Polymarket")


@router.get("/{market_id}/trades")
async def get_recent_trades(
    market_id: str,
    limit: int = Query(100, ge=1, le=500),
    _user: dict = Depends(verify_jwt),
):
    """Fetch recent trades from Polymarket CLOB API."""
    _validate_market_id(market_id)

    try:
        resp = await _client.get(
            f"{POLYMARKET_CLOB_URL}/trades",
            params={"token_id": market_id, "limit": limit},
        )
        if resp.status_code != 200:
            raise HTTPException(status_code=resp.status_code, detail="Failed to fetch trades from Polymarket")

        data = resp.json()

        trades = []
        for t in data:
            try:
                trades.append({
                    "price": str(t.get("price", "0")),
                    "size": str(t.get("size", "0")),
                    "side": t.get("side", "BUY"),
                    "timestamp": t.get("timestamp", ""),
                })
            except (ValueError, TypeError):
                continue

        return {"market_id": market_id, "trades": trades, "count": len(trades)}

    except httpx.HTTPError as e:
        logger.error("failed to fetch trades", exc_info=e)
        raise HTTPException(status_code=502, detail="Failed to fetch trades from Polymarket")
