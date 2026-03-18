CREATE TABLE IF NOT EXISTS metaldocs.document_families (
  code TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS metaldocs.document_profiles (
  code TEXT PRIMARY KEY,
  family_code TEXT NOT NULL REFERENCES metaldocs.document_families(code),
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  review_interval_days INT NOT NULL CHECK (review_interval_days > 0),
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO metaldocs.document_families (code, name, description)
VALUES
  ('policy', 'Policy', 'High-level governance and policy document'),
  ('procedure', 'Procedure', 'Operational procedure with controlled steps'),
  ('work_instruction', 'Work Instruction', 'Detailed execution instruction'),
  ('contract', 'Contract', 'Commercial or legal agreement'),
  ('supplier_document', 'Supplier Document', 'Document received from supplier'),
  ('technical_drawing', 'Technical Drawing', 'Engineering drawing or technical artifact'),
  ('certificate', 'Certificate', 'Certificate with issuer and validity context'),
  ('report', 'Report', 'Periodic or ad-hoc report'),
  ('form', 'Form', 'Structured business form'),
  ('manual', 'Manual', 'Reference or guidance manual')
ON CONFLICT (code) DO NOTHING;

INSERT INTO metaldocs.document_profiles (code, family_code, name, description, review_interval_days)
SELECT dt.code, dt.code, dt.name, dt.description, dt.review_interval_days
FROM metaldocs.document_types dt
ON CONFLICT (code) DO NOTHING;

ALTER TABLE metaldocs.documents
  ADD COLUMN IF NOT EXISTS document_profile_code TEXT;

ALTER TABLE metaldocs.documents
  ADD COLUMN IF NOT EXISTS document_family_code TEXT;

UPDATE metaldocs.documents d
SET
  document_profile_code = COALESCE(d.document_profile_code, d.document_type_code),
  document_family_code = COALESCE(d.document_family_code, dp.family_code)
FROM metaldocs.document_profiles dp
WHERE dp.code = d.document_type_code
  AND (
    d.document_profile_code IS NULL
    OR d.document_family_code IS NULL
  );

ALTER TABLE metaldocs.documents
  ALTER COLUMN document_profile_code SET NOT NULL;

ALTER TABLE metaldocs.documents
  ALTER COLUMN document_family_code SET NOT NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'fk_documents_document_profile_code'
  ) THEN
    ALTER TABLE metaldocs.documents
      ADD CONSTRAINT fk_documents_document_profile_code
      FOREIGN KEY (document_profile_code)
      REFERENCES metaldocs.document_profiles(code);
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'fk_documents_document_family_code'
  ) THEN
    ALTER TABLE metaldocs.documents
      ADD CONSTRAINT fk_documents_document_family_code
      FOREIGN KEY (document_family_code)
      REFERENCES metaldocs.document_families(code);
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_documents_document_profile_code ON metaldocs.documents (document_profile_code);
CREATE INDEX IF NOT EXISTS idx_documents_document_family_code ON metaldocs.documents (document_family_code);
