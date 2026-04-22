-- 0130_documents_drop_old_template_version_fk.sql
-- The documents table was created (0110) before templates_v2 existed.
-- Its template_version_id FK points to the legacy template_versions table.
-- documents_v2 module stores templates_v2_template_version IDs in this column,
-- which violates the FK. Drop the constraint; the column is kept as informational.

ALTER TABLE documents
  DROP CONSTRAINT IF EXISTS documents_template_version_id_fkey;
