param(
  [Parameter(Mandatory = $true)]
  [string]$BackupFile,
  [string]$TargetDatabase = "",
  [string]$PgRestorePath = "pg_restore",
  [string]$EnvironmentName = "local",
  [string]$PgUser = "",
  [string]$PgPassword = ""
)

$ErrorActionPreference = "Stop"
$startedAt = [DateTime]::UtcNow

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
  --exit-on-error `
  --no-owner `
  --no-privileges `
  $BackupFile

if ($LASTEXITCODE -ne 0) {
  throw "pg_restore falhou com exit code $LASTEXITCODE"
}

$finishedAt = [DateTime]::UtcNow
$durationSeconds = [Math]::Round(($finishedAt - $startedAt).TotalSeconds, 3)

Write-Host "Restore concluido com sucesso em database: $TargetDatabase"
[PSCustomObject]@{
  status = "success"
  operation = "restore"
  started_utc = $startedAt.ToString("o")
  finished_utc = $finishedAt.ToString("o")
  duration_seconds = $durationSeconds
  operator = $env:USERNAME
  environment = $EnvironmentName
  pg_host = $env:PGHOST
  pg_port = $env:PGPORT
  pg_user = $effectiveUser
  target_database = $TargetDatabase
  backup_file = $BackupFile
}
