-- 0123_taxonomy_extend_process_areas.sql

ALTER TABLE metaldocs.document_process_areas
  ADD COLUMN IF NOT EXISTS tenant_id UUID NOT NULL DEFAULT 'ffffffff-ffff-ffff-ffff-ffffffffffff',
  ADD COLUMN IF NOT EXISTS parent_code TEXT
    REFERENCES metaldocs.document_process_areas(code),
  ADD COLUMN IF NOT EXISTS owner_user_id UUID,
  ADD COLUMN IF NOT EXISTS default_approver_role TEXT,
  ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;

ALTER TABLE metaldocs.document_process_areas
  ADD CONSTRAINT IF NOT EXISTS area_code_format
    CHECK (code ~ '^[a-z][a-z0-9_-]{1,63}$');

CREATE UNIQUE INDEX IF NOT EXISTS ux_process_areas_tenant_code
  ON metaldocs.document_process_areas (tenant_id, code);

-- reject_code_update() already created in 0122
DROP TRIGGER IF EXISTS trg_process_areas_code_immutable ON metaldocs.document_process_areas;
CREATE TRIGGER trg_process_areas_code_immutable
  BEFORE UPDATE ON metaldocs.document_process_areas
  FOR EACH ROW EXECUTE FUNCTION reject_code_update();
