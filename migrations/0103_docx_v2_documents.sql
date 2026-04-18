-- 0103_docx_v2_documents.sql
BEGIN;

CREATE TABLE IF NOT EXISTS documents_v2 (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id             UUID NOT NULL,
  template_version_id   UUID NOT NULL REFERENCES template_versions(id),
  name                  TEXT NOT NULL,
  status                TEXT NOT NULL CHECK (status IN ('draft','finalized','archived')),
  form_data_json        JSONB NOT NULL,
  current_revision_id   UUID,
  active_session_id     UUID,
  finalized_at          TIMESTAMPTZ,
  archived_at           TIMESTAMPTZ,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by            UUID NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_documents_v2_tenant_status
  ON documents_v2 (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_documents_v2_template_version
  ON documents_v2 (template_version_id);
CREATE INDEX IF NOT EXISTS idx_documents_v2_form_data_gin
  ON documents_v2 USING GIN (form_data_json jsonb_path_ops);

COMMIT;
