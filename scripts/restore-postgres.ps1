param(
  [Parameter(Mandatory = $true)]
  [string]$BackupFile,
  [string]$TargetDatabase = "",
  [string]$PgRestorePath = "pg_restore",
  [string]$PgUser = "",
  [string]$PgPassword = ""
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $BackupFile)) {
  throw "Arquivo de backup nao encontrado: $BackupFile"
}

if (-not $env:PGHOST -or -not $env:PGPORT) {
  throw "PGHOST e PGPORT sao obrigatorios no ambiente."
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

if ([string]::IsNullOrWhiteSpace($TargetDatabase)) {
  if (-not $env:PGDATABASE) {
    throw "Informe -TargetDatabase ou configure PGDATABASE."
  }
  $TargetDatabase = $env:PGDATABASE
}

Write-Host "Iniciando restore PostgreSQL..."
$env:PGPASSWORD = $effectivePassword
& $PgRestorePath `
  --host $env:PGHOST `
  --port $env:PGPORT `
  --username $effectiveUser `
  --dbname $TargetDatabase `
  --clean `
  --if-exists `
  --no-owner `
  --no-privileges `
  $BackupFile

if ($LASTEXITCODE -ne 0) {
  throw "pg_restore falhou com exit code $LASTEXITCODE"
}

Write-Host "Restore concluido com sucesso em database: $TargetDatabase"
