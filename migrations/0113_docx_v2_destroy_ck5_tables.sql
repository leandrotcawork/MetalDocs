-- 0113_docx_v2_destroy_ck5_tables.sql
-- DESTRUCTIVE. Drops every CK5 / MDDM / old-docgen table.
-- Guarded by sentinel assertions on the W4-era schema; if ANY W4 table is
-- missing the migration aborts before executing a single DROP.
-- Idempotent: a successful prior apply is recorded in `schema_migrations`.
-- A re-run detects the ledger entry and short-circuits to a clean no-op
-- COMMIT so an accidental re-apply during a deploy cycle does NOT fail.
-- Depends on: 0101-0112 applied and Task 2 (w5_cutover_gate) green.

BEGIN;

-- Informational NOTICE for operator feedback on re-run.
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM public.schema_migrations WHERE version = '0113') THEN
    RAISE NOTICE '0113 already recorded in schema_migrations — running as no-op';
  END IF;
END $$;

-- Sentinel: every W4-era table must exist. Missing any of these aborts.
-- Skipped on re-run (ledger already has 0113).
DO $$
DECLARE
  required_tables TEXT[] := ARRAY[
    'templates', 'template_versions',
    'documents', 'document_revisions', 'document_checkpoints',
    'editor_sessions', 'autosave_pending_uploads',
    'document_exports', 'template_audit_log'
  ];
  t TEXT;
BEGIN
  IF EXISTS (SELECT 1 FROM public.schema_migrations WHERE version = '0113') THEN
    RETURN;
  END IF;
  FOREACH t IN ARRAY required_tables LOOP
    IF NOT EXISTS (
      SELECT 1 FROM information_schema.tables
       WHERE table_schema = 'public' AND table_name = t
    ) THEN
      RAISE EXCEPTION
        'W4-era table %.% missing — refusing to run destructive migration 0113',
        'public', t;
    END IF;
  END LOOP;
END $$;

-- Drop CK5 / MDDM / legacy-docgen tables. CASCADE handles FKs from dead tables
-- onto other dead tables; W4 tables were audited to ensure NONE of them FK
-- into these legacy tables (confirmed in Plan B Task 2 + Plan C Task 1).
-- Wrapped in a DO block so we can ledger-guard the whole thing.
DO $$
DECLARE
  kill_tables TEXT[] := ARRAY[
    'mddm_shadow_diff_events','mddm_audit_events','mddm_releases',
    'mddm_block_versions','mddm_blocks','mddm_templates',
    'template_drafts','document_template_versions_audit',
    'document_template_versions','document_templates_ck5',
    'document_profile_schemas','document_collaboration',
    'document_departments','document_profile_registry',
    'document_family','document_taxonomy','document_type_runtime',
    -- Legacy `documents` + `document_versions` (pre-pivot). Name MUST differ
    -- from the greenfield `documents` table — verified in Task 4 Step 1.
    -- If W1 suffixed legacy tables with `_legacy`, drop those.
    'documents_legacy','document_versions_legacy','document_versions',
    'blocks_json_artifacts','renderer_pin_events',
    'rich_envelope_events','shadow_diff_events'
  ];
  t TEXT;
BEGIN
  IF EXISTS (SELECT 1 FROM public.schema_migrations WHERE version = '0113') THEN
    RETURN;
  END IF;
  FOREACH t IN ARRAY kill_tables LOOP
    EXECUTE format('DROP TABLE IF EXISTS public.%I CASCADE', t);
  END LOOP;
END $$;

-- Post-drop assertion: every W4-era table still exists. Skipped on re-run.
DO $$
DECLARE
  required_tables TEXT[] := ARRAY[
    'templates', 'template_versions',
    'documents', 'document_revisions', 'document_checkpoints',
    'editor_sessions', 'autosave_pending_uploads',
    'document_exports', 'template_audit_log'
  ];
  t TEXT;
BEGIN
  IF EXISTS (SELECT 1 FROM public.schema_migrations WHERE version = '0113') THEN
    RETURN;
  END IF;
  FOREACH t IN ARRAY required_tables LOOP
    IF NOT EXISTS (
      SELECT 1 FROM information_schema.tables
       WHERE table_schema = 'public' AND table_name = t
    ) THEN
      RAISE EXCEPTION
        'CATASTROPHE: W4-era table %.% went missing during 0113 — ABORT',
        'public', t;
    END IF;
  END LOOP;
END $$;

-- Record 0113 as applied. Safe on re-run (ON CONFLICT DO NOTHING).
INSERT INTO public.schema_migrations (version, description)
VALUES ('0113', 'destroy CK5/MDDM/legacy-docgen tables (docx-v2/w5)')
ON CONFLICT (version) DO NOTHING;

COMMIT;
