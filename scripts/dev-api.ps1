$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

if (-not (Test-Path ".env")) {
  throw ".env not found. Copy .env.example to .env before running local dev."
}

Get-Content ".env" | ForEach-Object {
  if ($_ -match '^\s*#' -or $_ -match '^\s*$') {
    return
  }
  $name, $value = $_ -split '=', 2
  [System.Environment]::SetEnvironmentVariable($name, $value, 'Process')
}

$required = @(
  "METALDOCS_AUTH_SESSION_SECRET",
  "PGHOST",
  "PGPORT",
  "PGDATABASE",
  "PGUSER",
  "PGPASSWORD"
)

foreach ($name in $required) {
  $value = [System.Environment]::GetEnvironmentVariable($name, 'Process')
  if ([string]::IsNullOrWhiteSpace($value)) {
    throw "$name is required in .env for local API runtime."
  }
}

$appPort = if ([string]::IsNullOrWhiteSpace($env:APP_PORT)) { "8080" } else { $env:APP_PORT }
$listener = Get-NetTCPConnection -LocalPort $appPort -State Listen -ErrorAction SilentlyContinue
if ($listener) {
  $pids = ($listener | Select-Object -ExpandProperty OwningProcess -Unique) -join ","
  throw "APP_PORT $appPort is already in use by process(es): $pids. Stop the running API/gateway before starting local API."
}

Write-Host "[dev-api] Using Docker infra as source of truth:"
Write-Host "  PGHOST=$env:PGHOST"
Write-Host "  PGPORT=$env:PGPORT"
Write-Host "  METALDOCS_STORAGE_PROVIDER=$env:METALDOCS_STORAGE_PROVIDER"
Write-Host "[dev-api] Starting local API on port $appPort ..."

go run ./apps/api/cmd/metaldocs-api
