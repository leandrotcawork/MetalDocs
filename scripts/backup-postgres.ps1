param(
  [string]$BackupDir = "backups",
  [string]$PgDumpPath = "pg_dump",
  [string]$EnvironmentName = "local",
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
if ($effectiveUser -eq "metaldocs_app") {
  throw "Backup deve usar usuario dedicado (nao metaldocs_app)."
}

$effectivePassword = $PgPassword
if ([string]::IsNullOrWhiteSpace($effectivePassword)) {
  $effectivePassword = $env:PGPASSWORD
}
if ([string]::IsNullOrWhiteSpace($effectivePassword)) {
  throw "Informe -PgPassword ou configure PGPASSWORD."
}

New-Item -ItemType Directory -Force -Path $BackupDir | Out-Null
$startedAt = [DateTime]::UtcNow
$timestamp = [DateTime]::UtcNow.ToString("yyyyMMddTHHmmssZ")
$dbNameSanitized = ($env:PGDATABASE -replace "[^a-zA-Z0-9_\-]", "_")
$envSanitized = ($EnvironmentName -replace "[^a-zA-Z0-9_\-]", "_")
$filePath = Join-Path $BackupDir ("metaldocs_" + $envSanitized + "_" + $dbNameSanitized + "_" + $timestamp + ".dump")

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

$finishedAt = [DateTime]::UtcNow
$durationSeconds = [Math]::Round(($finishedAt - $startedAt).TotalSeconds, 3)
$checksum = (Get-FileHash -Path $filePath -Algorithm SHA256).Hash.ToLowerInvariant()
$result = [PSCustomObject]@{
  status = "success"
  operation = "backup"
  started_utc = $startedAt.ToString("o")
  finished_utc = $finishedAt.ToString("o")
  duration_seconds = $durationSeconds
  operator = $env:USERNAME
  environment = $EnvironmentName
  database = $env:PGDATABASE
  pg_host = $env:PGHOST
  pg_port = $env:PGPORT
  pg_user = $effectiveUser
  backup_file = $filePath
  checksum_sha256 = $checksum
}

Write-Host "Backup concluido com sucesso: $filePath"
$result
