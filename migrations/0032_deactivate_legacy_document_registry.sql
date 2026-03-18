UPDATE metaldocs.document_profiles
SET is_active = FALSE
WHERE code IN (
  'policy',
  'procedure',
  'work_instruction',
  'contract',
  'supplier_document',
  'technical_drawing',
  'certificate',
  'report',
  'form',
  'manual'
);

UPDATE metaldocs.document_families
SET is_active = FALSE
WHERE code IN (
  'policy',
  'contract',
  'supplier_document',
  'technical_drawing',
  'certificate',
  'report',
  'form',
  'manual'
);
