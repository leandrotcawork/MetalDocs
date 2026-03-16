CREATE TABLE IF NOT EXISTS metaldocs.audit_events (
  id TEXT PRIMARY KEY,
  occurred_at TIMESTAMPTZ NOT NULL,
  actor_id TEXT NOT NULL,
  action TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  trace_id TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_events_occurred_at ON metaldocs.audit_events (occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_actor_time ON metaldocs.audit_events (actor_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_resource_time ON metaldocs.audit_events (resource_type, resource_id, occurred_at DESC);

