-- 0074_lock_seed_etapa_item.sql
-- Sets locked=true on the seed repeatableItem "Etapa 1" inside Section 5
-- (Detalhamento das Etapas) so it cannot be deleted by users.
-- User-added etapas are NOT affected — they default to locked=false.
--
-- Path in definition_json:
--   children[4].children[0].children[0].props

UPDATE metaldocs.document_template_versions
SET definition_json = jsonb_set(
  definition_json,
  '{children,4,children,0,children,0,props,locked}',
  'true'::jsonb
)
WHERE template_key = 'po-mddm-canvas'
  AND version = 1;
