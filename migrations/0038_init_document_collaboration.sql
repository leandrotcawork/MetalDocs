CREATE TABLE IF NOT EXISTS metaldocs.document_collaboration_presence (
  document_id TEXT NOT NULL REFERENCES metaldocs.documents(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL,
  display_name TEXT NOT NULL,
  last_seen_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (document_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_document_collaboration_presence_last_seen
  ON metaldocs.document_collaboration_presence (document_id, last_seen_at DESC);

CREATE TABLE IF NOT EXISTS metaldocs.document_edit_locks (
  document_id TEXT PRIMARY KEY REFERENCES metaldocs.documents(id) ON DELETE CASCADE,
  locked_by TEXT NOT NULL,
  display_name TEXT NOT NULL,
  lock_reason TEXT NOT NULL DEFAULT '',
  acquired_at TIMESTAMPTZ NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_document_edit_locks_expiry CHECK (expires_at > acquired_at)
);

CREATE INDEX IF NOT EXISTS idx_document_edit_locks_expires_at
  ON metaldocs.document_edit_locks (expires_at);
