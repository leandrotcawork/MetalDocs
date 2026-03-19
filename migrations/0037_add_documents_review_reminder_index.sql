CREATE INDEX IF NOT EXISTS idx_documents_review_reminder_window
ON metaldocs.documents (status, expiry_at)
WHERE expiry_at IS NOT NULL
  AND status IN ('APPROVED', 'PUBLISHED');
