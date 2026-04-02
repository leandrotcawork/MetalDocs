CREATE TABLE IF NOT EXISTS metaldocs.document_template_versions (
  template_key TEXT NOT NULL,
  version INTEGER NOT NULL,
  profile_code TEXT NOT NULL,
  schema_version INTEGER NOT NULL,
  name TEXT NOT NULL,
  definition_json JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (template_key, version)
);

CREATE TABLE IF NOT EXISTS metaldocs.document_profile_template_defaults (
  profile_code TEXT PRIMARY KEY,
  template_key TEXT NOT NULL,
  template_version INTEGER NOT NULL,
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS metaldocs.document_template_assignments (
  document_id TEXT PRIMARY KEY REFERENCES metaldocs.documents(id) ON DELETE CASCADE,
  template_key TEXT NOT NULL,
  template_version INTEGER NOT NULL,
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE metaldocs.document_versions
  ADD COLUMN IF NOT EXISTS template_key TEXT,
  ADD COLUMN IF NOT EXISTS template_version INTEGER;

GRANT SELECT, INSERT, UPDATE ON TABLE metaldocs.document_template_versions TO metaldocs_app;
GRANT SELECT, INSERT, UPDATE ON TABLE metaldocs.document_profile_template_defaults TO metaldocs_app;
GRANT SELECT, INSERT, UPDATE ON TABLE metaldocs.document_template_assignments TO metaldocs_app;
GRANT SELECT, INSERT, UPDATE ON TABLE metaldocs.document_versions TO metaldocs_app;
