CREATE TABLE IF NOT EXISTS metaldocs.document_types (
  code TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  review_interval_days INT NOT NULL CHECK (review_interval_days > 0),
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE metaldocs.document_types
  ADD COLUMN IF NOT EXISTS type_key TEXT,
  ADD COLUMN IF NOT EXISTS family_key TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS active_version INTEGER NOT NULL DEFAULT 1,
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE metaldocs.document_types
SET
  type_key = COALESCE(NULLIF(type_key, ''), code),
  family_key = COALESCE(
    NULLIF(family_key, ''),
    CASE
      WHEN code IN ('policy', 'procedure', 'po') THEN 'procedure'
      WHEN code IN ('work_instruction', 'it') THEN 'work_instruction'
      WHEN code IN ('record', 'rg') THEN 'record'
      ELSE 'record'
    END
  ),
  active_version = COALESCE(active_version, 1)
WHERE type_key IS NULL
   OR type_key = ''
   OR family_key IS NULL
   OR family_key = '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_document_types_type_key_unique
  ON metaldocs.document_types (type_key);

CREATE TABLE IF NOT EXISTS metaldocs.document_type_schema_versions (
  type_key TEXT NOT NULL REFERENCES metaldocs.document_types(type_key),
  version INTEGER NOT NULL,
  schema_json JSONB NOT NULL,
  governance_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (type_key, version)
);

ALTER TABLE metaldocs.documents
  ADD COLUMN IF NOT EXISTS document_type_key TEXT,
  ADD COLUMN IF NOT EXISTS document_type_version INTEGER NOT NULL DEFAULT 1;

UPDATE metaldocs.documents
SET document_type_key = COALESCE(NULLIF(document_type_key, ''), document_type_code)
WHERE document_type_key IS NULL
   OR document_type_key = '';

ALTER TABLE metaldocs.documents
  ALTER COLUMN document_type_key SET NOT NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'fk_documents_document_type_key'
  ) THEN
    ALTER TABLE metaldocs.documents
      ADD CONSTRAINT fk_documents_document_type_key
      FOREIGN KEY (document_type_key)
      REFERENCES metaldocs.document_types(type_key);
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_documents_document_type_key
  ON metaldocs.documents (document_type_key);

ALTER TABLE metaldocs.document_versions
  ADD COLUMN IF NOT EXISTS values_json JSONB NOT NULL DEFAULT '{}'::jsonb;
