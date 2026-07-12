# PQAP Service Starter
# Usage: .\start-services.ps1
# Stops all existing services, clears cache, and starts everything fresh.

$ErrorActionPreference = "SilentlyContinue"
$root = $PSScriptRoot

Write-Host "=== PQAP Service Starter ===" -ForegroundColor Cyan

# ── 1. Kill existing processes ──────────────────────────────────────────────
Write-Host "`n[1/5] Stopping existing processes..." -ForegroundColor Yellow

Get-Process -Name "node" -ErrorAction SilentlyContinue | Stop-Process -Force
$apiPid = (netstat -ano | Select-String ":8080 " | Select-String "LISTENING" | Select-Object -First 1) -split "\s+" | Select-Object -Last 1
if ($apiPid) { Stop-Process -Id $apiPid -Force -ErrorAction SilentlyContinue }
$dashPid = (netstat -ano | Select-String ":3000 " | Select-String "LISTENING" | Select-Object -First 1) -split "\s+" | Select-Object -Last 1
if ($dashPid) { Stop-Process -Id $dashPid -Force -ErrorAction SilentlyContinue }
Start-Sleep -Seconds 3
Write-Host "  Done." -ForegroundColor Green

# ── 2. Clear Next.js cache ─────────────────────────────────────────────────
Write-Host "`n[2/5] Clearing Next.js cache..." -ForegroundColor Yellow
Remove-Item -Recurse -Force "$root\services\dashboard\.next" -ErrorAction SilentlyContinue
Write-Host "  Done." -ForegroundColor Green

# ── 3. Check infrastructure ────────────────────────────────────────────────
Write-Host "`n[3/5] Checking infrastructure..." -ForegroundColor Yellow

$pgPort = netstat -ano | Select-String ":5432 " | Select-String "LISTENING"
$redisPort = netstat -ano | Select-String ":6379 " | Select-String "LISTENING"

if ($pgPort) { Write-Host "  PostgreSQL:  OK (port 5432)" -ForegroundColor Green } else { Write-Host "  PostgreSQL:  NOT RUNNING" -ForegroundColor Red }
if ($redisPort) { Write-Host "  Redis:       OK (port 6379)" -ForegroundColor Green } else { Write-Host "  Redis:       NOT RUNNING" -ForegroundColor Red }

if (-not $pgPort -or -not $redisPort) {
    Write-Host "`n  WARNING: PostgreSQL and Redis must be running!" -ForegroundColor Red
    Write-Host "  Start them manually or via Docker before continuing." -ForegroundColor Red
    $continue = Read-Host "  Continue anyway? (y/N)"
    if ($continue -ne "y") { exit }
}

# ── 4. Start API Gateway ───────────────────────────────────────────────────
Write-Host "`n[4/5] Starting API Gateway (port 8080)..." -ForegroundColor Yellow

$env:JWT_SECRET = "dev-secret-key-for-testing-only"
$env:POSTGRES_URL = "postgres://postgres:postgres@localhost:5432/pqap"
$env:REDIS_URL = "redis://localhost:6379"

Start-Process -FilePath "cmd" -ArgumentList "/c", "python -m uvicorn app.main:app --host 0.0.0.0 --port 8080 --reload" `
    -WorkingDirectory "$root\services\api-gateway" `
    -WindowStyle Hidden

Start-Sleep -Seconds 6

# Verify API Gateway
try {
    $test = Invoke-WebRequest -Uri "http://localhost:8080/api/auth/csrf" -UseBasicParsing -TimeoutSec 5
    Write-Host "  API Gateway: OK (status $($test.StatusCode))" -ForegroundColor Green
} catch {
    Write-Host "  API Gateway: FAILED to start" -ForegroundColor Red
}

# ── 5. Start Dashboard ────────────────────────────────────────────────────
Write-Host "`n[5/5] Starting Dashboard (port 3000)..." -ForegroundColor Yellow

Start-Process -FilePath "cmd" -ArgumentList "/c", "npm run dev" `
    -WorkingDirectory "$root\services\dashboard" `
    -WindowStyle Hidden

Write-Host "  Waiting for Next.js to compile..." -ForegroundColor Gray
Start-Sleep -Seconds 15

# Verify Dashboard
try {
    $test = Invoke-WebRequest -Uri "http://localhost:3000/login" -UseBasicParsing -TimeoutSec 10
    Write-Host "  Dashboard:   OK (status $($test.StatusCode))" -ForegroundColor Green
} catch {
    Write-Host "  Dashboard:   FAILED to start" -ForegroundColor Red
}

# ── Summary ────────────────────────────────────────────────────────────────
Write-Host "`n=== All Services Started ===" -ForegroundColor Cyan
Write-Host ""
Write-Host "  Dashboard:     http://localhost:3000" -ForegroundColor White
Write-Host "  API Gateway:   http://localhost:8080" -ForegroundColor White
Write-Host "  API Docs:      http://localhost:8080/docs" -ForegroundColor White
Write-Host ""
Write-Host "  Login credentials:" -ForegroundColor White
Write-Host "  Username: admin" -ForegroundColor Yellow
Write-Host "  Password: admin" -ForegroundColor Yellow
Write-Host ""
Write-Host "  Press Ctrl+C in this window to stop monitoring." -ForegroundColor Gray
Write-Host "  To stop all services, close this window or run: Get-Process node,python | Stop-Process" -ForegroundColor Gray
Write-Host ""

# ── Monitor (optional) ────────────────────────────────────────────────────
Write-Host "Monitoring services (Ctrl+C to stop)..." -ForegroundColor Gray
while ($true) {
    Start-Sleep -Seconds 30
    $apiOk = (netstat -ano | Select-String ":8080 " | Select-String "LISTENING") -ne $null
    $dashOk = (netstat -ano | Select-String ":3000 " | Select-String "LISTENING") -ne $null
    $ts = Get-Date -Format "HH:mm:ss"
    if ($apiOk -and $dashOk) {
        Write-Host "  [$ts] All services running" -ForegroundColor Green
    } else {
        if (-not $apiOk) { Write-Host "  [$ts] API Gateway DOWN!" -ForegroundColor Red }
        if (-not $dashOk) { Write-Host "  [$ts] Dashboard DOWN!" -ForegroundColor Red }
    }
}
