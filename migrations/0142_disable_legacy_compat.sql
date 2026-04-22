-- migrations/0142_disable_legacy_compat.sql
-- Phase 5.10: remove the compat window from the document transition trigger.
--
-- Migration 0133 hardcoded two legacy transitions into enforce_document_transition():
--   (OLD.status = 'draft'     AND NEW.status = 'finalized')
--   (OLD.status = 'finalized' AND NEW.status = 'archived')
-- These were allowed during Phase 1-4 so pre-Spec-2 code kept working.
-- This migration recreates the function WITHOUT those clauses.
--
-- PREREQUISITE: run CutoverService.ValidateLegacyCutoverReady before applying
-- this migration. It verifies no documents remain with status 'finalized' or
-- 'archived'. Applying this migration with such documents still present will
-- strand them permanently (no valid transition out).

BEGIN;

CREATE OR REPLACE FUNCTION enforce_document_transition() RETURNS trigger AS $$
BEGIN
  IF OLD.status IS DISTINCT FROM NEW.status THEN
    IF NOT (
      -- Spec 2 graph (sole valid transitions after Phase 5 cutover)
      (OLD.status = 'draft'        AND NEW.status =  'under_review') OR
      (OLD.status = 'under_review' AND NEW.status IN ('approved','rejected')) OR
      (OLD.status = 'rejected'     AND NEW.status =  'draft') OR
      (OLD.status = 'approved'     AND NEW.status IN ('published','scheduled','draft')) OR
      (OLD.status = 'scheduled'    AND NEW.status IN ('published','draft')) OR
      (OLD.status = 'published'    AND NEW.status IN ('superseded','obsolete')) OR
      (OLD.status = 'superseded'   AND NEW.status =  'obsolete')
    ) THEN
      RAISE EXCEPTION 'illegal status transition % -> %', OLD.status, NEW.status
        USING ERRCODE = 'check_violation';
    END IF;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- The trigger binding is unchanged; recreating the function is sufficient
-- because PostgreSQL resolves the function by OID at trigger fire time.
-- Re-attaching it here makes the intent explicit and self-documenting.
DROP TRIGGER IF EXISTS trg_documents_v2_legal_transition ON documents;
CREATE TRIGGER trg_documents_v2_legal_transition
  BEFORE UPDATE ON documents
  FOR EACH ROW EXECUTE FUNCTION enforce_document_transition();

COMMIT;
