-- 0071_unlock_editable_blocks_in_po_template.sql
-- Sets locked: false on user-editable repeatable/dataTable blocks in the
-- po-mddm-canvas template definition_json.
--
-- The Etapas repeatable (children[3].children[0]) and KPIs dataTable
-- (children[4].children[0]) were seeded with locked: true, which prevented
-- the UI "add item / add row" buttons from appearing or being active.
-- Users must be able to add Etapas and KPI rows — these blocks are
-- structurally fixed (label, columns) but their children are user-owned.

UPDATE metaldocs.document_template_versions
SET definition_json = jsonb_set(
  jsonb_set(
    definition_json,
    '{children,3,children,0,props,locked}',
    'false'
  ),
  '{children,4,children,0,props,locked}',
  'false'
)
WHERE template_key = 'po-mddm-canvas'
  AND version = 1;
