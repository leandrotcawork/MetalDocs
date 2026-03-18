ALTER TABLE metaldocs.document_profiles
  ADD COLUMN IF NOT EXISTS alias TEXT;

UPDATE metaldocs.document_profiles
SET alias = CASE code
  WHEN 'po' THEN 'Procedimentos'
  WHEN 'it' THEN 'Instrucoes'
  WHEN 'rg' THEN 'Registros'
  ELSE TRIM(name)
END
WHERE alias IS NULL
   OR BTRIM(alias) = '';

ALTER TABLE metaldocs.document_profiles
  ALTER COLUMN alias SET NOT NULL;

ALTER TABLE metaldocs.document_profiles
  DROP CONSTRAINT IF EXISTS chk_document_profiles_alias_length;

ALTER TABLE metaldocs.document_profiles
  ADD CONSTRAINT chk_document_profiles_alias_length
  CHECK (char_length(alias) BETWEEN 1 AND 24);
