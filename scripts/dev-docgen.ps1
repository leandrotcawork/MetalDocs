$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

Write-Host "[dev-docgen] Starting local docgen on port 3001 ..."
Set-Location "$root/apps/docgen-v2"
npm.cmd run start
