UPDATE metaldocs.document_profile_schema_versions
SET metadata_rules_json = '[]'::jsonb
WHERE profile_code IN ('po', 'it', 'rg');
