export type VersionStatus = 'draft' | 'in_review' | 'approved' | 'published' | 'obsolete';

export interface TemplateDTO {
  id: string;
  tenant_id: string;
  doc_type_code: string | null;
  key: string;
  name: string;
  description: string | null;
  areas: string[];
  visibility: string;
  specific_areas: string[];
  latest_version: number;
  published_version_id: string | null;
  created_by: string;
  created_at: string;
  archived_at: string | null;
}

export interface VersionDTO {
  id: string;
  template_id: string;
  version_number: number;
  status: VersionStatus;
  docx_storage_key: string | null;
  content_hash: string | null;
  metadata_schema: Record<string, unknown> | null;
  placeholder_schema: Record<string, unknown> | null;
  editable_zones: Record<string, unknown> | null;
  author_id: string;
  pending_reviewer_role: string | null;
  pending_approver_role: string | null;
  reviewer_id: string | null;
  approver_id: string | null;
  submitted_at: string | null;
  reviewed_at: string | null;
  approved_at: string | null;
  published_at: string | null;
  obsoleted_at: string | null;
  created_at: string;
}

export type TemplateListRow = {
  id: string;
  key: string;
  name: string;
  description?: string;
  latest_version: number;
  latest_version_id?: string;
  published_version_id?: string | null;
  updated_at?: string;
  doc_type_code?: string | null;
  visibility: string;
  archived_at: string | null;
};

export interface PublishError {
  valid: false;
  parse_errors: Array<{ type: string; element?: string; ident?: string }>;
  missing_tokens: string[];
  orphan_tokens: string[];
}

export interface PublishSuccess {
  published_version_id: string;
  next_draft_id: string;
  next_draft_version_num: number;
}

async function apiJson<T>(res: Response): Promise<T> {
  if (!res.ok) {
    try {
      const body = (await res.json()) as { error?: { code?: string; message?: string } };
      const message = body?.error?.message;
      throw new Error(message || `HTTP ${res.status}`);
    } catch (err) {
      if (err instanceof Error) {
        throw err;
      }
      throw new Error(`HTTP ${res.status}`);
    }
  }

  return (await res.json()) as T;
}

export async function createTemplate(cmd: {
  key: string;
  name: string;
  description?: string;
  doc_type_code?: string;
  areas?: string[];
  visibility?: string;
  specific_areas?: string[];
  approver_role?: string;
  reviewer_role?: string | null;
}): Promise<{ template: TemplateDTO; version: VersionDTO }> {
  const res = await fetch('/api/v2/templates', {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify(cmd),
  });
  const body = await apiJson<{ data: { template: TemplateDTO; version: VersionDTO } }>(res);
  return body.data;
}

export async function listTemplates(params?: {
  limit?: number;
  offset?: number;
  doc_type?: string;
  area?: string[];
}): Promise<{ templates: TemplateDTO[]; meta: { limit: number; offset: number } }> {
  const qs = new URLSearchParams();
  if (params?.limit !== undefined) qs.set('limit', String(params.limit));
  if (params?.offset !== undefined) qs.set('offset', String(params.offset));
  if (params?.doc_type) qs.set('doc_type', params.doc_type);
  for (const area of params?.area ?? []) {
    qs.append('area', area);
  }

  const suffix = qs.toString() ? `?${qs.toString()}` : '';
  const res = await fetch(`/api/v2/templates${suffix}`);
  const body = await apiJson<{
    data: { templates: TemplateDTO[] };
    meta: { limit: number; offset: number };
  }>(res);

  return {
    templates: body.data.templates,
    meta: body.meta,
  };
}

export async function getTemplate(id: string): Promise<{ template: TemplateDTO; latest_version: VersionDTO }> {
  const res = await fetch(`/api/v2/templates/${id}`);
  const body = await apiJson<{ data: { template: TemplateDTO; latest_version: VersionDTO } }>(res);
  return body.data;
}

