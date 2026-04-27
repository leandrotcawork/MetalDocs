$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

# Load .env — split on first '=' only so PGPASSWORD=Lepa12<>! is preserved intact
Get-Content ".env" | ForEach-Object {
  if ($_ -match '^\s*#' -or $_ -match '^\s*$') { return }
  $name, $value = $_ -split '=', 2
  [System.Environment]::SetEnvironmentVariable($name.Trim(), $value.Trim(), 'Process')
}

# Build binary if missing or if -Build flag passed
$binary = Join-Path $root "metaldocs-worker.exe"
if (-not (Test-Path $binary) -or $args -contains "-Build") {
    Write-Host "Building metaldocs-worker.exe..."
    go build -o metaldocs-worker.exe ./apps/worker/cmd/metaldocs-worker/...
    if ($LASTEXITCODE -ne 0) { Write-Error "Build failed"; exit 1 }
}

Write-Host "Starting MetalDocs Worker (poll_interval=10s, docgen_v2=$env:METALDOCS_DOCGEN_V2_URL)"
& $binary
