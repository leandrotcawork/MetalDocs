param(
  [switch]$ApiOnly,
  [switch]$WebOnly
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

if ($ApiOnly -and $WebOnly) {
  throw "Use only one of -ApiOnly or -WebOnly."
}

if (-not (Test-Path ".env")) {
  throw ".env not found. Copy .env.example to .env before running."
}

if (-not $WebOnly) {
  Start-Process powershell -ArgumentList "-NoExit", "-ExecutionPolicy", "Bypass", "-File", "$PSScriptRoot/dev-api.ps1" -WorkingDirectory $root | Out-Null
}

if (-not $ApiOnly) {
  $webCommand = "Set-Location `"$root/frontend/apps/web`"; npm run dev"
  Start-Process powershell -ArgumentList "-NoExit", "-Command", $webCommand -WorkingDirectory $root | Out-Null
}

Write-Host ""
Write-Host "[run_metaldocs] Started."
Write-Host "  API: http://127.0.0.1:8080 (or APP_PORT)"
Write-Host "  Web: http://127.0.0.1:4173"
Write-Host ""
Write-Host "Tips:"
Write-Host "  - Only API: powershell -ExecutionPolicy Bypass -File scripts/run_metaldocs.ps1 -ApiOnly"
Write-Host "  - Only web: powershell -ExecutionPolicy Bypass -File scripts/run_metaldocs.ps1 -WebOnly"
