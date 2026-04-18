-- 0108_docx_v2_template_audit_log.sql
BEGIN;

CREATE TABLE IF NOT EXISTS template_audit_log (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id            UUID NOT NULL,
  template_id          UUID,
  template_version_id  UUID,
  document_id          UUID,
  action               TEXT NOT NULL,
  actor_user_id        UUID NOT NULL,
  metadata_json        JSONB,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_tenant_created
  ON template_audit_log (tenant_id, created_at DESC);

-- Append-only enforcement at DB layer.
-- Runtime role name is configured via METALDOCS_DB_APP_ROLE env; default 'metaldocs_app'.
DO $$
DECLARE role_name TEXT := current_setting('metaldocs.app_role', true);
BEGIN
  IF role_name IS NULL OR role_name = '' THEN
    role_name := 'metaldocs_app';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = role_name) THEN
    EXECUTE format('REVOKE UPDATE, DELETE ON template_audit_log FROM %I', role_name);
    EXECUTE format('GRANT  INSERT, SELECT ON template_audit_log TO %I', role_name);
  END IF;
END$$;

COMMIT;
