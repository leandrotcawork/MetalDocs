-- migrations/0142b_down.sql
-- Rollback for 0142b_role_capabilities_v2_enforce.sql.
--
-- Reverses in order:
--   1. Disable tripwire triggers on approval_instances + approval_signoffs
--   2. Drop the tripwire function
--   3. Drop CHECK constraints ck_cap_not_legacy and ck_cap_format
--   4. Re-insert legacy capability rows that were stripped by the enforce step
--
-- Note: rolling back here does NOT re-enable old binary compatibility on its own.
-- 0142a must also be rolled back (separately) if the additive caps are unwanted.

BEGIN;

-- ---------------------------------------------------------------------------
-- 1. Disable tripwire triggers.
-- ---------------------------------------------------------------------------
DROP TRIGGER IF EXISTS trg_require_cap_asserted_instances ON public.approval_instances;
DROP TRIGGER IF EXISTS trg_require_cap_asserted_signoffs  ON public.approval_signoffs;

-- ---------------------------------------------------------------------------
-- 2. Drop the tripwire function.
-- ---------------------------------------------------------------------------
DROP FUNCTION IF EXISTS public.enforce_capability_asserted();

-- ---------------------------------------------------------------------------
-- 3. Drop CHECK constraints added by the enforce step.
-- ---------------------------------------------------------------------------
ALTER TABLE metaldocs.role_capabilities
  DROP CONSTRAINT IF EXISTS ck_cap_not_legacy;

ALTER TABLE metaldocs.role_capabilities
  DROP CONSTRAINT IF EXISTS ck_cap_format;

-- ---------------------------------------------------------------------------
-- 4. Re-insert legacy capability rows.
--    ON CONFLICT DO NOTHING — safe if they were never fully deleted.
-- ---------------------------------------------------------------------------
INSERT INTO metaldocs.role_capabilities (role, capability, description) VALUES
  ('author',     'document.finalize', 'Legacy: finalize document'),
  ('area_admin', 'document.archive',  'Legacy: archive document')
ON CONFLICT (role, capability) DO NOTHING;

COMMIT;
