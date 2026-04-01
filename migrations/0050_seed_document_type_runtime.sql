INSERT INTO metaldocs.document_types (type_key, name, description, family_key, active_version)
VALUES ('po', 'Procedimento Operacional', 'Runtime type', 'procedure', 1)
ON CONFLICT (type_key) DO UPDATE
SET
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  family_key = EXCLUDED.family_key,
  active_version = EXCLUDED.active_version;
