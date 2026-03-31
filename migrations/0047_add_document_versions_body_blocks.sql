ALTER TABLE metaldocs.document_versions
  ADD COLUMN IF NOT EXISTS body_blocks JSONB DEFAULT '[]';
