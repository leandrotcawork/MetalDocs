param(
  [string]$BaseRef = "origin/main",
  [string]$OutputDir = "non_git/release",
  [bool]$SkipGovulncheck = $true
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
$timestamp = [DateTime]::UtcNow.ToString("yyyyMMddTHHmmssZ")
$evidenceFile = Join-Path $OutputDir ("phase3_release_readiness_" + $timestamp + ".json")

$result = [ordered]@{
  status = "running"
  started_utc = [DateTime]::UtcNow.ToString("o")
  finished_utc = $null
  duration_seconds = $null
  base_ref = $BaseRef
  checks = [ordered]@{
    governance_check = [ordered]@{
      status = "not_run"
      error = $null
    }
    hardening_gate = [ordered]@{
      status = "not_run"
      evidence_file = $null
      error = $null
    }
  }
  error = $null
}

$started = [DateTime]::UtcNow

try {
  & "$PSScriptRoot/check-governance.ps1" -BaseRef $BaseRef
  if ($LASTEXITCODE -ne 0) {
    throw "governance-check falhou com exit code $LASTEXITCODE"
  }
  $result.checks.governance_check.status = "approved"

  & "$PSScriptRoot/phase3-hardening-gate.ps1" -SkipGovulncheck:$SkipGovulncheck
  if ($LASTEXITCODE -ne 0) {
    throw "phase3-hardening-gate falhou com exit code $LASTEXITCODE"
  }

  $hardeningEvidence = Get-ChildItem "non_git/hardening/phase3_hardening_gate_*.json" `
    -File `
    | Sort-Object LastWriteTime -Descending `
    | Select-Object -First 1

  if (-not $hardeningEvidence) {
    throw "Nao foi encontrado arquivo de evidencia do hardening gate."
  }

  $hardeningResult = Get-Content $hardeningEvidence.FullName | ConvertFrom-Json
  $result.checks.hardening_gate.evidence_file = $hardeningEvidence.FullName
  $result.checks.hardening_gate.status = $hardeningResult.status
  if ($hardeningResult.status -ne "approved") {
    throw "hardening gate nao aprovado."
  }

  $result.status = "approved"
}
catch {
  $result.status = "rejected"
  $result.error = $_.Exception.Message
  if ($result.checks.governance_check.status -eq "not_run") {
    $result.checks.governance_check.error = $_.Exception.Message
  } elseif ($result.checks.hardening_gate.status -eq "not_run") {
    $result.checks.hardening_gate.error = $_.Exception.Message
  }
  throw
}
finally {
  $finished = [DateTime]::UtcNow
  $result.finished_utc = $finished.ToString("o")
  $result.duration_seconds = [Math]::Round(($finished - $started).TotalSeconds, 3)
  $result | ConvertTo-Json -Depth 8 | Set-Content -Encoding UTF8 $evidenceFile
  Write-Host "Evidence file: $evidenceFile"
}
