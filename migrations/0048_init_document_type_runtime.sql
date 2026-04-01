CREATE TABLE IF NOT EXISTS metaldocs.document_types (
  type_key TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  family_key TEXT NOT NULL DEFAULT '',
  active_version INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS metaldocs.document_type_schema_versions (
  type_key TEXT NOT NULL REFERENCES metaldocs.document_types(type_key),
  version INTEGER NOT NULL,
  schema_json JSONB NOT NULL,
  governance_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (type_key, version)
);

ALTER TABLE metaldocs.documents
  ADD COLUMN IF NOT EXISTS document_type_key TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS document_type_version INTEGER NOT NULL DEFAULT 1;

UPDATE metaldocs.documents
SET document_type_key = COALESCE(NULLIF(document_type_key, ''), document_type_code)
WHERE document_type_key IS NULL
   OR document_type_key = '';

CREATE INDEX IF NOT EXISTS idx_documents_document_type_key
  ON metaldocs.documents (document_type_key);

ALTER TABLE metaldocs.document_versions
  ADD COLUMN IF NOT EXISTS values_json JSONB NOT NULL DEFAULT '{}'::jsonb;
