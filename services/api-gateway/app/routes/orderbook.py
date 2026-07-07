import logging
from typing import Optional

import httpx
from fastapi import APIRouter, Depends, HTTPException, Query

from app.config import config
from app.middleware.auth import verify_jwt

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/orderbook", tags=["orderbook"])

POLYMARKET_CLOB_URL = "https://clob.polymarket.com"
HTTP_TIMEOUT = 10.0


@router.get("/{market_id}")
async def get_orderbook(
    market_id: str,
    _user: dict = Depends(verify_jwt),
):
    """Fetch orderbook snapshot from Polymarket CLOB API."""
    try:
        async with httpx.AsyncClient(timeout=HTTP_TIMEOUT) as client:
            resp = await client.get(
                f"{POLYMARKET_CLOB_URL}/book",
                params={"token_id": market_id},
            )
            if resp.status_code != 200:
                raise HTTPException(status_code=resp.status_code, detail="Failed to fetch orderbook from Polymarket")

            data = resp.json()

            # Transform to our format
            bids = []
            asks = []
            for level in data.get("bids", []):
                bids.append({
                    "price": str(level.get("price", "0")),
                    "size": str(level.get("size", "0")),
                })
            for level in data.get("asks", []):
                asks.append({
                    "price": str(level.get("price", "0")),
                    "size": str(level.get("size", "0")),
                })

            # Sort and calculate cumulative
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
    try:
        async with httpx.AsyncClient(timeout=HTTP_TIMEOUT) as client:
            resp = await client.get(
                f"{POLYMARKET_CLOB_URL}/trades",
                params={"token_id": market_id, "limit": limit},
            )
            if resp.status_code != 200:
                raise HTTPException(status_code=resp.status_code, detail="Failed to fetch trades from Polymarket")

            data = resp.json()

            trades = []
            for t in data:
                trades.append({
                    "price": str(t.get("price", "0")),
                    "size": str(t.get("size", "0")),
                    "side": t.get("side", "BUY"),
                    "timestamp": t.get("timestamp", ""),
                })

            return {"market_id": market_id, "trades": trades, "count": len(trades)}

    except httpx.HTTPError as e:
        logger.error("failed to fetch trades", exc_info=e)
        raise HTTPException(status_code=502, detail="Failed to fetch trades from Polymarket")
