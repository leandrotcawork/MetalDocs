param(
  [Parameter(Mandatory = $true)]
  [string]$BackupFile,
  [string]$PgRestorePath = "pg_restore"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $BackupFile)) {
  throw "Arquivo de backup nao encontrado: $BackupFile"
}

Write-Host "Validando integridade logica do dump com pg_restore --list..."
& $PgRestorePath --list $BackupFile | Out-Null

if ($LASTEXITCODE -ne 0) {
  throw "Validacao de dump falhou com exit code $LASTEXITCODE"
}

Write-Host "Dump valido para restore."

