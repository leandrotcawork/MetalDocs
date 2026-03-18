$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

if (-not (Test-Path ".env")) {
  throw ".env not found. Copy .env.example to .env before running e2e seed."
}

Get-Content ".env" | ForEach-Object {
  if ($_ -match '^\s*#' -or $_ -match '^\s*$') {
    return
  }
  $name, $value = $_ -split '=', 2
  [System.Environment]::SetEnvironmentVariable($name, $value, 'Process')
}

go run ./apps/api/cmd/metaldocs-e2e-seed
