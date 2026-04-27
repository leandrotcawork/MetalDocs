# Local Dev Startup

**Last verified:** 2026-04-26

## TL;DR

```powershell
# From repo root — PowerShell only, never bash
.\scripts\start-api.ps1
```

Frontend (separate terminal):
```bash
cd frontend/apps/web && pnpm dev
# → http://localhost:4174
```

---

## Why PowerShell, not bash

`.env` contains `PGPASSWORD=Lepa12<>!`. The `<>` characters are I/O redirect operators in bash. Running `source .env` or `set -o allexport; source .env` silently corrupts the value — Postgres connection fails with auth error that looks unrelated.

PowerShell string assignment is literal — `<>` is safe.

**Never use `scripts/start-api.sh` or `bash source .env`.**

---

## What the script does

1. Loads all vars from `.env` (split on first `=` — safe for `<>`)
2. Forces `APP_PORT=8081` (binary defaults to 8080 if this var is missing)
3. Kills any existing process on `:8081`
4. Builds `metaldocs-api.exe` if binary is missing
5. Starts the binary

Pass `-Build` to force rebuild:
```powershell
.\scripts\start-api.ps1 -Build
```

---

## docgen-v2 (token substitution service)

Required for document approval to produce frozen DOCX artifacts. Without it, approval succeeds but no DOCX is written to MinIO.

**Setup (first time):**
```powershell
# 1. Copy env template
Copy-Item .env.docgen-v2.example .env.docgen-v2
# 2. Fill in MinIO creds (dev: minioadmin/minioadmin) and set DOCGEN_V2_SERVICE_TOKEN
# 3. Install dependencies
cd apps/docgen-v2 && npm install
```

**Start:**
```powershell
.\scripts\dev-docgen.ps1   # runs on port 3001
```

**Wire to API:** In `.env`, set:
```
METALDOCS_FANOUT_URL=http://localhost:3001
METALDOCS_DOCGEN_V2_SERVICE_TOKEN=<same value as DOCGEN_V2_SERVICE_TOKEN in .env.docgen-v2>
```

**Health check:** `GET http://localhost:3001/health` → `{"status":"ok"}`

---

## Credentials

| field | value |
|---|---|
| Login endpoint | `POST /api/v1/auth/login` |
| Body field | `identifier` (NOT `username`) |
| identifier | `admin` |
| password | `AdminMetalDocs123!` |

Bootstrap creates this user automatically on first start when no admin role exists in DB.

**To reset / re-bootstrap:**
```sql
-- Run via: docker exec metaldocs-postgres psql -U metaldocs_app -d metaldocs -c "<query>"
TRUNCATE metaldocs.auth_sessions CASCADE;
TRUNCATE metaldocs.auth_identities CASCADE;
TRUNCATE metaldocs.iam_user_roles CASCADE;
TRUNCATE metaldocs.iam_users CASCADE;
```
Then restart API — bootstrap recreates `admin`.

---

## DB access

```powershell
docker exec metaldocs-postgres psql -U metaldocs_app -d metaldocs -c "SELECT 1;"
```

Port: `5433` (host) → `5432` (container). DB: `metaldocs`. Schema split:
- `metaldocs.*` — users, auth, IAM
- `public.*` — documents, templates, approvals

---

## Worker (PDF generation)

Required for PDF generation after document approval. Polls `messaging_outbox` every 10s, calls docgen-v2 `/convert/pdf`, writes `final_pdf_s3_key` to DB.

**Start (separate terminal, after API + docgen-v2 are up):**
```powershell
.\scripts\start-worker.ps1        # uses existing metaldocs-worker.exe
.\scripts\start-worker.ps1 -Build # rebuild binary first
```

**Verify running:** worker logs `MetalDocs Worker running (poll_interval_s=10 ...)` on start, then `worker_batch result=completed ...` every 10s.

**Env vars required (already in `.env`):**
- `METALDOCS_DOCGEN_V2_URL=http://localhost:3001`
- `METALDOCS_DOCGEN_V2_SERVICE_TOKEN=dev-local-service-token-32chars!!`

**If PDF not generated after signoff:** check worker log for `event_type=docgen_v2_pdf result=published`. If missing, event may not have been dispatched — check `METALDOCS_FANOUT_URL` is set (required for pdfDispatchAdapter to be wired).

---

## Common mistakes

| Mistake | Symptom | Fix |
|---|---|---|
| Used bash to source .env | `pq: password authentication failed` | Use PS script |
| Missing `APP_PORT` | API starts on :8080 not :8081 | Script sets it explicitly |
| Old process on :8081 | `bind: only one usage...` | Script kills it automatically |
| Wrong login body field | `AUTH_INVALID_CREDENTIALS` | Use `identifier`, not `username` |
| Bootstrap skipped | Can't create admin | Truncate iam_user_roles + restart |
