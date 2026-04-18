-- 0101_docx_v2_templates.sql
-- Logical template (e.g. "Purchase Order"). Owned by a tenant.
-- Part of docx-editor platform (W1 scaffold). Tables in this block
-- are prefixed docx_v2_ in the migration filename but take their
-- spec names as the table identifier because they supersede CK5.

BEGIN;

CREATE TABLE IF NOT EXISTS templates (
  id                            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id                     UUID NOT NULL,
  key                           TEXT NOT NULL,
  name                          TEXT NOT NULL,
  description                   TEXT,
  current_published_version_id  UUID,
  created_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by                    UUID NOT NULL,
  CONSTRAINT templates_tenant_key_unique UNIQUE (tenant_id, key)
);

CREATE INDEX IF NOT EXISTS idx_templates_tenant
  ON templates (tenant_id);

COMMIT;
