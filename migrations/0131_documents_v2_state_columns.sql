-- migrations/0131_documents_v2_state_columns.sql
-- Spec 2 Phase 1. SUPERSET CHECK keeps pre-Spec-2 code alive until Phase 5
-- tightens it. Transition trigger installed in 0133 after legacy remap.

BEGIN;

ALTER TABLE documents
  DROP CONSTRAINT IF EXISTS documents_status_check,
  ADD  CONSTRAINT documents_status_check
    CHECK (status IN (
      'draft','finalized','archived',
      'under_review','approved','rejected',
      'scheduled','published','superseded','obsolete'
    ));

ALTER TABLE documents
  ADD COLUMN IF NOT EXISTS effective_from         TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS effective_to           TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS revision_number        INT NOT NULL DEFAULT 1,
  ADD COLUMN IF NOT EXISTS revision_version       INT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS locked_at              TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS content_hash_at_submit TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS ux_documents_v2_cd_revision
  ON documents (controlled_document_id, revision_number)
  WHERE controlled_document_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_documents_v2_cd_active
  ON documents (controlled_document_id)
  WHERE controlled_document_id IS NOT NULL
    AND status IN ('draft','under_review','approved','rejected','scheduled');

COMMIT;
