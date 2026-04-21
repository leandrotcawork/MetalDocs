-- 0125_registry_iam_user_process_areas_governance_events.sql

CREATE TABLE IF NOT EXISTS user_process_areas (
  user_id         TEXT NOT NULL,
  tenant_id       UUID NOT NULL,
  area_code       TEXT NOT NULL,
  role            TEXT NOT NULL CHECK (role IN ('viewer','editor','reviewer','approver')),
  effective_from  TIMESTAMPTZ NOT NULL,
  effective_to    TIMESTAMPTZ,
  granted_by      TEXT,
  PRIMARY KEY (user_id, area_code, effective_from),
  FOREIGN KEY (tenant_id, area_code)
    REFERENCES metaldocs.document_process_areas (tenant_id, code)
);

CREATE INDEX IF NOT EXISTS ix_user_process_areas_active
  ON user_process_areas (user_id, area_code)
  WHERE effective_to IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_user_process_areas_one_active
  ON user_process_areas (user_id, tenant_id, area_code)
  WHERE effective_to IS NULL;

CREATE TABLE IF NOT EXISTS governance_events (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id      UUID NOT NULL,
  event_type     TEXT NOT NULL,
  actor_user_id  TEXT NOT NULL,
  resource_type  TEXT NOT NULL,
  resource_id    TEXT NOT NULL,
  reason         TEXT,
  payload_json   JSONB NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ix_governance_events_tenant_type
  ON governance_events (tenant_id, event_type, created_at DESC);

CREATE INDEX IF NOT EXISTS ix_governance_events_resource
  ON governance_events (resource_type, resource_id);
