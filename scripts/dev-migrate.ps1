param(
  [string]$ComposeFile = "deploy/compose/docker-compose.yml",
  [string]$EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

if (-not (Test-Path $EnvFile)) {
  throw "$EnvFile not found. Copy .env.example to .env before running migrations."
}

Get-Content $EnvFile | ForEach-Object {
  if ($_ -match '^\s*#' -or $_ -match '^\s*$') {
    return
  }
  $name, $value = $_ -split '=', 2
  [System.Environment]::SetEnvironmentVariable($name, $value, 'Process')
}

if ([string]::IsNullOrWhiteSpace($env:POSTGRES_USER) -or [string]::IsNullOrWhiteSpace($env:POSTGRES_DB)) {
  throw "POSTGRES_USER and POSTGRES_DB are required in $EnvFile to apply migrations."
}

if (-not (Test-Path $ComposeFile)) {
  throw "Compose file not found: $ComposeFile"
}

$migrationsPath = Join-Path $root "migrations"
if (-not (Test-Path $migrationsPath)) {
  throw "migrations folder not found: $migrationsPath"
}

$migrations = Get-ChildItem -Path $migrationsPath -Filter "*.sql" | Sort-Object -Property Name
if ($migrations.Count -eq 0) {
  throw "No migrations found in $migrationsPath"
}

Write-Host "[dev-migrate] Applying $($migrations.Count) migration(s) to Postgres container..."
Write-Host "  user=$env:POSTGRES_USER db=$env:POSTGRES_DB"

foreach ($migration in $migrations) {
  $containerPath = "/docker-entrypoint-initdb.d/$($migration.Name)"
  Write-Host "[dev-migrate] -> $($migration.Name)"

  docker compose -f $ComposeFile --env-file $EnvFile exec -T postgres `
    psql -v ON_ERROR_STOP=1 -U $env:POSTGRES_USER -d $env:POSTGRES_DB -f $containerPath | Out-Host
}

Write-Host "[dev-migrate] Done."
