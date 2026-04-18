export interface CreateTemplateResponse { id: string; version_id: string; }

export async function createTemplate(key: string, name: string, description?: string): Promise<CreateTemplateResponse> {
  const res = await fetch('/api/v2/templates', {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ key, name, description }),
  });
  if (!res.ok) throw new Error(`create template failed: ${res.status}`);
  return res.json();
}

export type TemplateListRow = {
  id: string;
  key: string;
  name: string;
  description?: string;
  latest_version: number;
  latest_version_id: string;
  updated_at?: string;
};

export async function listTemplates(): Promise<TemplateListRow[]> {
  const res = await fetch('/api/v2/templates');
  if (!res.ok) throw new Error(`list failed: ${res.status}`);
  return res.json();
}

export async function presignDocxUpload(templateId: string, versionNum: number): Promise<{ url: string; storage_key: string }> {
  const res = await fetch(`/api/v2/templates/${templateId}/versions/${versionNum}/docx-upload-url`, { method: 'POST' });
  if (!res.ok) throw new Error(`presign failed: ${res.status}`);
  return res.json();
}

export async function presignSchemaUpload(templateId: string, versionNum: number): Promise<{ url: string; storage_key: string }> {
  const res = await fetch(`/api/v2/templates/${templateId}/versions/${versionNum}/schema-upload-url`, { method: 'POST' });
  if (!res.ok) throw new Error(`schema presign failed: ${res.status}`);
  return res.json();
}

export async function saveDraft(
  templateId: string, versionNum: number,
  body: { expected_lock_version: number; docx_storage_key: string; schema_storage_key: string; docx_content_hash: string; schema_content_hash: string; }
): Promise<void> {
  const res = await fetch(`/api/v2/templates/${templateId}/versions/${versionNum}/draft`, {
    method: 'PUT',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (res.status === 409) throw new Error('template_draft_stale');
  if (!res.ok) throw new Error(`save failed: ${res.status}`);
}

export interface PublishError {
  valid: false;
  parse_errors: Array<{ type: string; element?: string; ident?: string; }>;
  missing_tokens: string[];
  orphan_tokens: string[];
}

export interface PublishSuccess {
  published_version_id: string;
  next_draft_id: string;
  next_draft_version_num: number;
}

export async function publishVersion(
  templateId: string, versionNum: number, docxKey: string, schemaKey: string
): Promise<PublishSuccess | PublishError> {
  const res = await fetch(`/api/v2/templates/${templateId}/versions/${versionNum}/publish`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ docx_key: docxKey, schema_key: schemaKey }),
  });
  if (res.status === 422) return res.json() as Promise<PublishError>;
  if (!res.ok) throw new Error(`publish failed: ${res.status}`);
  return res.json() as Promise<PublishSuccess>;
}
