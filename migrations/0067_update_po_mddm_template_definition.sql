-- 0067_update_po_mddm_template_definition.sql
-- Updates the po-mddm-canvas template definition_json to include the full MDDM
-- block structure from the canonical MDDM template table. Migration 0065 seeded
-- an empty placeholder definition; this fills it with the canonical block tree
-- so the browser editor can instantiate new documents with the full template.

UPDATE metaldocs.document_template_versions
SET definition_json = jsonb_build_object(
  'type', 'page',
  'id', 'po-mddm-root',
  'children', (
    SELECT content_blocks->'blocks'
    FROM metaldocs.document_template_versions_mddm
    WHERE template_id = '00000000-0000-0000-0000-0000000000a1'
      AND version = 1
    LIMIT 1
  )
)
WHERE template_key = 'po-mddm-canvas'
  AND version = 1;
