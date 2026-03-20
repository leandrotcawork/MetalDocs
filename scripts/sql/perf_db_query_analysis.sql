-- Performance DB analysis for profiles/schemas/metadata.
-- Update the variables below before running.
\set profile_code 'po'
\set document_id 'CHANGE_ME_DOCUMENT_ID'

-- 1) ListDocumentProfiles query (from repository.go)
EXPLAIN (ANALYZE, BUFFERS, VERBOSE)
SELECT p.code,
       p.family_code,
       p.name,
       p.alias,
       p.description,
       COALESCE(g.review_interval_days, p.review_interval_days) AS review_interval_days,
       COALESCE(s.active_version, 1) AS active_schema_version,
       COALESCE(g.workflow_profile, 'standard_approval') AS workflow_profile,
       COALESCE(g.approval_required, TRUE) AS approval_required,
       COALESCE(g.retention_days, 0) AS retention_days,
       COALESCE(g.validity_days, 0) AS validity_days
FROM metaldocs.document_profiles p
LEFT JOIN (
  SELECT profile_code, MAX(version) FILTER (WHERE is_active) AS active_version
  FROM metaldocs.document_profile_schema_versions
  GROUP BY profile_code
) s ON s.profile_code = p.code
LEFT JOIN metaldocs.document_profile_governance g ON g.profile_code = p.code
WHERE p.is_active = TRUE
ORDER BY code ASC;

-- 2) ListDocumentProfileSchemas (filtered by profile)
EXPLAIN (ANALYZE, BUFFERS, VERBOSE)
SELECT profile_code, version, is_active, metadata_rules_json, content_schema_json
FROM metaldocs.document_profile_schema_versions
WHERE (:'profile_code' = '' OR profile_code = :'profile_code')
ORDER BY profile_code ASC, version ASC;

-- 3) GetDocument (metadata_json access)
EXPLAIN (ANALYZE, BUFFERS, VERBOSE)
SELECT id,
       title,
       document_type_code,
       document_profile_code,
       document_family_code,
       process_area_code,
       subject_code,
       profile_schema_version,
       owner_id,
       business_unit,
       department,
       classification,
       status,
       tags,
       effective_at,
       expiry_at,
       metadata_json,
       created_at,
       updated_at
FROM metaldocs.documents
WHERE id = :'document_id';

-- 4) Index inventory for related tables
SELECT schemaname,
       tablename,
       indexname,
       indexdef
FROM pg_indexes
WHERE schemaname = 'metaldocs'
  AND tablename IN (
    'document_profiles',
    'document_profile_schema_versions',
    'document_profile_governance',
    'documents',
    'document_versions'
  )
ORDER BY tablename, indexname;

-- 5) Table size + scan stats (optional)
SELECT relname,
       seq_scan,
       idx_scan,
       n_live_tup,
       n_dead_tup
FROM pg_stat_user_tables
WHERE schemaname = 'metaldocs'
  AND relname IN (
    'document_profiles',
    'document_profile_schema_versions',
    'document_profile_governance',
    'documents',
    'document_versions'
  )
ORDER BY relname;
