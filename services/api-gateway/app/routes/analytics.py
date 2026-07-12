import logging
import os
import httpx
from fastapi import APIRouter, Depends, HTTPException, Query, Request
from fastapi.responses import Response
from app.middleware.auth import verify_jwt

logger = logging.getLogger(__name__)
ANALYTICS_URL = os.getenv("ANALYTICS_SERVICE_URL", "http://localhost:8084")
router = APIRouter(prefix="/api/analytics", tags=["analytics"])
_client = None

async def _get_client():
    global _client
    if _client is None or _client.is_closed:
        _client = httpx.AsyncClient(timeout=httpx.Timeout(30.0, connect=5.0))
    return _client

async def close_client():
    global _client
    if _client is not None:
        await _client.aclose()
        _client = None

async def _proxy(request, path, user):
    client = await _get_client()
    token = request.cookies.get("pqap_session", "")
    auth_header = request.headers.get("authorization", "")
    headers = {
        "Content-Type": "application/json",
        "Authorization": auth_header if auth_header else f"Bearer {token}",
    }
    qs = str(request.url.query)
    try:
        url = f"{ANALYTICS_URL}/api/analytics/{path}"
        if qs:
            url = f"{url}?{qs}"
        resp = await client.get(url, headers=headers)
        return Response(content=resp.content, status_code=resp.status_code, media_type=resp.headers.get("content-type", "application/json"))
    except httpx.ConnectError:
        raise HTTPException(status_code=503, detail="Analytics service unavailable")
    except Exception:
        raise HTTPException(status_code=500, detail="Analytics proxy error")

@router.get("/pnl")
async def get_pnl(request: Request, start_date: str = Query(...), end_date: str = Query(...), group_by: str = Query("day"), strategy_id: str = Query(None), market_id: str = Query(None), user: dict = Depends(verify_jwt)):
    return await _proxy(request, "pnl", user)

@router.get("/metrics")
async def get_metrics(request: Request, start_date: str = Query(...), end_date: str = Query(...), strategy_id: str = Query(None), market_id: str = Query(None), user: dict = Depends(verify_jwt)):
    return await _proxy(request, "metrics", user)

@router.get("/risk")
async def get_risk(request: Request, start_date: str = Query(...), end_date: str = Query(...), strategy_id: str = Query(None), market_id: str = Query(None), user: dict = Depends(verify_jwt)):
    return await _proxy(request, "risk", user)

@router.get("/summary")
async def get_summary(request: Request, start_date: str = Query(...), end_date: str = Query(...), strategy_id: str = Query(None), market_id: str = Query(None), user: dict = Depends(verify_jwt)):
    return await _proxy(request, "summary", user)

@router.get("/histogram")
async def get_histogram(request: Request, start_date: str = Query(...), end_date: str = Query(...), strategy_id: str = Query(None), market_id: str = Query(None), bins: int = Query(20), user: dict = Depends(verify_jwt)):
    return await _proxy(request, "histogram", user)

@router.get("/anomalies")
async def get_anomalies(request: Request, severity: str = Query(None), limit: int = Query(50), user: dict = Depends(verify_jwt)):
    return await _proxy(request, "anomalies", user)

@router.get("/export")
async def export_csv(request: Request, start_date: str = Query(...), end_date: str = Query(...), strategy_id: str = Query(None), market_id: str = Query(None), side: str = Query(None), pnl_sign: str = Query(None), user: dict = Depends(verify_jwt)):
    return await _proxy(request, "export", user)

@router.get("/export/json")
async def export_json(request: Request, start_date: str = Query(...), end_date: str = Query(...), strategy_id: str = Query(None), market_id: str = Query(None), side: str = Query(None), pnl_sign: str = Query(None), user: dict = Depends(verify_jwt)):
    return await _proxy(request, "export/json", user)