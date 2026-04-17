$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

Get-Content ".env" | ForEach-Object {
  if ($_ -match '^\s*#' -or $_ -match '^\s*$') { return }
  $name, $value = $_ -split '=', 2
  [System.Environment]::SetEnvironmentVariable($name.Trim(), $value.Trim(), 'Process')
}

# Override port for plan-c worktree to avoid conflict with main API on 8081
[System.Environment]::SetEnvironmentVariable('APP_PORT', '8083', 'Process')

go run ./apps/api/cmd/metaldocs-api
