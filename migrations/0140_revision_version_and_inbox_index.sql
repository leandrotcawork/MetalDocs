-- migrations/0140_revision_version_and_inbox_index.sql

BEGIN;

CREATE OR REPLACE FUNCTION enforce_revision_version_monotonic()
  RETURNS trigger AS $$
BEGIN
  IF NEW.revision_version < OLD.revision_version THEN
    RAISE EXCEPTION 'revision_version cannot decrease (OLD=%, NEW=%)',
                    OLD.revision_version, NEW.revision_version
      USING ERRCODE = 'check_violation';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql
   SET search_path = pg_catalog, pg_temp;

DROP TRIGGER IF EXISTS trg_documents_v2_revision_version_monotonic ON documents;
CREATE TRIGGER trg_documents_v2_revision_version_monotonic
  BEFORE UPDATE ON documents
  FOR EACH ROW EXECUTE FUNCTION enforce_revision_version_monotonic();

CREATE INDEX IF NOT EXISTS ix_approval_instances_inbox
  ON approval_instances (tenant_id, submitted_by, status, submitted_at DESC);

COMMIT;
