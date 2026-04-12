-- 0072_add_release_artifact_columns.sql
-- Stores immutable DOCX artifact key and canonical MDDM snapshot at release time.
-- These columns support release determinism: old released docs can always re-export
-- the exact same DOCX without re-rendering from potentially-migrated MDDM data.

ALTER TABLE metaldocs.document_versions
  ADD COLUMN IF NOT EXISTS release_artifact_key TEXT,
  ADD COLUMN IF NOT EXISTS canonical_mddm_snapshot JSONB;

COMMENT ON COLUMN metaldocs.document_versions.release_artifact_key IS
  'Storage key for the immutable DOCX artifact generated at release time';
COMMENT ON COLUMN metaldocs.document_versions.canonical_mddm_snapshot IS
  'Frozen MDDM envelope JSON captured at release time (post-migration, post-canonicalization)';
