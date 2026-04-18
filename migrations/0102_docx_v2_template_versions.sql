-- 0102_docx_v2_template_versions.sql
BEGIN;

CREATE TABLE IF NOT EXISTS template_versions (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  template_id           UUID NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
  version_num           INT NOT NULL,
  status                TEXT NOT NULL CHECK (status IN ('draft','published','deprecated')),
  grammar_version       INT NOT NULL DEFAULT 1,
  docx_storage_key      TEXT NOT NULL,
  schema_storage_key    TEXT NOT NULL,
  docx_content_hash     TEXT NOT NULL,
  schema_content_hash   TEXT NOT NULL,
  published_at          TIMESTAMPTZ,
  published_by          UUID,
  deprecated_at         TIMESTAMPTZ,
  lock_version          INT NOT NULL DEFAULT 0,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by            UUID NOT NULL,
  CONSTRAINT template_versions_template_num_unique UNIQUE (template_id, version_num)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_one_draft_per_template
  ON template_versions (template_id) WHERE status = 'draft';

ALTER TABLE templates
  ADD CONSTRAINT fk_templates_current_published
    FOREIGN KEY (current_published_version_id)
    REFERENCES template_versions(id)
    DEFERRABLE INITIALLY IMMEDIATE;

COMMIT;
