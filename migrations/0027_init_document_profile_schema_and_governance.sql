CREATE TABLE IF NOT EXISTS metaldocs.document_profile_schema_versions (
  profile_code TEXT NOT NULL REFERENCES metaldocs.document_profiles(code),
  version INT NOT NULL CHECK (version > 0),
  metadata_rules_json JSONB NOT NULL DEFAULT '[]'::jsonb,
  is_active BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (profile_code, version)
);

CREATE TABLE IF NOT EXISTS metaldocs.document_profile_governance (
  profile_code TEXT PRIMARY KEY REFERENCES metaldocs.document_profiles(code),
  workflow_profile TEXT NOT NULL DEFAULT 'standard_approval',
  review_interval_days INT NOT NULL CHECK (review_interval_days > 0),
  approval_required BOOLEAN NOT NULL DEFAULT TRUE,
  retention_days INT NOT NULL DEFAULT 0 CHECK (retention_days >= 0),
  validity_days INT NOT NULL DEFAULT 0 CHECK (validity_days >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO metaldocs.document_profile_schema_versions (profile_code, version, metadata_rules_json, is_active)
VALUES
  ('contract', 1, '[{"name":"counterparty","type":"string","required":true},{"name":"contract_number","type":"string","required":true},{"name":"start_date","type":"date","required":true},{"name":"end_date","type":"date","required":true}]'::jsonb, TRUE),
  ('certificate', 1, '[{"name":"issuer","type":"string","required":true},{"name":"issue_date","type":"date","required":true},{"name":"expiry_date","type":"date","required":true}]'::jsonb, TRUE),
  ('technical_drawing', 1, '[{"name":"drawing_code","type":"string","required":true},{"name":"revision_code","type":"string","required":true},{"name":"plant","type":"string","required":true}]'::jsonb, TRUE),
  ('supplier_document', 1, '[{"name":"supplier_name","type":"string","required":true},{"name":"supplier_document_code","type":"string","required":true}]'::jsonb, TRUE),
  ('policy', 1, '[{"name":"policy_code","type":"string","required":true}]'::jsonb, TRUE),
  ('procedure', 1, '[{"name":"procedure_code","type":"string","required":true}]'::jsonb, TRUE),
  ('work_instruction', 1, '[{"name":"instruction_code","type":"string","required":true}]'::jsonb, TRUE),
  ('report', 1, '[{"name":"report_period","type":"string","required":true}]'::jsonb, TRUE),
  ('form', 1, '[{"name":"form_code","type":"string","required":true}]'::jsonb, TRUE),
  ('manual', 1, '[{"name":"manual_code","type":"string","required":true}]'::jsonb, TRUE)
ON CONFLICT (profile_code, version) DO NOTHING;

INSERT INTO metaldocs.document_profile_governance (
  profile_code, workflow_profile, review_interval_days, approval_required, retention_days, validity_days
)
VALUES
  ('policy', 'standard_approval', 365, TRUE, 0, 0),
  ('procedure', 'standard_approval', 365, TRUE, 0, 0),
  ('work_instruction', 'standard_approval', 180, TRUE, 0, 0),
  ('contract', 'standard_approval', 365, TRUE, 3650, 0),
  ('supplier_document', 'standard_approval', 180, TRUE, 3650, 0),
  ('technical_drawing', 'standard_approval', 180, TRUE, 0, 0),
  ('certificate', 'standard_approval', 365, TRUE, 3650, 365),
  ('report', 'standard_approval', 365, TRUE, 3650, 0),
  ('form', 'standard_approval', 180, TRUE, 3650, 0),
  ('manual', 'standard_approval', 365, TRUE, 0, 0)
ON CONFLICT (profile_code) DO NOTHING;

ALTER TABLE metaldocs.documents
  ADD COLUMN IF NOT EXISTS profile_schema_version INT;

UPDATE metaldocs.documents d
SET profile_schema_version = COALESCE(
  d.profile_schema_version,
  (
    SELECT version
    FROM metaldocs.document_profile_schema_versions s
    WHERE s.profile_code = d.document_profile_code
      AND s.is_active = TRUE
    ORDER BY version DESC
    LIMIT 1
  ),
  1
)
WHERE d.profile_schema_version IS NULL;

ALTER TABLE metaldocs.documents
  ALTER COLUMN profile_schema_version SET NOT NULL;

ALTER TABLE metaldocs.documents
  ALTER COLUMN profile_schema_version SET DEFAULT 1;

CREATE INDEX IF NOT EXISTS idx_documents_profile_schema_version ON metaldocs.documents (document_profile_code, profile_schema_version);
CREATE INDEX IF NOT EXISTS idx_document_profile_schema_versions_active ON metaldocs.document_profile_schema_versions (profile_code, is_active);
