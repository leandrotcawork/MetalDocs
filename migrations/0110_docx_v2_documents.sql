-- 0110_docx_v2_documents.sql
-- Documents vertical schema for docx-editor platform (W3).
-- Depends on 0109 (templates_v2). Safe to run in one transaction.

BEGIN;

-- Drop W1 scaffold stubs (0104-0107) that reference documents_v2.
-- These were placeholders; W3 provides the real schema referencing `documents`.
DROP TABLE IF EXISTS document_checkpoints CASCADE;
DROP TABLE IF EXISTS autosave_pending_uploads CASCADE;
DROP TABLE IF EXISTS document_revisions CASCADE;
DROP TABLE IF EXISTS editor_sessions CASCADE;

CREATE TABLE documents (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id             UUID NOT NULL,
  template_version_id   UUID NOT NULL REFERENCES template_versions(id),
  name                  TEXT NOT NULL,
  status                TEXT NOT NULL CHECK (status IN ('draft','finalized','archived')),
  form_data_json        JSONB NOT NULL,
  current_revision_id   UUID,
  active_session_id     UUID,
  finalized_at          TIMESTAMPTZ,
  archived_at           TIMESTAMPTZ,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by            UUID NOT NULL
);

CREATE INDEX idx_documents_tenant_status ON documents (tenant_id, status);
CREATE INDEX idx_documents_template_version ON documents (template_version_id);
CREATE INDEX idx_documents_form_data_gin ON documents USING GIN (form_data_json jsonb_path_ops);

CREATE TABLE editor_sessions (
  id                              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id                     UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  user_id                         UUID NOT NULL,
  acquired_at                     TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at                      TIMESTAMPTZ NOT NULL,
  released_at                     TIMESTAMPTZ,
  last_acknowledged_revision_id   UUID NOT NULL,
  status                          TEXT NOT NULL CHECK (status IN ('active','expired','released','force_released'))
);

-- Single-writer invariant: only ONE active session per document.
CREATE UNIQUE INDEX idx_one_active_session_per_doc
  ON editor_sessions (document_id) WHERE status = 'active';

CREATE TABLE document_revisions (
  id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id            UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  revision_num           BIGSERIAL,
  parent_revision_id     UUID REFERENCES document_revisions(id),
  session_id             UUID NOT NULL REFERENCES editor_sessions(id),
  storage_key            TEXT NOT NULL,
  content_hash           TEXT NOT NULL,
  form_data_snapshot     JSONB,
  created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (document_id, content_hash)
);

CREATE INDEX idx_revisions_doc_num ON document_revisions (document_id, revision_num DESC);

-- Deferrable FKs so we can insert document+session+revision in one tx.
ALTER TABLE documents
  ADD CONSTRAINT fk_current_revision
    FOREIGN KEY (current_revision_id) REFERENCES document_revisions(id)
    DEFERRABLE INITIALLY DEFERRED,
  ADD CONSTRAINT fk_active_session
    FOREIGN KEY (active_session_id) REFERENCES editor_sessions(id)
    DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE autosave_pending_uploads (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id           UUID NOT NULL REFERENCES editor_sessions(id) ON DELETE CASCADE,
  document_id          UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  base_revision_id     UUID NOT NULL REFERENCES document_revisions(id),
  content_hash         TEXT NOT NULL,
  storage_key          TEXT NOT NULL,
  presigned_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at           TIMESTAMPTZ NOT NULL,
  consumed_at          TIMESTAMPTZ,
  UNIQUE (session_id, base_revision_id, content_hash)
);

CREATE INDEX idx_pending_expired
  ON autosave_pending_uploads (expires_at)
  WHERE consumed_at IS NULL;

CREATE TABLE document_checkpoints (
  id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id        UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  revision_id        UUID NOT NULL REFERENCES document_revisions(id),
  version_num        INT NOT NULL,
  label              TEXT,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by         UUID NOT NULL,
  UNIQUE (document_id, version_num)
);

COMMIT;
