# MDDM Operations Runbook

## Backups

MDDM stores everything in PostgreSQL: documents, templates, images (as bytea), draft state, and all audit data.

### Backup strategy

Use `pg_dump` (logical) or `pg_basebackup` (physical) on a regular schedule. Both capture the entire MDDM state in one operation.

```bash
# Logical backup
pg_dump -U metaldocs_app -d metaldocs -F c -f /backups/metaldocs-$(date +%Y%m%d).dump

# Physical backup
pg_basebackup -D /backups/metaldocs-$(date +%Y%m%d) -F t -X stream -P -U metaldocs_app
```

### Restore validation

After restoring, run:

```sql
-- Verify all referenced images exist
SELECT count(*) FROM metaldocs.document_version_images dvi
LEFT JOIN metaldocs.document_images i ON dvi.image_id = i.id
WHERE i.id IS NULL;
-- Expected: 0
```

## Image storage migration to S3

When v2 swaps from PostgresByteaStorage to S3Storage:

1. Set env: `MDDM_IMAGE_STORAGE=postgres_bytea` (still v1)
2. Run a one-time migration job: walk `document_images` rows, upload bytes to S3 keyed by `id` or `sha256`, verify
3. Set env: `MDDM_IMAGE_STORAGE=s3`
4. Restart the backend
5. After validation period: drop the `bytes` column from `document_images` (still keep id, sha256, mime_type, byte_size for indexing)

## Template repair / rebind

If a `TEMPLATE_SNAPSHOT_MISMATCH` is detected (template content_hash doesn't match what the document expects):

1. Identify the affected document via the structured error log (`document_id` field)
2. Investigate WHY the hash differs (DB corruption, buggy migration, manual edit)
3. Either:
   - Restore the original template version from backup
   - OR (admin only) explicitly rebind the document to a different template version via the rebind admin endpoint (Phase 2 feature)

## DOCX re-render (rare)

If a renderer bug requires regenerating historical DOCX bytes:

1. Use the admin re-render endpoint (Phase 2 feature)
2. Each re-render is logged as a special audit event with the reason
3. Never used in normal flow
