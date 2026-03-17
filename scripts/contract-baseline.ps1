param(
  [string]$OutputDir = "non_git/contract"
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
$evidenceFile = Join-Path $OutputDir ("contract_baseline_" + $timestamp + ".json")

$result = [ordered]@{
  status = "running"
  started_utc = [DateTime]::UtcNow.ToString("o")
  finished_utc = $null
  duration_seconds = $null
  test_target = "./tests/contract"
  exit_code = $null
  error = $null
}

$started = [DateTime]::UtcNow

try {
  & "C:\Program Files\Go\bin\go.exe" test ./tests/contract -count=1
  $result.exit_code = $LASTEXITCODE
  if ($LASTEXITCODE -ne 0) {
    throw "contract tests falharam com exit code $LASTEXITCODE"
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
