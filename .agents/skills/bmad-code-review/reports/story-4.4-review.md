# Story 4.4 Review: Trade History Filtering & Export

**Date:** 2026-07-07
**Scope:** `analytics_repo.py:26-65`, `analytics.py:148-240`, `page.tsx`, `api.ts:168-218`

---

## FINDINGS

### 1. CSV Injection via Unescaped Numeric Fields

**Severity:** HIGH
**Category:** Injection
**File:** `analytics.py:180-187`

**Evidence:**
```python
str(t.get("price", "")),        # NOT escaped
str(t.get("quantity", "")),      # NOT escaped
str(t.get("filled_quantity", "")), # NOT escaped
str(t.get("pnl", "")),           # NOT escaped
str(t.get("fee", "")),           # NOT escaped
str(t.get("slippage_pct", "")),  # NOT escaped
```

Numeric fields pass through `str()` but skip `_csv_escape()`. If any Decimal value is crafted or corrupted to contain `=`, `+`, `-`, or `@` as a prefix (e.g., from a DB compromise or data corruption), Excel/Sheets will interpret it as a formula.

**Impact:** A malicious or corrupted `pnl` value like `=CMD('calc')` executes arbitrary commands when the CSV is opened in Excel. Low likelihood on clean data, but a defense-in-depth failure.

**Suggestion:** Apply `_csv_escape()` to ALL fields, not just string ones. Or prefix numeric fields with a tab/space to neutralize formula injection:
```python
def _csv_safe(value: str) -> str:
    if value and value and value[0] in ("=", "+", "-", "@"):
        return "\t" + value
    return _csv_escape(value)
```

---

### 2. No Date Range Bound Enforcement (DoS via Massive Export)

**Severity:** HIGH
**Category:** Denial of Service
**File:** `analytics.py:161-162`

**Evidence:**
```python
if start >= end:
    raise HTTPException(status_code=400, detail="start_date must be before end_date")
```

Only validates `start < end`. A request with `start_date=2000-01-01` and `end_date=2099-12-31` loads the entire trades table into memory before streaming. For the JSON endpoint (`/export/json`), all rows are materialized in `result = []` at line 218.

**Impact:** Single authenticated request can OOM the analytics service or cause sustained DB lock contention. The JSON endpoint is worse — it holds all rows in a list before returning.

**Suggestion:**
```python
MAX_RANGE_DAYS = 365
if (end - start).days > MAX_RANGE_DAYS:
    raise HTTPException(status_code=400, detail=f"Date range exceeds {MAX_RANGE_DAYS} days")
```

---

### 3. JSON Export Materializes Entire Result Set in Memory

**Severity:** MEDIUM
**Category:** Resource Exhaustion
**File:** `analytics.py:218-236`

**Evidence:**
```python
result = []
for t in trades:
    result.append({...})
return {"trades": result, "count": len(result)}
```

Unlike the CSV endpoint which streams, the JSON endpoint loads all rows into a Python list. For a large date range, this doubles memory usage (rows from DB + serialized JSON).

**Impact:** OOM under moderate load with large exports.

**Suggestion:** Stream JSON using `StreamingResponse` with `ijson` or manual JSON array streaming, or impose a hard row limit (e.g., 10,000) and paginate.

---

### 4. `side` Parameter Lacks Validation at Repo Layer

**Severity:** MEDIUM
**Category:** Input Validation
**File:** `analytics_repo.py:48-51`

**Evidence:**
```python
if side:
    conditions.append(f"side = ${idx}")
    params.append(side)
    idx += 1
```

The route layer validates `side` with `pattern="^(YES|NO)$"`, but the repo function `get_trades_in_range` accepts any string. If this repo function is called from another path (e.g., a Celery task, CLI, or future endpoint) without the FastAPI validator, arbitrary values reach the DB.

**Impact:** Defense-in-depth violation. Current exploitability: none (parameterized). Future risk: real if repo is reused.

**Suggestion:** Add a guard in the repo function itself:
```python
if side and side not in ("YES", "NO"):
    raise ValueError(f"Invalid side: {side}")
```

---

### 5. `strategy_id` and `market_id` Unvalidated — Potential Exfiltration Vector

**Severity:** MEDIUM
**Category:** Authorization / Input Validation
**File:** `analytics.py:152-153`

**Evidence:**
```python
strategy_id: Optional[str] = Query(None),
market_id: Optional[str] = Query(None),
```

No length limit, no pattern constraint, no sanitization. These are passed directly to the SQL query. While parameterized (safe from SQL injection), there's no authorization check that the requesting user should have access to the specified strategy/market. Any authenticated user can enumerate trades across all strategies.

