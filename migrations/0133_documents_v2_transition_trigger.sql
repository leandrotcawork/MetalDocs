-- migrations/0133_documents_v2_transition_trigger.sql

BEGIN;

CREATE OR REPLACE FUNCTION enforce_document_transition() RETURNS trigger AS $$
BEGIN
  IF OLD.status IS DISTINCT FROM NEW.status THEN
    IF NOT (
      -- Spec 2 graph
      (OLD.status = 'draft'        AND NEW.status =  'under_review') OR
      (OLD.status = 'under_review' AND NEW.status IN ('approved','rejected')) OR
      (OLD.status = 'rejected'     AND NEW.status =  'draft') OR
      (OLD.status = 'approved'     AND NEW.status IN ('published','scheduled','draft')) OR
      (OLD.status = 'scheduled'    AND NEW.status IN ('published','draft')) OR
      (OLD.status = 'published'    AND NEW.status IN ('superseded','obsolete')) OR
      (OLD.status = 'superseded'   AND NEW.status =  'obsolete') OR
      -- Compat window (Phase 1..4): legacy pre-Spec-2 writes. Removed in Phase 5.
      (OLD.status = 'draft'        AND NEW.status =  'finalized') OR
      (OLD.status = 'finalized'    AND NEW.status =  'archived')
    ) THEN
      RAISE EXCEPTION 'illegal status transition % -> %', OLD.status, NEW.status
        USING ERRCODE = 'check_violation';
    END IF;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_documents_v2_legal_transition ON documents;
CREATE TRIGGER trg_documents_v2_legal_transition
  BEFORE UPDATE ON documents
  FOR EACH ROW EXECUTE FUNCTION enforce_document_transition();

COMMIT;
