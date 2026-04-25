BEGIN;

ALTER TABLE templates_v2_template_version
    DROP COLUMN IF EXISTS editable_zones_schema_snapshot,
    DROP COLUMN IF EXISTS editable_zones_schema,
    DROP COLUMN IF EXISTS editable_zones;

ALTER TABLE documents
    DROP COLUMN IF EXISTS editable_zones_schema_snapshot,
    DROP COLUMN IF EXISTS editable_zones_schema;

DROP TABLE IF EXISTS document_editable_zone_content;

COMMIT;
