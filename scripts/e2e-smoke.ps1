$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

if (-not (Test-Path ".env")) {
  throw ".env not found. Copy .env.example to .env before running E2E smoke."
}

Get-Content ".env" | ForEach-Object {
  if ($_ -match '^\s*#' -or $_ -match '^\s*$') {
    return
  }
  $name, $value = $_ -split '=', 2
  [System.Environment]::SetEnvironmentVariable($name, $value, 'Process')
}

$apiLog = Join-Path $root "non_git/e2e_api.log"
$apiErr = Join-Path $root "non_git/e2e_api.err.log"
$webLog = Join-Path $root "non_git/e2e_web.log"
$webErr = Join-Path $root "non_git/e2e_web.err.log"

$appPort = if ([string]::IsNullOrWhiteSpace($env:APP_PORT)) { "8080" } else { $env:APP_PORT }
$proxyTarget = if ([string]::IsNullOrWhiteSpace($env:VITE_API_PROXY_TARGET)) { "http://127.0.0.1:$appPort" } else { $env:VITE_API_PROXY_TARGET }

New-Item -ItemType Directory -Force -Path (Join-Path $root "non_git") | Out-Null

powershell -ExecutionPolicy Bypass -File scripts/dev-local.ps1

$apiProcess = Start-Process powershell -ArgumentList "-ExecutionPolicy", "Bypass", "-File", "scripts/dev-api.ps1" -WorkingDirectory $root -RedirectStandardOutput $apiLog -RedirectStandardError $apiErr -PassThru
$webCommand = "`$env:VITE_API_PROXY_TARGET='$proxyTarget'; cd frontend/apps/web; & 'C:\Program Files\nodejs\npm.cmd' run dev -- --host 127.0.0.1 --port 4173"
$webProcess = Start-Process powershell -ArgumentList "-Command", $webCommand -WorkingDirectory $root -RedirectStandardOutput $webLog -RedirectStandardError $webErr -PassThru

try {
  $deadline = (Get-Date).AddSeconds(60)
  do {
    Start-Sleep -Milliseconds 750
    try {
      $apiReady = (Invoke-WebRequest -UseBasicParsing -Uri "http://127.0.0.1:$appPort/api/v1/health/ready" -TimeoutSec 3).StatusCode -eq 200
    } catch {
      $apiReady = $false
    }
    try {
      $webReady = (Invoke-WebRequest -UseBasicParsing -Uri "http://127.0.0.1:4173" -TimeoutSec 3).StatusCode -eq 200
    } catch {
      $webReady = $false
    }
  } while ((-not ($apiReady -and $webReady)) -and (Get-Date) -lt $deadline)

  if (-not $apiReady) {
    throw "API local did not become ready. Check $apiLog and $apiErr."
  }
  if (-not $webReady) {
    throw "Web local did not become ready. Check $webLog and $webErr."
  }

  powershell -ExecutionPolicy Bypass -File scripts/e2e-seed.ps1
  & 'C:\Program Files\nodejs\npm.cmd' --prefix frontend/apps/web run e2e:smoke
}
finally {
  if ($apiProcess -and -not $apiProcess.HasExited) {
    Stop-Process -Id $apiProcess.Id -Force
  }
  if ($webProcess -and -not $webProcess.HasExited) {
    Stop-Process -Id $webProcess.Id -Force
  }
}
