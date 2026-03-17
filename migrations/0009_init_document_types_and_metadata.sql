CREATE TABLE IF NOT EXISTS metaldocs.document_types (
  code TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  review_interval_days INT NOT NULL CHECK (review_interval_days > 0),
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO metaldocs.document_types (code, name, description, review_interval_days)
VALUES
  ('policy', 'Policy', 'High-level governance and policy document', 365),
  ('procedure', 'Procedure', 'Operational procedure with controlled steps', 365),
  ('work_instruction', 'Work Instruction', 'Detailed execution instruction', 180),
  ('contract', 'Contract', 'Commercial or legal agreement', 365),
  ('supplier_document', 'Supplier Document', 'Document received from supplier', 180),
  ('technical_drawing', 'Technical Drawing', 'Engineering drawing or technical artifact', 180),
  ('certificate', 'Certificate', 'Certificate with issuer and validity context', 365),
  ('report', 'Report', 'Periodic or ad-hoc report', 365),
  ('form', 'Form', 'Structured business form', 180),
  ('manual', 'Manual', 'Reference or guidance manual', 365)
ON CONFLICT (code) DO NOTHING;

ALTER TABLE metaldocs.documents
  ADD COLUMN IF NOT EXISTS document_type_code TEXT;

ALTER TABLE metaldocs.documents
  ADD COLUMN IF NOT EXISTS business_unit TEXT;

ALTER TABLE metaldocs.documents
  ADD COLUMN IF NOT EXISTS department TEXT;

ALTER TABLE metaldocs.documents
  ADD COLUMN IF NOT EXISTS tags JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE metaldocs.documents
  ADD COLUMN IF NOT EXISTS effective_at TIMESTAMPTZ;

ALTER TABLE metaldocs.documents
  ADD COLUMN IF NOT EXISTS expiry_at TIMESTAMPTZ;

ALTER TABLE metaldocs.documents
  ADD COLUMN IF NOT EXISTS metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE metaldocs.documents
SET
  document_type_code = COALESCE(document_type_code, 'manual'),
  business_unit = COALESCE(NULLIF(TRIM(business_unit), ''), 'general'),
  department = COALESCE(NULLIF(TRIM(department), ''), 'general')
WHERE
  document_type_code IS NULL
  OR business_unit IS NULL
  OR TRIM(business_unit) = ''
  OR department IS NULL
  OR TRIM(department) = '';

ALTER TABLE metaldocs.documents
  ALTER COLUMN document_type_code SET NOT NULL;

ALTER TABLE metaldocs.documents
  ALTER COLUMN business_unit SET NOT NULL;

ALTER TABLE metaldocs.documents
  ALTER COLUMN department SET NOT NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'fk_documents_document_type_code'
  ) THEN
    ALTER TABLE metaldocs.documents
      ADD CONSTRAINT fk_documents_document_type_code
      FOREIGN KEY (document_type_code)
      REFERENCES metaldocs.document_types(code);
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_documents_document_type_code ON metaldocs.documents (document_type_code);
CREATE INDEX IF NOT EXISTS idx_documents_business_unit ON metaldocs.documents (business_unit);
CREATE INDEX IF NOT EXISTS idx_documents_department ON metaldocs.documents (department);
