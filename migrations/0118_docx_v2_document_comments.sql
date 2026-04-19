CREATE TABLE document_comments (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id            UUID NOT NULL,
  document_id          UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  library_comment_id   INTEGER NOT NULL,
  parent_library_id    INTEGER,
  author_id            TEXT NOT NULL,
  author_display       TEXT NOT NULL,
  content_json         JSONB NOT NULL,
  resolved_at          TIMESTAMPTZ,
  resolved_by          TEXT,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_document_comments_doc_lib ON document_comments(document_id, library_comment_id);
CREATE INDEX idx_document_comments_doc ON document_comments(document_id, created_at);
CREATE INDEX idx_document_comments_parent ON document_comments(document_id, parent_library_id) WHERE parent_library_id IS NOT NULL;
