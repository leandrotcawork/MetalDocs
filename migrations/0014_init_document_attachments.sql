CREATE TABLE IF NOT EXISTS metaldocs.document_attachments (
  id TEXT PRIMARY KEY,
  document_id TEXT NOT NULL REFERENCES metaldocs.documents(id) ON DELETE RESTRICT,
  file_name TEXT NOT NULL,
  content_type TEXT NOT NULL,
  size_bytes BIGINT NOT NULL CHECK (size_bytes > 0),
  storage_key TEXT NOT NULL UNIQUE,
  uploaded_by TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_document_attachments_document_id_created_at
  ON metaldocs.document_attachments (document_id, created_at DESC);
