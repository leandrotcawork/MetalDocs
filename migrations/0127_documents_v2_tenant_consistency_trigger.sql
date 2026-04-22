-- 0127_documents_v2_tenant_consistency_trigger.sql

CREATE OR REPLACE FUNCTION check_document_tenant_consistency() RETURNS trigger AS $$
DECLARE
  cd_tenant UUID;
BEGIN
  IF TG_OP = 'UPDATE' AND OLD.controlled_document_id IS NOT DISTINCT FROM NEW.controlled_document_id THEN
    RETURN NEW;
  END IF;

  IF NEW.controlled_document_id IS NOT NULL THEN
    SELECT tenant_id INTO cd_tenant
      FROM controlled_documents WHERE id = NEW.controlled_document_id;
    IF NOT FOUND THEN
      RAISE EXCEPTION 'controlled_document_id % does not exist in controlled_documents',
        NEW.controlled_document_id;
    END IF;
    IF cd_tenant IS DISTINCT FROM NEW.tenant_id THEN
      RAISE EXCEPTION 'tenant mismatch between document (%) and controlled_document (%)',
        NEW.tenant_id, cd_tenant;
    END IF;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_documents_v2_tenant_consistency ON documents_v2;
CREATE TRIGGER trg_documents_v2_tenant_consistency
  BEFORE INSERT OR UPDATE ON documents_v2
  FOR EACH ROW EXECUTE FUNCTION check_document_tenant_consistency();
