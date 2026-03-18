ALTER TABLE metaldocs.notifications
  DROP CONSTRAINT IF EXISTS notifications_status_check;

ALTER TABLE metaldocs.notifications
  ADD COLUMN IF NOT EXISTS read_at TIMESTAMPTZ;

ALTER TABLE metaldocs.notifications
  ADD CONSTRAINT notifications_status_check
  CHECK (status IN ('PENDING', 'SENT', 'READ'));
