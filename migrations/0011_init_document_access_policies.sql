CREATE TABLE IF NOT EXISTS metaldocs.document_access_policies (
  id BIGSERIAL PRIMARY KEY,
  subject_type TEXT NOT NULL,
  subject_id TEXT NOT NULL,
  resource_scope TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  capability TEXT NOT NULL,
  effect TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_document_access_policies_subject_type
    CHECK (subject_type IN ('user', 'role', 'group')),
  CONSTRAINT chk_document_access_policies_resource_scope
    CHECK (resource_scope IN ('document', 'document_type', 'area')),
  CONSTRAINT chk_document_access_policies_capability
    CHECK (capability IN (
      'document.create',
      'document.view',
      'document.edit',
      'document.upload_attachment',
      'document.change_workflow',
      'document.manage_permissions'
    )),
  CONSTRAINT chk_document_access_policies_effect
    CHECK (effect IN ('allow', 'deny')),
  CONSTRAINT uq_document_access_policies_rule
    UNIQUE (subject_type, subject_id, resource_scope, resource_id, capability)
);

CREATE INDEX IF NOT EXISTS idx_document_access_policies_resource
  ON metaldocs.document_access_policies (resource_scope, resource_id);

CREATE INDEX IF NOT EXISTS idx_document_access_policies_subject
  ON metaldocs.document_access_policies (subject_type, subject_id);
