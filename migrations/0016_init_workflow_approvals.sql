CREATE TABLE IF NOT EXISTS metaldocs.workflow_approvals (
  id TEXT PRIMARY KEY,
  document_id TEXT NOT NULL REFERENCES metaldocs.documents(id) ON DELETE RESTRICT,
  requested_by TEXT NOT NULL,
  assigned_reviewer TEXT NOT NULL,
  decision_by TEXT,
  status TEXT NOT NULL CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED')),
  request_reason TEXT NOT NULL DEFAULT '',
  decision_reason TEXT,
  requested_at TIMESTAMPTZ NOT NULL,
  decided_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_workflow_approvals_document_id_requested_at
  ON metaldocs.workflow_approvals (document_id, requested_at DESC);
