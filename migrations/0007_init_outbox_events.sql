CREATE TABLE IF NOT EXISTS metaldocs.outbox_events (
  event_id TEXT PRIMARY KEY,
  event_type TEXT NOT NULL,
  aggregate_type TEXT NOT NULL,
  aggregate_id TEXT NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL,
  version INT NOT NULL,
  idempotency_key TEXT NOT NULL UNIQUE,
  producer TEXT NOT NULL,
  trace_id TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_outbox_unpublished ON metaldocs.outbox_events (occurred_at ASC)
WHERE published_at IS NULL;

