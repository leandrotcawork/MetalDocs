-- 0126_documents_v2_bridge_columns.sql

-- Phase A (shadow): add as NULLABLE -- Phase C (enforce) adds NOT NULL after backfill
ALTER TABLE documents_v2
  ADD COLUMN IF NOT EXISTS controlled_document_id UUID
    REFERENCES controlled_documents(id),
  ADD COLUMN IF NOT EXISTS profile_code_snapshot TEXT,
  ADD COLUMN IF NOT EXISTS process_area_code_snapshot TEXT;

CREATE INDEX IF NOT EXISTS ix_documents_v2_controlled_doc
  ON documents_v2 (controlled_document_id)
  WHERE controlled_document_id IS NOT NULL;
