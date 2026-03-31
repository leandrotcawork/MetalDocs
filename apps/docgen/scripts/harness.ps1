$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root

try {
  Write-Host "==> Typecheck (tsc --noEmit)"
  npx tsc --noEmit

  Write-Host "==> Build (tsc -p tsconfig.build.json)"
  npx tsc -p tsconfig.build.json

  Write-Host "==> Start server (node dist/index.js)"
  $proc = Start-Process -FilePath "node" -ArgumentList "dist/index.js" -PassThru -NoNewWindow
  Start-Sleep -Seconds 2

  Write-Host "==> POST /generate"
  $resp = curl.exe -s -D - -o "$env:TEMP\\docgen-harness.docx" `
    -H "Content-Type: application/json" `
    -X POST "http://localhost:3001/generate" `
    --data-binary "@$PSScriptRoot\\sample-payload.json"

  $len = (Get-Item "$env:TEMP\\docgen-harness.docx").Length
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
  Pop-Location
}
