-- 0077_add_template_publish_state.sql
-- Adds CK5 template publish state machine (draft → pending_review → published).
-- The published_html column stores the frozen snapshot used by fill-mode.
-- The draft_status column tracks the CK5 review workflow (distinct from
-- the template_versions.status column which tracks the lifecycle of
-- published versions: draft/published/deprecated).

ALTER TABLE metaldocs.template_drafts
  ADD COLUMN IF NOT EXISTS published_html TEXT,
  ADD COLUMN IF NOT EXISTS draft_status TEXT NOT NULL DEFAULT 'draft'
    CHECK (draft_status IN ('draft', 'pending_review', 'published'));

CREATE INDEX IF NOT EXISTS idx_template_drafts_draft_status
  ON metaldocs.template_drafts(draft_status);
