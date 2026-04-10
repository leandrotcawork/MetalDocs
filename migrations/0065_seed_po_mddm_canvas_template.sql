-- 0065_seed_po_mddm_canvas_template.sql
-- Seeds the po-mddm-canvas template into document_template_versions.
-- This is the template used by the MDDM BlockNote browser editor for PO documents.
-- The Go canonical definition lives in domain.DefaultDocumentTemplateVersions().

INSERT INTO metaldocs.document_template_versions (
  template_key, version, profile_code, schema_version, name, definition_json, editor, content_format, body_html
)
VALUES (
  'po-mddm-canvas',
  1,
  'po',
  3,
  'PO MDDM Canvas v1',
  '{"type": "page", "id": "po-mddm-root", "children": []}'::jsonb,
  'mddm-blocknote',
  'mddm',
  ''
)
ON CONFLICT (template_key, version) DO UPDATE
SET
  editor         = EXCLUDED.editor,
  content_format = EXCLUDED.content_format,
  name           = EXCLUDED.name,
  definition_json = EXCLUDED.definition_json;
