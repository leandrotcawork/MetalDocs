-- 0119_documents_v2_link_template_version.sql

ALTER TABLE documents_v2_documents
  ADD COLUMN templates_v2_template_version_id uuid NULL
  REFERENCES templates_v2_template_version(id);

CREATE INDEX idx_documents_v2_template_version
  ON documents_v2_documents (templates_v2_template_version_id)
  WHERE templates_v2_template_version_id IS NOT NULL;
