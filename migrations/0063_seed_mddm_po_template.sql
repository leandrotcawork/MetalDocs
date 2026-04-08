-- 0063_seed_mddm_po_template.sql
-- Inserts the new MDDM-format PO template. Body comes from POTemplateMDDM() in Go,
-- but we seed via SQL for portability. Application-side seed code (if any) should
-- INSERT through the Go layer to keep parity with this migration.

INSERT INTO metaldocs.document_template_versions_mddm
  (template_id, version, mddm_version, content_blocks, content_hash, is_published)
SELECT
  '00000000-0000-0000-0000-0000000000a1'::uuid,
  1,
  1,
  '{"mddm_version":1,"template_ref":null,"blocks":[]}'::jsonb,
  'replaced-by-go-seed',
  false
WHERE NOT EXISTS (
  SELECT 1 FROM metaldocs.document_template_versions_mddm
  WHERE template_id = '00000000-0000-0000-0000-0000000000a1'::uuid AND version = 1
);
