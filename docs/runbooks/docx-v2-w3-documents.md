# Runbook: docx-v2 W3 Documents Vertical

## Route map + RBAC (admin vs document_filler)

All routes are under `/api/v2/documents` and require auth headers (`X-Tenant-ID`, `X-User-ID`, `X-User-Roles`).
`document_filler` access is ownership-scoped unless otherwise noted.

| # | Method | Path | admin | document_filler |
|---|--------|------|-------|-----------------|
| 1 | `POST` | `/api/v2/documents` | allow | allow |
| 2 | `GET` | `/api/v2/documents/{id}` | allow | allow (owner only) |
| 3 | `POST` | `/api/v2/documents/{id}/autosave/presign` | allow | allow (owner + active session) |
| 4 | `POST` | `/api/v2/documents/{id}/autosave/commit` | allow | allow (owner + session holder) |
| 5 | `POST` | `/api/v2/documents/{id}/session/acquire` | allow | allow (owner only) |
| 6 | `POST` | `/api/v2/documents/{id}/session/heartbeat` | allow | allow (owner + session holder) |
| 7 | `POST` | `/api/v2/documents/{id}/session/release` | allow | allow (owner + session holder) |
| 8 | `POST` | `/api/v2/documents/{id}/session/force-release` | allow | deny |
| 9 | `GET` + `POST` | `/api/v2/documents/{id}/checkpoints` | allow | allow (owner + active session for POST) |
| 10 | `POST` | `/api/v2/documents/{id}/checkpoints/{versionNum}/restore` | allow | allow (owner + active session) |
| 11 | `POST` | `/api/v2/documents/{id}/finalize` | allow | allow (owner only) |
| 12 | `POST` | `/api/v2/documents/{id}/archive` | allow | allow (owner-only draft/archive path) |
| 13 | `GET` | `/api/v2/documents` | allow | allow (filtered to own docs) |

## Autosave flow (presign vs commit)

1. `POST /autosave/presign` allocates pending upload metadata and returns PUT target information for S3/object storage.
2. Client uploads bytes to object storage.
3. `POST /autosave/commit` consumes the pending upload and advances head revision.

`commit` is server-authoritative for integrity: the server streams the uploaded object from S3 and recomputes SHA256 during commit, then compares against the expected hash captured at presign time.

Idempotency key for autosave commit is `(session_id, base_revision_id, content_hash)`. Replays with the same triple do not create duplicate logical commits.

## Checkpoint restore semantics

Checkpoint restore is forward-only. Restore never rewinds history in place; it creates a new head revision from the selected checkpoint payload.

Idempotency is enforced by `ON CONFLICT (document_id, content_hash)`. If head already matches the target content hash, restore resolves to the same effective revision.

Audit requirement: each restore writes `document.checkpoint_restored`.

## Session force-release (wedged session)

Route: `POST /api/v2/documents/{id}/session/force-release`

- Admin-only.
- Use when heartbeat or release is wedged and no writer can continue.
- After force-release, the active user should reload and reacquire session.

## Orphan cleanup

Two cleanup paths are expected:

1. CreateDocument deferred cleanup: temporary objects are deleted on create failure via deferred cleanup (`defer` path).
2. Pending upload sweeper: `StartOrphanPendingSweeper` removes orphan `pending_uploads`/objects older than 24h.

## Troubleshooting

| HTTP/code | Typical cause | Action |
|-----------|---------------|--------|
| `409 stale_base` | Two tabs editing same doc with diverged base revision | Reload losing tab, reacquire session, reapply edit |
| `410 expired_upload` | Presign window elapsed (15 min) before commit | Re-presign, re-upload, retry commit |
| `410 upload_missing` | Uploaded object removed before commit | Upload blob again and commit with fresh pending id |
| `422 content_hash_mismatch` | Corrupt/tampered upload bytes | Re-upload blob from source and retry |