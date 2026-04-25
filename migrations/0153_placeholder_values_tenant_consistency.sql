-- migrations/0153_placeholder_values_tenant_consistency.sql
BEGIN;

CREATE OR REPLACE FUNCTION enforce_placeholder_value_tenant_consistent() RETURNS trigger AS $$
DECLARE doc_tenant UUID;
BEGIN
    SELECT tenant_id INTO doc_tenant FROM documents WHERE id = NEW.revision_id;
    IF doc_tenant IS NULL THEN
        RAISE EXCEPTION 'document % not found', NEW.revision_id USING ERRCODE = 'foreign_key_violation';
    END IF;
    IF doc_tenant <> NEW.tenant_id THEN
        RAISE EXCEPTION 'tenant mismatch: document=% value=%', doc_tenant, NEW.tenant_id
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS enforce_placeholder_value_tenant_trg ON document_placeholder_values;
CREATE TRIGGER enforce_placeholder_value_tenant_trg
    BEFORE INSERT OR UPDATE ON document_placeholder_values
    FOR EACH ROW EXECUTE FUNCTION enforce_placeholder_value_tenant_consistent();

DROP TRIGGER IF EXISTS enforce_zone_content_tenant_trg ON document_editable_zone_content;
CREATE TRIGGER enforce_zone_content_tenant_trg
    BEFORE INSERT OR UPDATE ON document_editable_zone_content
    FOR EACH ROW EXECUTE FUNCTION enforce_placeholder_value_tenant_consistent();

COMMIT;