export async function getVersion(templateId: string, n: number): Promise<VersionDTO> {
  const res = await fetch(`/api/v2/templates/${templateId}/versions/${n}`);
  const body = await apiJson<{ data: { version: VersionDTO } }>(res);
  return body.data.version;
}

export async function presignAutosave(
  templateId: string,
  versionNum: number,
): Promise<{ upload_url: string; storage_key: string; expires_at: string }> {
  const res = await fetch(`/api/v2/templates/${templateId}/versions/${versionNum}/autosave/presign`, {
    method: 'POST',
  });
  const body = await apiJson<{
    data: { upload_url: string; storage_key: string; expires_at: string };
  }>(res);
  return body.data;
}

export async function commitAutosave(
  templateId: string,
  versionNum: number,
  expectedContentHash: string,
): Promise<VersionDTO> {
  const res = await fetch(`/api/v2/templates/${templateId}/versions/${versionNum}/autosave/commit`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ expected_content_hash: expectedContentHash }),
  });
  const body = await apiJson<{ data: { version: VersionDTO } }>(res);
  return body.data.version;
}

export async function presignDocxUpload(
  templateId: string,
  versionNum: number,
): Promise<{ url: string; storage_key: string }> {
  const r = await presignAutosave(templateId, versionNum);
  return { url: r.upload_url, storage_key: r.storage_key };
}

export async function presignSchemaUpload(
  templateId: string,
  versionNum: number,
): Promise<{ url: string; storage_key: string }> {
  const r = await presignAutosave(templateId, versionNum);
  return { url: r.upload_url, storage_key: r.storage_key };
}

export async function saveDraft(
  templateId: string,
  versionNum: number,
  body: {
    expected_lock_version: number;
    docx_storage_key: string;
    schema_storage_key: string;
    docx_content_hash: string;
    schema_content_hash: string;
  },
): Promise<void> {
  await commitAutosave(templateId, versionNum, body.docx_content_hash || body.schema_content_hash);
}

export async function publishVersion(
  templateId: string,
  versionNum: number,
  docxKey: string,
  schemaKey: string,
): Promise<PublishSuccess | PublishError> {
  const res = await fetch(`/api/v2/templates/${templateId}/versions/${versionNum}/publish`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ docx_key: docxKey, schema_key: schemaKey }),
  });
  if (res.status === 422) {
    return (await res.json()) as PublishError;
  }
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}`);
  }
  return (await res.json()) as PublishSuccess;
}

export async function getDocxURL(templateId: string, versionNum: number): Promise<string> {
  const res = await fetch(`/api/v2/templates/${templateId}/versions/${versionNum}/docx-url`);
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error((body as any)?.error?.message || `HTTP ${res.status}`);
  }
  const body = (await res.json()) as { data: { url: string } };
  return body.data.url;
}

export async function submitForReview(templateId: string, versionNum: number): Promise<VersionDTO> {
  const res = await fetch(`/api/v2/templates/${templateId}/versions/${versionNum}/submit`, {
    method: 'POST',
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error((body as any)?.error?.message || `HTTP ${res.status}`);
  }
  const data = (await res.json()) as { data: { version: VersionDTO } };
  return data.data.version;
}

export async function reviewVersion(
  templateId: string,
  versionNum: number,
  accept: boolean,
  reason?: string,
): Promise<VersionDTO> {
  const res = await fetch(`/api/v2/templates/${templateId}/versions/${versionNum}/review`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ accept, reason: reason || '' }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error((body as any)?.error?.message || `HTTP ${res.status}`);
  }
  const data = (await res.json()) as { data: { version: VersionDTO } };
  return data.data.version;
}

export async function approveVersion(
  templateId: string,
  versionNum: number,
  accept: boolean,
  reason?: string,
): Promise<VersionDTO> {
  const res = await fetch(`/api/v2/templates/${templateId}/versions/${versionNum}/approve`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ accept, reason: reason || '' }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error((body as any)?.error?.message || `HTTP ${res.status}`);
  }
  const data = (await res.json()) as { data: { version: VersionDTO } };
  return data.data.version;
}
