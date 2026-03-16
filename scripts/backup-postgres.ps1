param(
  [string]$BackupDir = "backups",
  [string]$PgDumpPath = "pg_dump",
  [string]$PgUser = "",
  [string]$PgPassword = ""
)

$ErrorActionPreference = "Stop"

if (-not $env:PGHOST -or -not $env:PGPORT -or -not $env:PGDATABASE) {
  throw "PGHOST, PGPORT e PGDATABASE sao obrigatorios no ambiente."
}

$effectiveUser = $PgUser
if ([string]::IsNullOrWhiteSpace($effectiveUser)) {
  $effectiveUser = $env:PGUSER
}
if ([string]::IsNullOrWhiteSpace($effectiveUser)) {
  throw "Informe -PgUser ou configure PGUSER."
}

$effectivePassword = $PgPassword
if ([string]::IsNullOrWhiteSpace($effectivePassword)) {
  $effectivePassword = $env:PGPASSWORD
}
if ([string]::IsNullOrWhiteSpace($effectivePassword)) {
  throw "Informe -PgPassword ou configure PGPASSWORD."
}

New-Item -ItemType Directory -Force -Path $BackupDir | Out-Null
$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$filePath = Join-Path $BackupDir ("metaldocs_" + $timestamp + ".dump")

Write-Host "Iniciando backup PostgreSQL..."
$env:PGPASSWORD = $effectivePassword
& $PgDumpPath `
  --host $env:PGHOST `
  --port $env:PGPORT `
  --username $effectiveUser `
  --format custom `
  --no-owner `
  --no-privileges `
  --file $filePath `
  $env:PGDATABASE

if ($LASTEXITCODE -ne 0) {
  throw "pg_dump falhou com exit code $LASTEXITCODE"
}

Write-Host "Backup concluido com sucesso:"
Write-Host $filePath
