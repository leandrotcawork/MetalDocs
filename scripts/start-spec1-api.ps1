$ErrorActionPreference = "Stop"
$worktree = "C:\Users\leandro.theodoro.MN-NTB-LEANDROT\Documents\MetalDocs\.claude\worktrees\feature+foundation-spec1"
Set-Location $worktree

Get-Content "C:\Users\leandro.theodoro.MN-NTB-LEANDROT\Documents\MetalDocs\.env" | ForEach-Object {
  if ($_ -match '^\s*#' -or $_ -match '^\s*$') { return }
  $name, $value = $_ -split '=', 2
  [System.Environment]::SetEnvironmentVariable($name.Trim(), $value.Trim(), 'Process')
}

[System.Environment]::SetEnvironmentVariable('PGHOST', 'localhost', 'Process')
[System.Environment]::SetEnvironmentVariable('PGPORT', '5433', 'Process')
[System.Environment]::SetEnvironmentVariable('PGDATABASE', 'metaldocs', 'Process')
[System.Environment]::SetEnvironmentVariable('PGUSER', 'metaldocs_app', 'Process')
[System.Environment]::SetEnvironmentVariable('PGPASSWORD', 'Lepa12<>!', 'Process')
[System.Environment]::SetEnvironmentVariable('PGSSLMODE', 'disable', 'Process')
[System.Environment]::SetEnvironmentVariable('APP_PORT', '8081', 'Process')

go run ./apps/api/cmd/metaldocs-api
