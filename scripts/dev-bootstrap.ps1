$ErrorActionPreference = "Stop"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  $goPath = "C:\Program Files\Go\bin\go.exe"
  if (Test-Path $goPath) {
    $env:Path = "C:\Program Files\Go\bin;" + $env:Path
  }
}

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  throw "Go not found. Install Go and re-run this script."
}

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$env:GOCACHE = Join-Path $root ".gocache"
if (-not (Test-Path $env:GOCACHE)) {
  New-Item -ItemType Directory -Path $env:GOCACHE | Out-Null
}

go version
go mod tidy
go test ./...

Write-Host "[dev-bootstrap] environment ready"
