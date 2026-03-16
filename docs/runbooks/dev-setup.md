# Runbook: Dev Setup (Go)

## Important
Go projects do not use Python `venv` or `requirements.txt`.

Official dependency source of truth:
- `go.mod` (direct requirements and module metadata)
- `go.sum` (dependency lock/checksum)

## 1) Install Go on Windows
Preferred:
```powershell
winget install -e --id GoLang.Go
```

Validate:
```powershell
go version
```

## 2) Bootstrap project locally
From repo root:
```powershell
powershell -ExecutionPolicy Bypass -File scripts/dev-bootstrap.ps1
```

## 3) Commands used day to day
```powershell
# run tests
powershell -ExecutionPolicy Bypass -File scripts/test.ps1

# sync go.mod/go.sum
powershell -ExecutionPolicy Bypass -File scripts/tidy.ps1
```

## 4) Dependency policy
- Never use `go get -u ./...` blindly.
- Add/update only what is needed for the current feature.
- Commit `go.mod` and `go.sum` together when dependency changes.
- Any critical dependency addition requires ADR/RFC according to `AGENTS.md`.
