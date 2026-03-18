INSERT INTO metaldocs.document_types (code, name, description, review_interval_days)
VALUES
  ('po', 'PO', 'Procedimento operacional da Metal Nobre', 365),
  ('it', 'IT', 'Instrucao de trabalho da Metal Nobre', 180),
  ('rg', 'RG', 'Registro operacional da Metal Nobre', 365)
ON CONFLICT (code) DO NOTHING;

WITH legacy_documents AS (
  SELECT id
  FROM metaldocs.documents
  WHERE document_type_code IN (
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
    )
    OR document_profile_code IN (
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
    )
)
DELETE FROM metaldocs.workflow_approvals
WHERE document_id IN (SELECT id FROM legacy_documents);

WITH legacy_documents AS (
  SELECT id
  FROM metaldocs.documents
  WHERE document_type_code IN (
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
    )
    OR document_profile_code IN (
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
    )
)
DELETE FROM metaldocs.document_attachments
WHERE document_id IN (SELECT id FROM legacy_documents);

WITH legacy_documents AS (
  SELECT id
  FROM metaldocs.documents
  WHERE document_type_code IN (
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
    )
    OR document_profile_code IN (
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
    )
)
DELETE FROM metaldocs.document_versions
WHERE document_id IN (SELECT id FROM legacy_documents);

WITH legacy_documents AS (
  SELECT id
  FROM metaldocs.documents
  WHERE document_type_code IN (
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
    )
    OR document_profile_code IN (
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
    )
)
DELETE FROM metaldocs.document_access_policies
WHERE (resource_scope = 'document' AND resource_id IN (SELECT id FROM legacy_documents))
   OR (resource_scope = 'document_type' AND resource_id IN (
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
   ));

WITH legacy_documents AS (
  SELECT id
  FROM metaldocs.documents
  WHERE document_type_code IN (
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
    )
    OR document_profile_code IN (
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
    )
)
DELETE FROM metaldocs.notifications
WHERE resource_type = 'document'
  AND resource_id IN (SELECT id FROM legacy_documents);

WITH legacy_documents AS (
  SELECT id
  FROM metaldocs.documents
  WHERE document_type_code IN (
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
    )
    OR document_profile_code IN (
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
    )
)
DELETE FROM metaldocs.audit_events
WHERE resource_type = 'document'
  AND resource_id IN (SELECT id FROM legacy_documents);

DELETE FROM metaldocs.documents
WHERE document_type_code IN (
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
  )
  OR document_profile_code IN (
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

DELETE FROM metaldocs.document_profile_governance
WHERE profile_code IN (
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

DELETE FROM metaldocs.document_profile_schema_versions
WHERE profile_code IN (
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

DELETE FROM metaldocs.document_profiles
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

DELETE FROM metaldocs.document_types
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

DELETE FROM metaldocs.document_families
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
