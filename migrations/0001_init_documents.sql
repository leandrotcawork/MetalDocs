-- MetalDocs initial schema (documents + versions)

CREATE SCHEMA IF NOT EXISTS metaldocs;

CREATE TABLE IF NOT EXISTS metaldocs.documents (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  classification TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_documents_owner_id ON metaldocs.documents (owner_id);
CREATE INDEX IF NOT EXISTS idx_documents_created_at ON metaldocs.documents (created_at DESC);

CREATE TABLE IF NOT EXISTS metaldocs.document_versions (
  document_id TEXT NOT NULL REFERENCES metaldocs.documents(id) ON DELETE RESTRICT,
  version_number INT NOT NULL,
  content TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (document_id, version_number)
);

CREATE INDEX IF NOT EXISTS idx_document_versions_created_at ON metaldocs.document_versions (created_at DESC);
