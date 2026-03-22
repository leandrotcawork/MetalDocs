$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

if (-not (Test-Path ".env")) {
  throw ".env not found. Copy .env.example to .env before running local dev."
}

Write-Host "[dev-local] Stopping Docker app containers that should not run in fast local dev..."
docker compose -f deploy/compose/docker-compose.yml --env-file .env stop api web gateway worker | Out-Host

Write-Host "[dev-local] Starting Docker infra containers..."
docker compose -f deploy/compose/docker-compose.yml --env-file .env up -d postgres redis minio | Out-Host

Write-Host "[dev-local] Applying migrations (idempotent) ..."
powershell -ExecutionPolicy Bypass -File scripts/dev-migrate.ps1 | Out-Host

Write-Host ""
Write-Host "[dev-local] Fast local development mode is ready."
Write-Host "  1. Start API: powershell -ExecutionPolicy Bypass -File scripts/dev-api.ps1"
Write-Host "  2. Start web: cd frontend/apps/web; npm run dev"
Write-Host "  3. Open browser: http://127.0.0.1:4173"
Write-Host ""
Write-Host "[dev-local] Docker remains the single source of truth for Postgres/Redis/MinIO."
