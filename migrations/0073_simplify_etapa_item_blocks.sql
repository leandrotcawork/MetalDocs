-- 0073_simplify_etapa_item_blocks.sql
-- Removes "Conteúdo da etapa" (richBlock) and "Checklist da etapa" (dataTable)
-- from the seed repeatableItem inside Section 5 (Detalhamento das Etapas).
--
-- Structure path in definition_json:
--   children[4]              → section "Detalhamento das Etapas"
--   children[4].children[0] → repeatable "Etapas"
--   children[4].children[0].children[0] → repeatableItem "Etapa 1"
--   children[4].children[0].children[0].children → [] (was richBlock + dataTable)

UPDATE metaldocs.document_template_versions
SET definition_json = jsonb_set(
  definition_json,
  '{children,4,children,0,children,0,children}',
  '[]'::jsonb
)
WHERE template_key = 'po-mddm-canvas'
  AND version = 1;
