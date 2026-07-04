import pytest
import os

os.environ["JWT_SECRET"] = "test-secret-key-for-testing-only"
os.environ["AUTH_CSRF_ENABLED"] = "true"

from fastapi import FastAPI, Request
from fastapi.testclient import TestClient
from starlette.responses import JSONResponse

from app.middleware.csrf import CSRFMiddleware


app = FastAPI()
app.add_middleware(CSRFMiddleware)


@app.get("/api/test")
async def test_get():
    return {"ok": True}


@app.post("/api/test")
async def test_post():
    return {"ok": True}


@app.put("/api/test")
async def test_put():
    return {"ok": True}


@app.delete("/api/test")
async def test_delete():
    return {"ok": True}


@app.get("/other")
async def test_other():
    return {"ok": True}


client = TestClient(app)


class TestCSRFMiddleware:
    def test_get_requests_skip_csrf(self):
        res = client.get("/api/test")
        assert res.status_code == 200

    def test_head_requests_skip_csrf(self):
        res = client.head("/api/test")
        assert res.status_code == 200

    def test_options_requests_skip_csrf(self):
        res = client.options("/api/test")
        assert res.status_code == 200

    def test_non_api_paths_skip_csrf(self):
        res = client.get("/other")
        assert res.status_code == 200

    def test_post_without_csrf_token_rejected(self):
        res = client.post("/api/test")
        assert res.status_code == 403
        assert "CSRF token missing" in res.json()["detail"]

    def test_post_with_mismatched_csrf_rejected(self):
        res = client.post(
            "/api/test",
            headers={"X-CSRF-Token": "token1"},
            cookies={"pqap_csrf": "token2"},
        )
        assert res.status_code == 403
        assert "CSRF token mismatch" in res.json()["detail"]

    def test_post_with_valid_csrf_accepted(self):
        res = client.post(
            "/api/test",
            headers={"X-CSRF-Token": "same-token"},
            cookies={"pqap_csrf": "same-token"},
        )
        assert res.status_code == 200

    def test_put_without_csrf_rejected(self):
        res = client.put("/api/test")
        assert res.status_code == 403

    def test_delete_without_csrf_rejected(self):
        res = client.delete("/api/test")
        assert res.status_code == 403

    def test_put_with_valid_csrf_accepted(self):
        res = client.put(
            "/api/test",
            headers={"X-CSRF-Token": "valid"},
            cookies={"pqap_csrf": "valid"},
        )
        assert res.status_code == 200

    def test_delete_with_valid_csrf_accepted(self):
        res = client.delete(
            "/api/test",
            headers={"X-CSRF-Token": "valid"},
            cookies={"pqap_csrf": "valid"},
        )
        assert res.status_code == 200
