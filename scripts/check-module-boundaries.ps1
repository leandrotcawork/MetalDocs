param()

$ErrorActionPreference = "Stop"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  if (Test-Path "C:\Program Files\Go\bin\go.exe") {
    $env:Path = "C:\Program Files\Go\bin;" + $env:Path
  }
}

$goCmd = Get-Command go -ErrorAction SilentlyContinue
if (-not $goCmd) {
  throw "Go toolchain nao encontrada no PATH."
}

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$moduleRoot = Join-Path $root "internal/modules"
if (-not (Test-Path $moduleRoot)) {
  throw "Diretorio internal/modules nao encontrado."
}

$violations = New-Object System.Collections.Generic.List[string]

$goFiles = Get-ChildItem -Path $moduleRoot -Recurse -Filter *.go -File `
  | Where-Object { $_.FullName -notmatch '_test\.go$' }

foreach ($file in $goFiles) {
  $fullName = (Resolve-Path $file.FullName).Path
  $rootWithSep = $root.TrimEnd('\') + '\'
  if ($fullName.StartsWith($rootWithSep, [System.StringComparison]::OrdinalIgnoreCase)) {
    $relativePath = $fullName.Substring($rootWithSep.Length).Replace("\", "/")
  } else {
    $relativePath = $fullName.Replace("\", "/")
  }
  if ($relativePath -notmatch '^internal/modules/([^/]+)/') {
    continue
  }
  $currentModule = $Matches[1]

  $content = Get-Content $file.FullName -Raw
  $importMatches = [regex]::Matches($content, '"metaldocs/internal/modules/([^/]+)/([^"]+)"')
  foreach ($match in $importMatches) {
    $targetModule = $match.Groups[1].Value
    $targetLayer = $match.Groups[2].Value

    if ($targetModule -eq $currentModule) {
      continue
    }

    if ($targetLayer -ne "domain") {
      $violations.Add("$relativePath -> metaldocs/internal/modules/$targetModule/$targetLayer")
    }
  }
}

if ($violations.Count -gt 0) {
  Write-Host "[module-boundaries] FAIL"
  Write-Host "Violacoes encontradas:"
  foreach ($v in $violations) {
    Write-Host (" - " + $v)
  }
  exit 1
}

Write-Host "[module-boundaries] OK"
