$ErrorActionPreference = "Stop"

param(
  [switch]$SkipInfra
)

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

if (-not $SkipInfra) {
  & "$PSScriptRoot/dev-local.ps1"
}

$apiCommand = "& `"$PSScriptRoot/dev-api.ps1`""
Start-Process powershell -ArgumentList "-NoExit", "-ExecutionPolicy", "Bypass", "-Command", $apiCommand -WorkingDirectory $root | Out-Null

$webCommand = "Set-Location `"$root/frontend/apps/web`"; npm run dev"
Start-Process powershell -ArgumentList "-NoExit", "-Command", $webCommand -WorkingDirectory $root | Out-Null

Write-Host ""
Write-Host "[dev-api-web] API e web iniciados em janelas separadas."
Write-Host "  API: http://127.0.0.1:8080 (ou APP_PORT)"
Write-Host "  Web: http://127.0.0.1:4173"
Write-Host ""
Write-Host "Dica: use -SkipInfra para nao reiniciar containers."
