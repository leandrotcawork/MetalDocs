-- 0123_taxonomy_extend_process_areas.sql

ALTER TABLE metaldocs.document_process_areas
  ADD COLUMN IF NOT EXISTS tenant_id UUID NOT NULL DEFAULT 'ffffffff-ffff-ffff-ffff-ffffffffffff',
  ADD COLUMN IF NOT EXISTS parent_code TEXT,
  ADD COLUMN IF NOT EXISTS owner_user_id TEXT,
  ADD COLUMN IF NOT EXISTS default_approver_role TEXT,
  ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;

ALTER TABLE metaldocs.document_process_areas
  ADD CONSTRAINT IF NOT EXISTS fk_area_parent_tenant
    FOREIGN KEY (tenant_id, parent_code)
    REFERENCES metaldocs.document_process_areas (tenant_id, code);

ALTER TABLE metaldocs.document_process_areas
  ADD CONSTRAINT IF NOT EXISTS area_code_format
    CHECK (code ~ '^[a-z][a-z0-9_-]{1,63}$');

CREATE UNIQUE INDEX IF NOT EXISTS ux_process_areas_tenant_code
  ON metaldocs.document_process_areas (tenant_id, code);

-- idempotent: safe to re-create, identical to definition in 0122
CREATE OR REPLACE FUNCTION reject_code_update() RETURNS trigger AS $$
BEGIN
  IF NEW.code IS DISTINCT FROM OLD.code THEN
    RAISE EXCEPTION 'code column is immutable (table=%, old=%, new=%)',
      TG_TABLE_NAME, OLD.code, NEW.code
      USING ERRCODE = 'check_violation';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_process_areas_code_immutable ON metaldocs.document_process_areas;
CREATE TRIGGER trg_process_areas_code_immutable
  BEFORE UPDATE ON metaldocs.document_process_areas
  FOR EACH ROW EXECUTE FUNCTION reject_code_update();
