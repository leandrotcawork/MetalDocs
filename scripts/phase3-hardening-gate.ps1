param(
  [string]$OutputDir = "non_git/hardening",
  [bool]$SkipGovulncheck = $true
)

$ErrorActionPreference = "Stop"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  $env:Path = "C:\Program Files\Go\bin;" + $env:Path
}

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root
$env:GOCACHE = Join-Path $root ".gocache"

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
$timestamp = [DateTime]::UtcNow.ToString("yyyyMMddTHHmmssZ")
$evidenceFile = Join-Path $OutputDir ("phase3_hardening_gate_" + $timestamp + ".json")

$result = [ordered]@{
  status = "running"
  started_utc = [DateTime]::UtcNow.ToString("o")
  finished_utc = $null
  duration_seconds = $null
  steps = [ordered]@{
    go_test = [ordered]@{
      exit_code = $null
      passed = $false
    }
    security_baseline = [ordered]@{
      skip_govulncheck = $SkipGovulncheck
      evidence_file = $null
      status = "not_run"
    }
  }
  error = $null
}

$started = [DateTime]::UtcNow

try {
  & "C:\Program Files\Go\bin\go.exe" test ./...
  $result.steps.go_test.exit_code = $LASTEXITCODE
  if ($LASTEXITCODE -ne 0) {
    throw "go test falhou com exit code $LASTEXITCODE"
  }
  $result.steps.go_test.passed = $true

  if ($SkipGovulncheck) {
    & "$PSScriptRoot/security-baseline.ps1" -SkipGovulncheck
  } else {
    & "$PSScriptRoot/security-baseline.ps1"
  }
  if ($LASTEXITCODE -ne 0) {
    throw "security-baseline falhou com exit code $LASTEXITCODE"
  }

  $securityEvidence = Get-ChildItem "non_git/security/security_baseline_*.json" `
    -File `
    | Sort-Object LastWriteTime -Descending `
    | Select-Object -First 1
  if (-not $securityEvidence) {
    throw "Nao foi encontrado arquivo de evidencia de security baseline."
  }

  $securityResult = Get-Content $securityEvidence.FullName | ConvertFrom-Json
  $result.steps.security_baseline.evidence_file = $securityEvidence.FullName
  $result.steps.security_baseline.status = $securityResult.status
  if ($securityResult.status -ne "approved") {
    throw "Security baseline nao aprovado."
  }

  $result.status = "approved"
}
catch {
  $result.status = "rejected"
  $result.error = $_.Exception.Message
  throw
}
finally {
  $finished = [DateTime]::UtcNow
  $result.finished_utc = $finished.ToString("o")
  $result.duration_seconds = [Math]::Round(($finished - $started).TotalSeconds, 3)
  $result | ConvertTo-Json -Depth 8 | Set-Content -Encoding UTF8 $evidenceFile
  Write-Host "Evidence file: $evidenceFile"
}
