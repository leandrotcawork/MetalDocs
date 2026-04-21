-- 0129_documents_v2_bridge_not_null.sql
-- Phase C: enforce NOT NULL after 24h monitoring confirms 0 NULL rows.
-- Preflight guard: fail if any NULLs remain.

DO $$
DECLARE null_count INT;
BEGIN
  SELECT COUNT(*) INTO null_count FROM documents_v2 WHERE controlled_document_id IS NULL;
  IF null_count > 0 THEN
    RAISE EXCEPTION 'Phase C blocked: % documents_v2 rows still have NULL controlled_document_id',
      null_count;
  END IF;
END $$;

ALTER TABLE documents_v2
  ALTER COLUMN controlled_document_id SET NOT NULL,
  ALTER COLUMN profile_code_snapshot SET NOT NULL,
  ALTER COLUMN process_area_code_snapshot SET NOT NULL;
