UPDATE metaldocs.document_profiles
SET
  name = CASE code
    WHEN 'po' THEN 'Procedimento Operacional'
    WHEN 'it' THEN 'Instrucao de Trabalho'
    WHEN 'rg' THEN 'Registro'
    ELSE name
  END,
  description = CASE code
    WHEN 'po' THEN 'Procedimento operacional da Metal Nobre'
    WHEN 'it' THEN 'Instrucao de trabalho da Metal Nobre'
    WHEN 'rg' THEN 'Registro operacional da Metal Nobre'
    ELSE description
  END
WHERE code IN ('po', 'it', 'rg');

UPDATE metaldocs.document_types
SET
  name = CASE code
    WHEN 'po' THEN 'Procedimento Operacional'
    WHEN 'it' THEN 'Instrucao de Trabalho'
    WHEN 'rg' THEN 'Registro'
    ELSE name
  END,
  description = CASE code
    WHEN 'po' THEN 'Procedimento operacional da Metal Nobre'
    WHEN 'it' THEN 'Instrucao de trabalho da Metal Nobre'
    WHEN 'rg' THEN 'Registro operacional da Metal Nobre'
    ELSE description
  END
WHERE code IN ('po', 'it', 'rg');
