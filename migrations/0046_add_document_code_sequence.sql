ALTER TABLE metaldocs.documents
  ADD COLUMN IF NOT EXISTS document_sequence INT,
  ADD COLUMN IF NOT EXISTS document_code TEXT;

CREATE TABLE IF NOT EXISTS metaldocs.document_sequences (
  profile_code TEXT PRIMARY KEY REFERENCES metaldocs.document_profiles(code),
  next_value INT NOT NULL CHECK (next_value > 0)
);

WITH ranked AS (
  SELECT
    id,
    document_profile_code,
    ROW_NUMBER() OVER (
      PARTITION BY document_profile_code
      ORDER BY created_at, id
    ) AS seq
  FROM metaldocs.documents
  WHERE document_sequence IS NULL OR document_code IS NULL
)
UPDATE metaldocs.documents d
SET
  document_sequence = r.seq,
  document_code = UPPER(d.document_profile_code) || '-' || LPAD(r.seq::text, 3, '0')
FROM ranked r
WHERE d.id = r.id;

INSERT INTO metaldocs.document_sequences (profile_code, next_value)
SELECT
  p.code,
  COALESCE(MAX(d.document_sequence), 0) + 1
FROM metaldocs.document_profiles p
LEFT JOIN metaldocs.documents d ON d.document_profile_code = p.code
GROUP BY p.code
ON CONFLICT (profile_code) DO UPDATE
SET next_value = EXCLUDED.next_value;

ALTER TABLE metaldocs.documents
  ALTER COLUMN document_sequence SET NOT NULL,
  ALTER COLUMN document_code SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_documents_document_code_unique
  ON metaldocs.documents (document_code);

CREATE UNIQUE INDEX IF NOT EXISTS idx_documents_profile_sequence_unique
  ON metaldocs.documents (document_profile_code, document_sequence);
