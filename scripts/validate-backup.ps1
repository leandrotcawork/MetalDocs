param(
  [Parameter(Mandatory = $true)]
  [string]$BackupFile,
  [string]$PgRestorePath = "pg_restore",
  [string]$EnvironmentName = "local"
)

$ErrorActionPreference = "Stop"
$startedAt = [DateTime]::UtcNow

if (-not (Test-Path $BackupFile)) {
  throw "Arquivo de backup nao encontrado: $BackupFile"
}

Write-Host "Validando integridade logica do dump com pg_restore --list..."
& $PgRestorePath --list $BackupFile | Out-Null

if ($LASTEXITCODE -ne 0) {
  throw "Validacao de dump falhou com exit code $LASTEXITCODE"
}

Write-Host "Dump valido para restore."
$finishedAt = [DateTime]::UtcNow
$durationSeconds = [Math]::Round(($finishedAt - $startedAt).TotalSeconds, 3)

[PSCustomObject]@{
  status = "success"
  operation = "validate"
  started_utc = $startedAt.ToString("o")
  finished_utc = $finishedAt.ToString("o")
  duration_seconds = $durationSeconds
  operator = $env:USERNAME
  environment = $EnvironmentName
  backup_file = $BackupFile
  validation_passed = $true
}
