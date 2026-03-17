param(
  [string]$EnvironmentName = "local",
  [string]$BackupDir = "backups",
  [string]$EvidenceDir = "backups/evidence",
  [string]$RestoreValidationDatabase = "metaldocs_restore_test",
  [string]$PgDumpPath = "pg_dump",
  [string]$PgRestorePath = "pg_restore",
  [string]$PsqlPath = "psql",
  [Parameter(Mandatory = $true)]
  [string]$BackupUser,
  [Parameter(Mandatory = $true)]
  [string]$BackupPassword,
  [Parameter(Mandatory = $true)]
  [string]$RestoreUser,
  [Parameter(Mandatory = $true)]
  [string]$RestorePassword
)

$ErrorActionPreference = "Stop"

if ($BackupUser -eq "metaldocs_app") {
  throw "BackupUser nao pode ser metaldocs_app."
}

if (-not $env:PGHOST -or -not $env:PGPORT -or -not $env:PGDATABASE) {
  throw "PGHOST, PGPORT e PGDATABASE sao obrigatorios no ambiente."
}

New-Item -ItemType Directory -Force -Path $EvidenceDir | Out-Null
$gateStartedAt = [DateTime]::UtcNow
$gateTimestamp = [DateTime]::UtcNow.ToString("yyyyMMddTHHmmssZ")
$evidenceFile = Join-Path $EvidenceDir ("backup_restore_gate_" + $EnvironmentName + "_" + $env:PGDATABASE + "_" + $gateTimestamp + ".json")

function Invoke-ScalarQuery {
  param(
    [string]$Database,
    [string]$User,
    [string]$Password,
    [string]$Sql
  )
  $env:PGPASSWORD = $Password
  $value = & $PsqlPath `
    --host $env:PGHOST `
    --port $env:PGPORT `
    --username $User `
    --dbname $Database `
    --tuples-only `
    --no-align `
    --set ON_ERROR_STOP=1 `
    --command $Sql

  if ($LASTEXITCODE -ne 0) {
    throw "psql falhou ao executar query no database '$Database'."
  }
  return "$value".Trim()
}

function Invoke-NonQuery {
  param(
    [string]$Database,
    [string]$User,
    [string]$Password,
    [string]$Sql
  )
  $env:PGPASSWORD = $Password
  & $PsqlPath `
    --host $env:PGHOST `
    --port $env:PGPORT `
    --username $User `
    --dbname $Database `
    --set ON_ERROR_STOP=1 `
    --command $Sql | Out-Null

  if ($LASTEXITCODE -ne 0) {
    throw "psql falhou ao executar comando no database '$Database'."
  }
}

$evidence = [ordered]@{
  status = "running"
  operation = "backup_restore_gate"
  started_utc = $gateStartedAt.ToString("o")
  finished_utc = $null
  duration_seconds = $null
  operator = $env:USERNAME
  environment = $EnvironmentName
  source_database = $env:PGDATABASE
  restore_validation_database = $RestoreValidationDatabase
  backup = $null
  validation = $null
  restore = $null
  smoke = $null
  error = $null
}

try {
  $quotedDbNameLiteral = $RestoreValidationDatabase.Replace("'", "''")
  $quotedDbNameIdentifier = $RestoreValidationDatabase.Replace('"', '""')

  $backupResult = & "$PSScriptRoot/backup-postgres.ps1" `
    -BackupDir $BackupDir `
    -PgDumpPath $PgDumpPath `
    -EnvironmentName $EnvironmentName `
    -PgUser $BackupUser `
    -PgPassword $BackupPassword
  $evidence.backup = $backupResult

  $validateResult = & "$PSScriptRoot/validate-backup.ps1" `
    -BackupFile $backupResult.backup_file `
    -PgRestorePath $PgRestorePath `
    -EnvironmentName $EnvironmentName
  $evidence.validation = $validateResult

  $exists = Invoke-ScalarQuery `
    -Database "postgres" `
    -User $RestoreUser `
    -Password $RestorePassword `
    -Sql "SELECT 1 FROM pg_database WHERE datname = '$quotedDbNameLiteral';"

  if ($exists -ne "1") {
    Invoke-NonQuery `
      -Database "postgres" `
      -User $RestoreUser `
      -Password $RestorePassword `
      -Sql "CREATE DATABASE `"$quotedDbNameIdentifier`";"
  }

  $restoreResult = & "$PSScriptRoot/restore-postgres.ps1" `
    -BackupFile $backupResult.backup_file `
    -TargetDatabase $RestoreValidationDatabase `
    -PgRestorePath $PgRestorePath `
    -EnvironmentName $EnvironmentName `
    -PgUser $RestoreUser `
    -PgPassword $RestorePassword
  $evidence.restore = $restoreResult

  $documentsCount = Invoke-ScalarQuery `
    -Database $RestoreValidationDatabase `
    -User $RestoreUser `
    -Password $RestorePassword `
    -Sql "SELECT COUNT(*) FROM metaldocs.documents;"
  $versionsCount = Invoke-ScalarQuery `
    -Database $RestoreValidationDatabase `
    -User $RestoreUser `
    -Password $RestorePassword `
    -Sql "SELECT COUNT(*) FROM metaldocs.document_versions;"
  $usersCount = Invoke-ScalarQuery `
    -Database $RestoreValidationDatabase `
    -User $RestoreUser `
    -Password $RestorePassword `
    -Sql "SELECT COUNT(*) FROM metaldocs.iam_users;"
  $rolesCount = Invoke-ScalarQuery `
    -Database $RestoreValidationDatabase `
    -User $RestoreUser `
    -Password $RestorePassword `
    -Sql "SELECT COUNT(*) FROM metaldocs.iam_user_roles;"
  $auditCount = Invoke-ScalarQuery `
    -Database $RestoreValidationDatabase `
    -User $RestoreUser `
    -Password $RestorePassword `
    -Sql "SELECT COUNT(*) FROM metaldocs.audit_events;"
  $outboxCount = Invoke-ScalarQuery `
    -Database $RestoreValidationDatabase `
    -User $RestoreUser `
    -Password $RestorePassword `
    -Sql "SELECT COUNT(*) FROM metaldocs.outbox_events;"

  $evidence.smoke = [ordered]@{
    status = "success"
    checks = [ordered]@{
      documents_count = [int64]$documentsCount
      document_versions_count = [int64]$versionsCount
      iam_users_count = [int64]$usersCount
      iam_user_roles_count = [int64]$rolesCount
      audit_events_count = [int64]$auditCount
      outbox_events_count = [int64]$outboxCount
    }
  }

  $evidence.status = "approved"
}
catch {
  $evidence.status = "rejected"
  $evidence.error = $_.Exception.Message
  throw
}
finally {
  $gateFinishedAt = [DateTime]::UtcNow
  $evidence.finished_utc = $gateFinishedAt.ToString("o")
  $evidence.duration_seconds = [Math]::Round(($gateFinishedAt - $gateStartedAt).TotalSeconds, 3)
  $evidence | ConvertTo-Json -Depth 8 | Set-Content -Encoding UTF8 $evidenceFile
  Write-Host "Evidence file: $evidenceFile"
  $evidence
}
