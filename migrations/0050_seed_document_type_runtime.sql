INSERT INTO metaldocs.document_types (type_key, name, description, family_key, active_version)
VALUES
  ('po', 'Procedimento Operacional', 'Runtime type', 'procedure', 1),
  ('it', 'Instrucao de Trabalho', 'Runtime type', 'work_instruction', 1),
  ('rg', 'Registro', 'Runtime type', 'record', 1)
ON CONFLICT (type_key) DO UPDATE
SET
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  family_key = EXCLUDED.family_key,
  active_version = EXCLUDED.active_version;

INSERT INTO metaldocs.document_type_schema_versions (type_key, version, schema_json, governance_json)
VALUES
  (
    'po',
    1,
    $${"sections":[{"key":"identificacao","num":"1","title":"Identificação","fields":[{"key":"elaboradoPor","label":"Elaborado por","type":"text"}]}]}$$::jsonb,
    '{}'::jsonb
  ),
  (
    'it',
    1,
    '{"sections":[{"key":"contexto","num":"1","title":"Contexto"}]}'::jsonb,
    '{}'::jsonb
  ),
  (
    'rg',
    1,
    '{"sections":[{"key":"evento","num":"1","title":"Evento"}]}'::jsonb,
    '{}'::jsonb
  )
ON CONFLICT (type_key, version) DO NOTHING;
