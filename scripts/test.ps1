$ErrorActionPreference = "Stop"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  $env:Path = "C:\Program Files\Go\bin;" + $env:Path
}

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root
$env:GOCACHE = Join-Path $root ".gocache"

go test ./...