**Impact:** Horizontal privilege escalation — user A can export user B's trades by guessing strategy/market IDs.

**Suggestion:** Add ownership validation or scope the query to the authenticated user's strategies:
```python
# In the query, add:
conditions.append(f"user_id = ${idx}")
params.append(_user["sub"])
```

---

### 6. No Rate Limiting on Export Endpoints

**Severity:** MEDIUM
**Category:** Abuse / DoS
**File:** `analytics.py:148, 198`

**Evidence:**
No rate limiter decorator or middleware on `/export` or `/export/json`. Both are authenticated but have no throttle.

**Impact:** Automated scripts can hammer export endpoints, saturating DB connections and CPU.

**Suggestion:** Apply rate limiting (e.g., `slowapi` or middleware):
```python
@router.get("/export")
@limiter.limit("5/minute")
async def export_trades(...):
```

---

### 7. `_csv_escape` Doesn't Handle `None` Properly

**Severity:** LOW
**Category:** Bug
**File:** `analytics.py:241`

**Evidence:**
```python
if not value or value == "None":
    return ""
```

The caller passes `str(t.get("market_slug", ""))`. If `market_slug` is `None`, `str(None)` produces `"None"`, which is caught. But if the value is `0`, `str(0)` = `"0"`, which is truthy — correct. However, `not value` catches empty strings AND `None`, but the `"None"` check is a string comparison for the literal string "None". This is fragile — if a field legitimately contains "None" as data, it's silently erased.

**Impact:** Data corruption on edge case values.

**Suggestion:** Handle `None` before calling `_csv_escape`:
```python
def _csv_escape(value: Optional[str]) -> str:
    if value is None:
        return ""
    ...
```

---

### 8. Blob URL Memory Leak on Download Failure

**Severity:** LOW
**Category:** Resource Leak
**File:** `api.ts:190-196`

**Evidence:**
```typescript
const blob = await res.blob();
const url = URL.createObjectURL(blob);
const a = document.createElement("a");
a.href = url;
a.download = `trades_${startDate}_${endDate}.csv`;
a.click();
URL.revokeObjectURL(url);
```

If `res.blob()` throws (e.g., network interruption mid-stream), `URL.revokeObjectURL` is never called. Also, `a.click()` is synchronous but the download is async — if the tab closes immediately after, the blob may not fully write.

**Impact:** Minor memory leak in the browser on repeated failures.

**Suggestion:** Use try/finally:
```typescript
const url = URL.createObjectURL(blob);
try {
  const a = document.createElement("a");
  a.href = url;
  a.download = `trades_${startDate}_${endDate}.csv`;
  a.click();
} finally {
  URL.revokeObjectURL(url);
}
```

---

### 9. `exporting` State Allows Concurrent Exports

**Severity:** LOW
**Category:** Race Condition
**File:** `page.tsx:23-43`

**Evidence:**
```typescript
const handleExportCSV = async () => {
    setExporting(true);
    try { await downloadCSV(...); }
    finally { setExporting(false); }
};
```

`setExporting` is async (React state update). If a user double-clicks the CSV button rapidly, both handlers fire before the first `setExporting(true)` takes effect in the render cycle. The `disabled={exporting}` guard only prevents clicks between renders.

**Impact:** Duplicate downloads, double server load.

**Suggestion:** Use a ref for synchronous guard:
```typescript
const exportingRef = useRef(false);
const handleExportCSV = async () => {
    if (exportingRef.current) return;
    exportingRef.current = true;
    setExporting(true);
    try { ... }
    finally { exportingRef.current = false; setExporting(false); }
};
```

---

## SUMMARY

| # | Severity | Title |
|---|----------|-------|
| 1 | HIGH | CSV Injection via Unescaped Numeric Fields |
| 2 | HIGH | No Date Range Bound Enforcement (DoS) |
| 3 | MEDIUM | JSON Export Materializes Entire Result Set |
| 4 | MEDIUM | `side` Parameter Lacks Validation at Repo Layer |
| 5 | MEDIUM | No Ownership Check on strategy/market ID |
| 6 | MEDIUM | No Rate Limiting on Export Endpoints |
| 7 | LOW | `_csv_escape` Fragile None Handling |
| 8 | LOW | Blob URL Memory Leak on Failure |
| 9 | LOW | Double-Click Race on Export Button |

**Verdict:** 2 HIGHs require fix before merge. Findings 1 and 2 are production-impacting. Finding 5 (no ownership scoping) is the most architecturally concerning — it assumes all authenticated users see all data.
