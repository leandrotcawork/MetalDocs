-- 0104_docx_v2_editor_sessions.sql
BEGIN;

CREATE TABLE IF NOT EXISTS editor_sessions (
  id                              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id                     UUID NOT NULL REFERENCES documents_v2(id) ON DELETE CASCADE,
  user_id                         UUID NOT NULL,
  acquired_at                     TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at                      TIMESTAMPTZ NOT NULL,
  released_at                     TIMESTAMPTZ,
  last_acknowledged_revision_id   UUID NOT NULL,
  status                          TEXT NOT NULL CHECK (status IN ('active','expired','released','force_released'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_one_active_session_per_doc
  ON editor_sessions (document_id) WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_editor_sessions_expires
  ON editor_sessions (expires_at) WHERE status = 'active';

COMMIT;
