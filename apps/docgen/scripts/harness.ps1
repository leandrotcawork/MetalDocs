$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root

$tempDir = [System.IO.Path]::GetTempPath()
$docxOut = Join-Path $tempDir "docgen-harness.docx"

try {
  Write-Host "==> Typecheck (tsc --noEmit)"
  npx tsc --noEmit

  Write-Host "==> Build (tsc -p tsconfig.build.json)"
  npx tsc -p tsconfig.build.json

  Write-Host "==> Start server (node dist/index.js)"
  $oldPort = $env:PORT
  $env:PORT = "3002"
  $proc = Start-Process -FilePath "node" -ArgumentList "dist/index.js" -PassThru -NoNewWindow

  $ready = $false
  for ($i = 0; $i -lt 30; $i++) {
    try {
      $probe = curl.exe -s -o NUL -w "%{http_code}" "http://localhost:3002/"
      if ($probe) {
        $ready = $true
        break
      }
    } catch {
      Start-Sleep -Milliseconds 500
    }
    Start-Sleep -Milliseconds 500
  }
  if (!$ready) { throw "docgen did not start on port 3002" }

  Write-Host "==> POST /generate"
  $resp = curl.exe -s -D - -o "$docxOut" `
    -H "Content-Type: application/json" `
    -X POST "http://localhost:3002/generate" `
    --data-binary "@$PSScriptRoot\\sample-payload.json"

  $len = (Get-Item $docxOut).Length
  if ($len -le 0) { throw "DOCX is empty" }

  $headerText = $resp -join " "
  if ($headerText -notmatch "(?i)application/vnd.openxmlformats-officedocument.wordprocessingml.document") {
    Write-Host $headerText
    throw "Unexpected content type"
  }

  Write-Host "OK: DOCX size = $len bytes"
}
finally {
  if ($proc -and !$proc.HasExited) { Stop-Process -Id $proc.Id }
  $env:PORT = $oldPort
  if (Test-Path $docxOut) { Remove-Item $docxOut -Force }
  Pop-Location
}
