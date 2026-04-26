$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

# Load .env — split on first '=' only so PGPASSWORD=Lepa12<>! is preserved intact
Get-Content ".env" | ForEach-Object {
  if ($_ -match '^\s*#' -or $_ -match '^\s*$') { return }
  $name, $value = $_ -split '=', 2
  [System.Environment]::SetEnvironmentVariable($name.Trim(), $value.Trim(), 'Process')
}

# APP_PORT must be 8081 — override in case .env is missing it
[System.Environment]::SetEnvironmentVariable('APP_PORT', '8081', 'Process')

# Kill any process already holding :8081
$held = netstat -ano 2>$null | Select-String ":8081 " | ForEach-Object { ($_ -split '\s+')[5] } | Select-Object -First 1
if ($held) {
    Write-Host "Killing PID $held (was holding :8081)"
    Stop-Process -Id $held -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 1
}

# Build binary if missing or if -Build flag passed
$binary = Join-Path $root "metaldocs-api.exe"
if (-not (Test-Path $binary) -or $args -contains "-Build") {
    Write-Host "Building metaldocs-api.exe..."
    go build -o metaldocs-api.exe ./apps/api/cmd/metaldocs-api/...
    if ($LASTEXITCODE -ne 0) { Write-Error "Build failed"; exit 1 }
}

Write-Host "Starting MetalDocs API on :8081  (admin / AdminMetalDocs123!)"
& $binary
