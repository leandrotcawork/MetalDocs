INSERT INTO metaldocs.document_profile_schema_versions (
  profile_code, version, metadata_rules_json, is_active
)
VALUES
  ('po', 1, '[]'::jsonb, TRUE),
  ('it', 1, '[]'::jsonb, TRUE),
  ('rg', 1, '[]'::jsonb, TRUE)
ON CONFLICT (profile_code, version)
DO UPDATE SET
  metadata_rules_json = EXCLUDED.metadata_rules_json,
  is_active = TRUE;
