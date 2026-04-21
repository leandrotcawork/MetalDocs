-- 0124_registry_controlled_documents.sql

CREATE TABLE IF NOT EXISTS controlled_documents (
  id                              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id                       UUID NOT NULL,
  profile_code                    TEXT NOT NULL,
  process_area_code               TEXT NOT NULL,
  department_code                 TEXT,
  code                            TEXT NOT NULL,
  sequence_num                    INT,
  title                           TEXT NOT NULL,
  owner_user_id                   TEXT NOT NULL,
  override_template_version_id    UUID REFERENCES templates_v2_template_version(id),
  status                          TEXT NOT NULL DEFAULT 'active'
                                    CHECK (status IN ('active','obsolete','superseded')),
  created_at                      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at                      TIMESTAMPTZ NOT NULL DEFAULT now(),

  FOREIGN KEY (tenant_id, profile_code)
    REFERENCES metaldocs.document_profiles (tenant_id, code),
  FOREIGN KEY (tenant_id, process_area_code)
    REFERENCES metaldocs.document_process_areas (tenant_id, code),

  UNIQUE (tenant_id, profile_code, code)
);

ALTER TABLE controlled_documents
  ADD CONSTRAINT IF NOT EXISTS controlled_document_code_format
    CHECK (length(code) >= 2 AND length(code) <= 100);

CREATE TABLE IF NOT EXISTS profile_sequence_counters (
  tenant_id     UUID NOT NULL,
  profile_code  TEXT NOT NULL,
  next_seq      INT NOT NULL DEFAULT 1,
  PRIMARY KEY (tenant_id, profile_code),
  FOREIGN KEY (tenant_id, profile_code)
    REFERENCES metaldocs.document_profiles (tenant_id, code)
);

CREATE INDEX IF NOT EXISTS ix_controlled_documents_tenant_area
  ON controlled_documents (tenant_id, process_area_code);

CREATE INDEX IF NOT EXISTS ix_controlled_documents_tenant_profile
  ON controlled_documents (tenant_id, profile_code);

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

DROP TRIGGER IF EXISTS trg_controlled_documents_code_immutable ON controlled_documents;
CREATE TRIGGER trg_controlled_documents_code_immutable
  BEFORE UPDATE ON controlled_documents
  FOR EACH ROW EXECUTE FUNCTION reject_code_update();
