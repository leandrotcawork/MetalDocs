-- Clear legacy Carbone-era content schemas from profile schema versions.
-- The content-builder now uses document_type_schema_versions instead.
UPDATE metaldocs.document_profile_schema_versions
SET content_schema_json = '{}'::jsonb
WHERE profile_code IN ('po', 'it', 'rg');
