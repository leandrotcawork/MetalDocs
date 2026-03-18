CREATE TABLE IF NOT EXISTS metaldocs.document_process_areas (
  code TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS metaldocs.document_subjects (
  code TEXT PRIMARY KEY,
  process_area_code TEXT NOT NULL REFERENCES metaldocs.document_process_areas(code),
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE metaldocs.documents
  ADD COLUMN IF NOT EXISTS process_area_code TEXT;

ALTER TABLE metaldocs.documents
  ADD COLUMN IF NOT EXISTS subject_code TEXT;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'fk_documents_process_area_code'
  ) THEN
    ALTER TABLE metaldocs.documents
      ADD CONSTRAINT fk_documents_process_area_code
      FOREIGN KEY (process_area_code)
      REFERENCES metaldocs.document_process_areas(code);
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'fk_documents_subject_code'
  ) THEN
    ALTER TABLE metaldocs.documents
      ADD CONSTRAINT fk_documents_subject_code
      FOREIGN KEY (subject_code)
      REFERENCES metaldocs.document_subjects(code);
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_documents_process_area_code ON metaldocs.documents (process_area_code);
CREATE INDEX IF NOT EXISTS idx_documents_subject_code ON metaldocs.documents (subject_code);
CREATE INDEX IF NOT EXISTS idx_document_subjects_process_area_code ON metaldocs.document_subjects (process_area_code);
