-- 0068_add_theme_to_po_mddm_template.sql
-- Adds template theme to po-mddm-canvas definition_json for editor + DOCX alignment.

UPDATE metaldocs.document_template_versions
SET definition_json =
  definition_json ||
  '{"theme":{"accent":"#6b1f2a","accentLight":"#f9f3f3","accentDark":"#3e1018","accentBorder":"#dfc8c8"}}'::jsonb
WHERE template_key = 'po-mddm-canvas'
  AND version = 1;
