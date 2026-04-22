-- migrations/0136_user_process_areas_hardening.sql

BEGIN;

ALTER TABLE user_process_areas
  ADD COLUMN IF NOT EXISTS revoked_by TEXT;

-- Codex Round 2 fix #4: pre-existing revoked rows may lack revoked_by.
-- Backfill FIRST (system:legacy sentinel), then enforce CHECK.
UPDATE user_process_areas
   SET revoked_by = 'system:legacy'
 WHERE effective_to IS NOT NULL
   AND revoked_by   IS NULL;

ALTER TABLE user_process_areas
  DROP CONSTRAINT IF EXISTS revoked_by_required_when_revoked,
  ADD  CONSTRAINT revoked_by_required_when_revoked
    CHECK ((effective_to IS NULL AND revoked_by IS NULL)
        OR (effective_to IS NOT NULL AND revoked_by IS NOT NULL))
    NOT VALID;

ALTER TABLE user_process_areas
  VALIDATE CONSTRAINT revoked_by_required_when_revoked;

ALTER TABLE user_process_areas
  DROP CONSTRAINT IF EXISTS effective_interval_valid,
  ADD  CONSTRAINT effective_interval_valid
    CHECK (effective_to IS NULL OR effective_to > effective_from);

CREATE UNIQUE INDEX IF NOT EXISTS ux_user_process_areas_single_active
  ON user_process_areas (tenant_id, user_id, area_code, role)
  WHERE effective_to IS NULL;

-- NOT VALID FKs first. VALIDATE after explicit verification below.
ALTER TABLE user_process_areas
  DROP CONSTRAINT IF EXISTS user_process_areas_granted_by_same_tenant,
  DROP CONSTRAINT IF EXISTS user_process_areas_revoked_by_same_tenant,
  ADD  CONSTRAINT user_process_areas_granted_by_same_tenant
    FOREIGN KEY (tenant_id, granted_by)
      REFERENCES metaldocs.iam_users (tenant_id, user_id)
    NOT VALID,
  ADD  CONSTRAINT user_process_areas_revoked_by_same_tenant
    FOREIGN KEY (tenant_id, revoked_by)
      REFERENCES metaldocs.iam_users (tenant_id, user_id)
    NOT VALID;

-- Explicit backfill verification. Raises if any row would fail FK -- migration aborts.
DO $$
DECLARE
  missing_granted INT;
  missing_revoked INT;
BEGIN
  SELECT COUNT(*) INTO missing_granted
    FROM user_process_areas upa
    LEFT JOIN metaldocs.iam_users u
      ON u.tenant_id = upa.tenant_id AND u.user_id = upa.granted_by
   WHERE upa.granted_by IS NOT NULL AND u.user_id IS NULL;

  SELECT COUNT(*) INTO missing_revoked
    FROM user_process_areas upa
    LEFT JOIN metaldocs.iam_users u
      ON u.tenant_id = upa.tenant_id AND u.user_id = upa.revoked_by
   WHERE upa.revoked_by IS NOT NULL AND u.user_id IS NULL;

  IF missing_granted > 0 OR missing_revoked > 0 THEN
    RAISE EXCEPTION
      'FK backfill verification failed: % granted_by, % revoked_by rows lack matching iam_users. Remediate before VALIDATE.',
      missing_granted, missing_revoked;
  END IF;
END $$;

ALTER TABLE user_process_areas
  VALIDATE CONSTRAINT user_process_areas_granted_by_same_tenant,
  VALIDATE CONSTRAINT user_process_areas_revoked_by_same_tenant;

-- No-DELETE trigger.
CREATE OR REPLACE FUNCTION reject_user_process_areas_delete() RETURNS trigger AS $$
BEGIN
  RAISE EXCEPTION 'user_process_areas rows cannot be deleted (revoke via UPDATE effective_to)'
    USING ERRCODE = 'check_violation';
END;
$$ LANGUAGE plpgsql
   SET search_path = pg_catalog, pg_temp;

DROP TRIGGER IF EXISTS trg_user_process_areas_no_delete ON user_process_areas;
CREATE TRIGGER trg_user_process_areas_no_delete
  BEFORE DELETE ON user_process_areas
  FOR EACH ROW EXECUTE FUNCTION reject_user_process_areas_delete();

-- Identity-immutable + no un-revoke trigger.
CREATE OR REPLACE FUNCTION enforce_user_process_areas_update_contract() RETURNS trigger AS $$
BEGIN
  IF NEW.tenant_id      IS DISTINCT FROM OLD.tenant_id      OR
     NEW.user_id        IS DISTINCT FROM OLD.user_id        OR
     NEW.area_code      IS DISTINCT FROM OLD.area_code      OR
     NEW.role           IS DISTINCT FROM OLD.role           OR
     NEW.effective_from IS DISTINCT FROM OLD.effective_from OR
     NEW.granted_by     IS DISTINCT FROM OLD.granted_by     THEN
    RAISE EXCEPTION 'identity columns are immutable on user_process_areas'
      USING ERRCODE = 'check_violation';
  END IF;
  IF OLD.effective_to IS NOT NULL AND NEW.effective_to IS NULL THEN
    RAISE EXCEPTION 'cannot un-revoke membership (re-grant creates new row)'
      USING ERRCODE = 'check_violation';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql
   SET search_path = pg_catalog, pg_temp;

DROP TRIGGER IF EXISTS trg_user_process_areas_update_contract ON user_process_areas;
CREATE TRIGGER trg_user_process_areas_update_contract
  BEFORE UPDATE ON user_process_areas
  FOR EACH ROW EXECUTE FUNCTION enforce_user_process_areas_update_contract();

COMMIT;
