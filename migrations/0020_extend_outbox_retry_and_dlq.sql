ALTER TABLE metaldocs.outbox_events
  ADD COLUMN IF NOT EXISTS attempt_count INT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS last_error TEXT,
  ADD COLUMN IF NOT EXISTS last_attempt_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS next_attempt_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS dead_lettered_at TIMESTAMPTZ;

DROP INDEX IF EXISTS metaldocs.idx_outbox_unpublished;

CREATE INDEX IF NOT EXISTS idx_outbox_claimable
ON metaldocs.outbox_events (occurred_at ASC)
WHERE published_at IS NULL
  AND dead_lettered_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_outbox_dead_lettered
ON metaldocs.outbox_events (dead_lettered_at DESC)
WHERE dead_lettered_at IS NOT NULL;
