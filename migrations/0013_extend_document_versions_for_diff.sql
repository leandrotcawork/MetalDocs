ALTER TABLE metaldocs.document_versions
  ADD COLUMN IF NOT EXISTS content_hash TEXT;

ALTER TABLE metaldocs.document_versions
  ADD COLUMN IF NOT EXISTS change_summary TEXT NOT NULL DEFAULT '';

UPDATE metaldocs.document_versions
SET content_hash = md5(content)
WHERE content_hash IS NULL OR TRIM(content_hash) = '';

ALTER TABLE metaldocs.document_versions
  ALTER COLUMN content_hash SET NOT NULL;
