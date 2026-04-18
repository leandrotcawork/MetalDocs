# W4 Exports Runbook

## Overview

W4 adds PDF export (`POST /api/v2/documents/{id}/export/pdf`) and DOCX signed-URL
(`GET /api/v2/documents/{id}/export/docx-url`) to the docx-v2 platform. PDFs are
generated via docgen-v2 → Gotenberg and cached by composite hash (SHA-256 over
content hash + template version + grammar version + docgen version + render opts).

---

## New Infrastructure

| Component | Purpose |
|---|---|
| `document_exports` table (migration `0111`) | Append-only PDF cache ledger |
| `ExportService` (`application/export_service.go`) | Cache probe, docgen call, audit |
| `ExportHandler` (`delivery/http/export_handler.go`) | HTTP layer, rate-limit wiring |
| docgen-v2 `/convert-pdf` route | DOCX → PDF via Gotenberg |
| Rate-limit middleware (`platform/ratelimit`) | 20/min per user for export/pdf |

---

## Deployment Checklist

1. **Apply migration**: `psql -f migrations/0111_docx_v2_exports.sql`
2. **Verify table**: `SELECT to_regclass('public.document_exports');` → `document_exports`
3. **Verify index**: `SELECT indexname FROM pg_indexes WHERE indexname = 'uq_document_exports_doc_hash';` → 1 row
4. **Restart docgen-v2**: requires `DOCGEN_V2_GOTENBERG_URL` in env (default `http://gotenberg:3000`)
5. **Restart metaldocs-api**: new routes registered in `module.go`
6. **Smoke**: `curl -X POST /api/v2/documents/{id}/export/pdf` → 200 with `signed_url`

---

## Rate Limits

| Route | Limit |
|---|---|
| `POST /export/pdf` | 20 req/min per user |
| `POST /autosave/presign` | 60 req/min per user |
| `POST /autosave/commit` | 30 req/min per user |

429 response body: `{"error":"rate_limited","retry_after_seconds":<n>}`

---

## Composite Hash Debugging

To determine whether a PDF will be served from cache:

```sql
SELECT id, storage_key, size_bytes, created_at
FROM document_exports
WHERE document_id = '<uuid>'
ORDER BY created_at DESC
LIMIT 5;
```

If the row exists but the S3 object is missing (e.g. after bucket wipe), the
`ExportService` probes S3 with `HeadObject` and falls through to re-generate.

---

## Audit Events

Every `POST /export/pdf` call emits an `export.pdf_generated` audit event regardless
of cache status. Field `cached: true/false` allows cache-miss ratio alerting.

Query in audit log:

```sql
SELECT payload->>'cached', count(*)
FROM audit_log
WHERE event_type = 'export.pdf_generated'
  AND created_at > now() - interval '1 hour'
GROUP BY 1;
```

---

## Rollback

1. Revert `metaldocs-api` to previous release (routes become 404).
2. Revert `docgen-v2` to previous release.
3. Migration is additive (no column drops) — table can stay; drop manually if needed:
   ```sql
   DROP TABLE IF EXISTS document_exports;
   ```
4. No frontend feature flag — `ExportMenu` renders only when `canExport=true`
   (controlled by session phase). Disable by removing `ExportMenu` render in
   `DocumentEditorPage.tsx` if necessary.

---

## Alerts to Configure

| Alert | Condition | Action |
|---|---|---|
| PDF cache miss ratio > 80% | `cached=false` rate > 80% in 5-min window | Investigate docgen-v2 errors |
| Export 429 spike | >5% of export requests return 429 | Raise rate limit or investigate bot |
| Gotenberg latency P99 > 30s | docgen-v2 `/convert-pdf` slow | Scale Gotenberg or check LibreOffice |
