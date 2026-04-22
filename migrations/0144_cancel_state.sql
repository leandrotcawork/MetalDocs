-- migrations/0144_cancel_state.sql
-- Spec 2 Phase 6 (A5). Extends status CHECK constraints to allow 'cancelled'
-- on approval_instances and approval_stage_instances.
-- Updates enforce_document_transition() to permit under_review→draft when
-- GUC metaldocs.cancel_in_progress is set (cancel workflow path).
-- Idempotent.

BEGIN;

-- -------------------------------------------------------------------------
-- 1. approval_instances: extend status CHECK to include 'cancelled'
-- -------------------------------------------------------------------------
DO $$ BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conrelid = 'public.approval_instances'::regclass
      AND conname  = 'approval_instances_status_check'
  ) THEN
    ALTER TABLE public.approval_instances
      DROP CONSTRAINT approval_instances_status_check;
  END IF;
END $$;

ALTER TABLE public.approval_instances
  ADD CONSTRAINT approval_instances_status_check
    CHECK (status IN ('in_progress','approved','rejected','cancelled'));

-- -------------------------------------------------------------------------
-- 2. approval_stage_instances: extend status CHECK to include 'cancelled'
-- -------------------------------------------------------------------------
DO $$ BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conrelid = 'public.approval_stage_instances'::regclass
      AND conname  = 'approval_stage_instances_status_check'
  ) THEN
    ALTER TABLE public.approval_stage_instances
      DROP CONSTRAINT approval_stage_instances_status_check;
  END IF;
END $$;

ALTER TABLE public.approval_stage_instances
  ADD CONSTRAINT approval_stage_instances_status_check
    CHECK (status IN ('pending','active','completed','skipped','rejected_here','cancelled'));

-- -------------------------------------------------------------------------
-- 3. Update enforce_document_transition() to allow under_review→draft
--    when metaldocs.cancel_in_progress GUC is set (cancel path only).
-- -------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION enforce_document_transition()
  RETURNS trigger
  LANGUAGE plpgsql
  SET search_path = pg_catalog, pg_temp
AS $$
DECLARE
  cancel_instance_id TEXT;
BEGIN
  IF OLD.status IS DISTINCT FROM NEW.status THEN
    -- Check cancel path: under_review→draft allowed if cancel GUC is set.
    IF OLD.status = 'under_review' AND NEW.status = 'draft' THEN
      cancel_instance_id := current_setting('metaldocs.cancel_in_progress', true);
      IF cancel_instance_id IS NULL OR cancel_instance_id = '' THEN
        RAISE EXCEPTION 'illegal status transition % -> % (set metaldocs.cancel_in_progress GUC to authorise cancel rollback)',
          OLD.status, NEW.status
          USING ERRCODE = 'check_violation';
      END IF;
      RETURN NEW;
    END IF;

    IF NOT (
      -- Spec 2 graph
      (OLD.status = 'draft'        AND NEW.status =  'under_review') OR
      (OLD.status = 'under_review' AND NEW.status IN ('approved','rejected')) OR
      (OLD.status = 'rejected'     AND NEW.status =  'draft') OR
      (OLD.status = 'approved'     AND NEW.status IN ('published','scheduled','draft')) OR
      (OLD.status = 'scheduled'    AND NEW.status IN ('published','draft')) OR
      (OLD.status = 'published'    AND NEW.status IN ('superseded','obsolete')) OR
      (OLD.status = 'superseded'   AND NEW.status =  'obsolete')
      -- Note: compat window (draft→finalized, finalized→archived) removed by 0142_disable_legacy_compat.sql
    ) THEN
      RAISE EXCEPTION 'illegal status transition % -> %', OLD.status, NEW.status
        USING ERRCODE = 'check_violation';
    END IF;
  END IF;
  RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_documents_v2_legal_transition ON documents;
CREATE TRIGGER trg_documents_v2_legal_transition
  BEFORE UPDATE ON documents
  FOR EACH ROW EXECUTE FUNCTION enforce_document_transition();

COMMIT;
