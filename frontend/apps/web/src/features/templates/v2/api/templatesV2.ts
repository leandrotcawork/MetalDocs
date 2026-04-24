export type VersionStatus = 'draft' | 'in_review' | 'approved' | 'published' | 'obsolete';

import type { Placeholder, EditableZone, CompositionConfig } from '../../../placeholder-types';
export type { Placeholder, EditableZone, CompositionConfig };

export interface TemplateSchemas {
  placeholders: Placeholder[];
  zones: EditableZone[];
  composition: CompositionConfig | null;
}

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

// Wire-format types (backend snake_case)
interface WireContentPolicy { allow_tables: boolean; allow_images: boolean; allow_headings: boolean; allow_lists: boolean; }
interface WireZone { id: string; label: string; required: boolean; content_policy: WireContentPolicy; max_length?: number; }
interface WirePlaceholder { id: string; label: string; type: string; required: boolean; options?: string[]; regex?: string; min_number?: number; max_number?: number; min_date?: string; max_date?: string; max_length?: number; resolver_key?: string; visible_if?: { placeholder_id: string; op: string; value?: unknown }; }

function zoneFromWire(w: WireZone): EditableZone {
  return {
    id: w.id,
    label: w.label,
    ...(w.max_length != null ? { maxLength: w.max_length } : {}),
    contentPolicy: {
      allowTables: w.content_policy.allow_tables,
      allowImages: w.content_policy.allow_images,
      allowHeadings: w.content_policy.allow_headings,
      allowLists: w.content_policy.allow_lists,
    },
  };
}

function zoneToWire(z: EditableZone): WireZone {
  return {
    id: z.id,
    label: z.label,
    required: false,
    content_policy: {
      allow_tables: z.contentPolicy.allowTables,
      allow_images: z.contentPolicy.allowImages,
      allow_headings: z.contentPolicy.allowHeadings,
      allow_lists: z.contentPolicy.allowLists,
    },
    ...(z.maxLength != null ? { max_length: z.maxLength } : {}),
  };
}

function placeholderFromWire(w: WirePlaceholder): Placeholder {
  return {
    id: w.id,
    label: w.label,
    type: w.type as Placeholder['type'],
    ...(w.required ? { required: true } : {}),
    ...(w.options ? { options: w.options } : {}),
    ...(w.regex != null ? { regex: w.regex } : {}),
    ...(w.min_number != null ? { minNumber: w.min_number } : {}),
    ...(w.max_number != null ? { maxNumber: w.max_number } : {}),
    ...(w.min_date != null ? { minDate: w.min_date } : {}),
    ...(w.max_date != null ? { maxDate: w.max_date } : {}),
    ...(w.max_length != null ? { maxLength: w.max_length } : {}),
    ...(w.resolver_key != null ? { resolverKey: w.resolver_key } : {}),
    ...(w.visible_if ? { visibleIf: { placeholderID: w.visible_if.placeholder_id, operator: w.visible_if.op as Placeholder['visibleIf']['operator'], value: w.visible_if.value as string | undefined } } : {}),
  };
}

function placeholderToWire(p: Placeholder): WirePlaceholder {
  return {
    id: p.id,
    label: p.label,
    type: p.type,
    required: p.required ?? false,
    ...(p.options ? { options: p.options } : {}),
    ...(p.regex != null ? { regex: p.regex } : {}),
    ...(p.minNumber != null ? { min_number: p.minNumber } : {}),
    ...(p.maxNumber != null ? { max_number: p.maxNumber } : {}),
    ...(p.minDate != null ? { min_date: p.minDate } : {}),
    ...(p.maxDate != null ? { max_date: p.maxDate } : {}),
    ...(p.maxLength != null ? { max_length: p.maxLength } : {}),
    ...(p.resolverKey != null ? { resolver_key: p.resolverKey } : {}),
    ...(p.visibleIf ? { visible_if: { placeholder_id: p.visibleIf.placeholderID, op: p.visibleIf.operator, value: p.visibleIf.value } } : {}),
  };
}

export async function getTemplateSchemas(templateId: string, versionNum: number): Promise<TemplateSchemas> {
  const res = await fetch(`/api/v2/templates/${templateId}/versions/${versionNum}`);
  const body = await apiJson<{ data: { version: VersionDTO & { placeholder_schema: WirePlaceholder[] | null; editable_zones: WireZone[] | null } } }>(res);
  const v = body.data.version;
  return {
    placeholders: Array.isArray(v.placeholder_schema) ? v.placeholder_schema.map(placeholderFromWire) : [],
    zones: Array.isArray(v.editable_zones) ? v.editable_zones.map(zoneFromWire) : [],
    composition: null,
  };
}

export async function putTemplateSchemas(templateId: string, versionNum: number, schemas: TemplateSchemas): Promise<void> {
  const res = await fetch(`/api/v2/templates/${templateId}/versions/${versionNum}/schema`, {
    method: 'PUT',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({
      metadata_schema: {},
      placeholder_schema: schemas.placeholders.map(placeholderToWire),
      editable_zones: schemas.zones.map(zoneToWire),
      expected_content_hash: '',
    }),
  });
  await apiJson<unknown>(res);
}
