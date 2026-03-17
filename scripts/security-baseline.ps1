param(
  [string]$OutputDir = "non_git/security",
  [switch]$SkipGosec,
  [switch]$SkipGovulncheck
)

$ErrorActionPreference = "Stop"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  $env:Path = "C:\Program Files\Go\bin;" + $env:Path
}
$goBin = & go env GOPATH
if (-not [string]::IsNullOrWhiteSpace($goBin)) {
  $goBinPath = Join-Path $goBin "bin"
  if (Test-Path $goBinPath) {
    $env:Path = $goBinPath + ";" + $env:Path
  }
}

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root
$env:GOCACHE = Join-Path $root ".gocache"

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
$timestamp = [DateTime]::UtcNow.ToString("yyyyMMddTHHmmssZ")
$gosecOut = Join-Path $OutputDir ("gosec_" + $timestamp + ".txt")
$govulnOut = Join-Path $OutputDir ("govulncheck_" + $timestamp + ".txt")
$evidenceOut = Join-Path $OutputDir ("security_baseline_" + $timestamp + ".json")

$result = [ordered]@{
  status = "running"
  started_utc = [DateTime]::UtcNow.ToString("o")
  finished_utc = $null
  duration_seconds = $null
  gosec = [ordered]@{
    enabled = (-not $SkipGosec)
    tool_found = $false
    exit_code = $null
    output_file = $gosecOut
  }
  govulncheck = [ordered]@{
    enabled = (-not $SkipGovulncheck)
    tool_found = $false
    exit_code = $null
    output_file = $govulnOut
  }
  error = $null
}

$started = [DateTime]::UtcNow

try {
  if (-not $SkipGosec) {
    $gosecCmd = Get-Command gosec -ErrorAction SilentlyContinue
    if (-not $gosecCmd) {
      throw "Ferramenta 'gosec' nao encontrada. Instale com: go install github.com/securego/gosec/v2/cmd/gosec@latest"
    }

    $result.gosec.tool_found = $true
    $prevEap = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    & $gosecCmd.Source ./... *>&1 | Tee-Object -FilePath $gosecOut | Out-Null
    $ErrorActionPreference = $prevEap
    $result.gosec.exit_code = $LASTEXITCODE
    if ($LASTEXITCODE -ne 0) {
      throw "gosec falhou com exit code $LASTEXITCODE"
    }
  }

  if (-not $SkipGovulncheck) {
    $govulnCmd = Get-Command govulncheck -ErrorAction SilentlyContinue
    if (-not $govulnCmd) {
      throw "Ferramenta 'govulncheck' nao encontrada. Instale com: go install golang.org/x/vuln/cmd/govulncheck@latest"
    }

    $result.govulncheck.tool_found = $true
    $prevEap = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    & $govulnCmd.Source ./... *>&1 | Tee-Object -FilePath $govulnOut | Out-Null
    $ErrorActionPreference = $prevEap
    $result.govulncheck.exit_code = $LASTEXITCODE
    if ($LASTEXITCODE -ne 0) {
      throw "govulncheck falhou com exit code $LASTEXITCODE"
    }
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
  $result | ConvertTo-Json -Depth 8 | Set-Content -Encoding UTF8 $evidenceOut
  Write-Host "Evidence file: $evidenceOut"
}
