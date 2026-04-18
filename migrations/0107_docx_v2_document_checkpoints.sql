-- 0107_docx_v2_document_checkpoints.sql
BEGIN;

CREATE TABLE IF NOT EXISTS document_checkpoints (
  id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id        UUID NOT NULL REFERENCES documents_v2(id) ON DELETE CASCADE,
  revision_id        UUID NOT NULL REFERENCES document_revisions(id),
  version_num        INT NOT NULL,
  label              TEXT,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by         UUID NOT NULL,
  CONSTRAINT document_checkpoints_doc_num_unique UNIQUE (document_id, version_num)
);

CREATE INDEX IF NOT EXISTS idx_checkpoints_doc
  ON document_checkpoints (document_id, version_num DESC);

COMMIT;
