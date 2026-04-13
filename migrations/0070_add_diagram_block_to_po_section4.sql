-- 0070_add_diagram_block_to_po_section4.sql
-- Adds a "Diagrama" richBlock (with an empty image placeholder) to Section 4
-- (Visão Geral do Processo) of the po-mddm-canvas template.
-- Section 4 is at index 3 in definition_json.children.

UPDATE metaldocs.document_template_versions
SET definition_json = jsonb_set(
  definition_json,
  '{children,3,children}',
  (definition_json #> '{children,3,children}') || jsonb_build_array(
    jsonb_build_object(
      'id',       'a0000034-0000-0000-0000-000000000034',
      'type',     'richBlock',
      'props',    jsonb_build_object('label', 'Diagrama', 'locked', true),
      'children', jsonb_build_array(
        jsonb_build_object(
          'id',       'a0000035-0000-0000-0000-000000000035',
          'type',     'image',
          'props',    jsonb_build_object('src', '', 'alt', 'Diagrama do processo', 'caption', ''),
          'children', '[]'::jsonb
        )
      )
    )
  )
)
WHERE template_key = 'po-mddm-canvas'
  AND version = 1;
