-- 0076_add_status_to_template_versions.sql
-- Adds a lifecycle status column to document_template_versions so that
-- template versions can be marked as published or deprecated.
-- Existing rows are back-filled to 'published' since all historical versions
-- were published at insert time.

ALTER TABLE metaldocs.document_template_versions
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'published';

UPDATE metaldocs.document_template_versions
SET status = 'published'
WHERE status = 'published'; -- no-op update, ensures constraint awareness
