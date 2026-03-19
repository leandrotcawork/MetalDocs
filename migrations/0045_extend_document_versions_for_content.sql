ALTER TABLE metaldocs.document_versions
  ADD COLUMN IF NOT EXISTS content_source TEXT NOT NULL DEFAULT 'native';

ALTER TABLE metaldocs.document_versions
  ADD COLUMN IF NOT EXISTS native_content JSONB;

ALTER TABLE metaldocs.document_versions
  ADD COLUMN IF NOT EXISTS docx_storage_key TEXT;

ALTER TABLE metaldocs.document_versions
  ADD COLUMN IF NOT EXISTS pdf_storage_key TEXT;

ALTER TABLE metaldocs.document_versions
  ADD COLUMN IF NOT EXISTS text_content TEXT;

ALTER TABLE metaldocs.document_versions
  ADD COLUMN IF NOT EXISTS file_size_bytes BIGINT;

ALTER TABLE metaldocs.document_versions
  ADD COLUMN IF NOT EXISTS original_filename TEXT;

ALTER TABLE metaldocs.document_versions
  ADD COLUMN IF NOT EXISTS page_count INTEGER;

UPDATE metaldocs.document_versions
SET text_content = content
WHERE (text_content IS NULL OR text_content = '')
  AND content IS NOT NULL;

UPDATE metaldocs.document_versions
SET content_source = 'native'
WHERE content_source IS NULL OR content_source = '';

ALTER TABLE metaldocs.document_versions
  ADD COLUMN IF NOT EXISTS search_vector TSVECTOR
    GENERATED ALWAYS AS (
      to_tsvector('portuguese', coalesce(text_content, ''))
    ) STORED;

CREATE INDEX IF NOT EXISTS idx_doc_versions_search
  ON metaldocs.document_versions USING GIN (search_vector);
