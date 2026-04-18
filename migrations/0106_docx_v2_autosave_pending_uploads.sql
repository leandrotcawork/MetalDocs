-- 0106_docx_v2_autosave_pending_uploads.sql
BEGIN;

CREATE TABLE IF NOT EXISTS autosave_pending_uploads (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id           UUID NOT NULL REFERENCES editor_sessions(id) ON DELETE CASCADE,
  document_id          UUID NOT NULL REFERENCES documents_v2(id) ON DELETE CASCADE,
  base_revision_id     UUID NOT NULL REFERENCES document_revisions(id),
  content_hash         TEXT NOT NULL,
  storage_key          TEXT NOT NULL,
  presigned_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at           TIMESTAMPTZ NOT NULL,
  consumed_at          TIMESTAMPTZ,
  CONSTRAINT autosave_pending_uniq
    UNIQUE (session_id, base_revision_id, content_hash)
);

CREATE INDEX IF NOT EXISTS idx_pending_expired
  ON autosave_pending_uploads (expires_at) WHERE consumed_at IS NULL;

COMMIT;
