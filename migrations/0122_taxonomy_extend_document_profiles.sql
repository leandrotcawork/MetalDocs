-- 0122_taxonomy_extend_document_profiles.sql

-- Step 1: add tenant_id with sentinel default (single-tenant MVP)
ALTER TABLE metaldocs.document_profiles
  ADD COLUMN IF NOT EXISTS tenant_id UUID NOT NULL DEFAULT 'ffffffff-ffff-ffff-ffff-ffffffffffff';

-- Step 2: new governance columns
ALTER TABLE metaldocs.document_profiles
  ADD COLUMN IF NOT EXISTS default_template_version_id UUID
    REFERENCES templates_v2_template_version(id),
  ADD COLUMN IF NOT EXISTS owner_user_id TEXT,
  ADD COLUMN IF NOT EXISTS editable_by_role TEXT NOT NULL DEFAULT 'admin',
  ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;

-- Step 3: code format constraint
ALTER TABLE metaldocs.document_profiles
  ADD CONSTRAINT IF NOT EXISTS profile_code_format
    CHECK (code ~ '^[a-z][a-z0-9_-]{1,63}$');

-- Step 4: tenant-scoped unique index (includes archived rows -- codes non-reusable)
CREATE UNIQUE INDEX IF NOT EXISTS ux_document_profiles_tenant_code
  ON metaldocs.document_profiles (tenant_id, code);

-- Step 5: immutability trigger
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

DROP TRIGGER IF EXISTS trg_document_profiles_code_immutable ON metaldocs.document_profiles;
CREATE TRIGGER trg_document_profiles_code_immutable
  BEFORE UPDATE ON metaldocs.document_profiles
  FOR EACH ROW EXECUTE FUNCTION reject_code_update();
