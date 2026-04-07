-- 0062_create_mddm_triggers.sql
-- Enforces template immutability via DB trigger.

CREATE OR REPLACE FUNCTION metaldocs.prevent_published_template_mutation()
RETURNS TRIGGER AS $$
BEGIN
  IF OLD.is_published = true AND NEW.content_blocks IS DISTINCT FROM OLD.content_blocks THEN
    RAISE EXCEPTION 'Cannot modify content_blocks of a published template version (id=%)', OLD.id;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_template_immutable ON metaldocs.document_template_versions_mddm;
CREATE TRIGGER trg_template_immutable
  BEFORE UPDATE ON metaldocs.document_template_versions_mddm
  FOR EACH ROW
  EXECUTE FUNCTION metaldocs.prevent_published_template_mutation();
