$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$envFile = Join-Path $root ".env.docgen-v2"

if (-not (Test-Path $envFile)) {
    Write-Error "[dev-docgen] Missing $envFile — copy from .env.docgen-v2.example and fill in values."
    exit 1
}

# Load env vars from .env.docgen-v2
Get-Content $envFile | ForEach-Object {
    if ($_ -match '^\s*([^#=]+?)\s*=\s*(.*)\s*$') {
        [System.Environment]::SetEnvironmentVariable($Matches[1], $Matches[2], 'Process')
    }
}

Write-Host "[dev-docgen] Starting docgen-v2 on port $env:DOCGEN_V2_PORT ..."
Set-Location (Join-Path $root "apps/docgen-v2")
npm.cmd run dev
