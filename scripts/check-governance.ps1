param(
  [string]$BaseRef = "origin/main"
)

$ErrorActionPreference = "Stop"

$changed = git diff --name-only "$BaseRef...HEAD"
$changedText = ($changed -join "`n")

Write-Host "Changed files:"
Write-Host $changedText

function Fail([string]$msg) {
  Write-Error "[governance-check] $msg"
  exit 1
}

# API contract-impacting changes must update OpenAPI.
# We intentionally scope to delivery/http handlers and API spec files to avoid false positives
# for non-contract bootstrap changes in apps/api.
if ($changedText -match '(?m)^internal/modules/.+/delivery/http/.+\.go$') {
  if ($changedText -notmatch '(?m)^api/openapi/v1/openapi.yaml$') {
    Fail "API contract change detected without OpenAPI update."
  }
}

if ($changedText -match '(?m)^internal/modules/') {
  if ($changedText -notmatch '(?m)^tests/') {
    Fail "Domain change detected without test updates under tests/."
  }
}

if ($changedText -match '(?m)^(deploy/|scripts/)') {
  if ($changedText -notmatch '(?m)^docs/runbooks/') {
    Fail "Infra/ops change detected without runbook update."
  }
}

Write-Host "[governance-check] OK"
