-- 0105_docx_v2_document_revisions.sql
BEGIN;

CREATE TABLE IF NOT EXISTS document_revisions (
  id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id            UUID NOT NULL REFERENCES documents_v2(id) ON DELETE CASCADE,
  revision_num           BIGSERIAL,
  parent_revision_id     UUID REFERENCES document_revisions(id),
  session_id             UUID NOT NULL REFERENCES editor_sessions(id),
  storage_key            TEXT NOT NULL,
  content_hash           TEXT NOT NULL,
  form_data_snapshot     JSONB,
  created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT document_revisions_doc_hash_unique UNIQUE (document_id, content_hash)
);

CREATE INDEX IF NOT EXISTS idx_revisions_doc_num
  ON document_revisions (document_id, revision_num DESC);

ALTER TABLE documents_v2
  ADD CONSTRAINT fk_documents_v2_current_revision
    FOREIGN KEY (current_revision_id) REFERENCES document_revisions(id)
    DEFERRABLE INITIALLY IMMEDIATE,
  ADD CONSTRAINT fk_documents_v2_active_session
    FOREIGN KEY (active_session_id) REFERENCES editor_sessions(id)
    DEFERRABLE INITIALLY IMMEDIATE;

COMMIT;
