import { request, requestBlob, API_BASE_URL } from "./client";

// ---------------------------------------------------------------------------
// DTOs
// ---------------------------------------------------------------------------

export interface TemplateDraftDTO {
  templateKey: string;
  profileCode: string;
  name: string;
  status: string; // "draft"
  lockVersion: number;
  hasStrippedFields: boolean;
  blocks: unknown; // JSON
  theme?: unknown;
  meta?: unknown;
  updatedAt: string;
}

export interface TemplateVersionDTO {
  templateKey: string;
  version: number;
  profileCode: string;
  name: string;
  status: string; // "published" | "deprecated"
}

export interface TemplateListItemDTO {
  templateKey: string;
  version: number;
  profileCode: string;
  name: string;
  status: string;
}

export interface PublishErrorDTO {
  blockId: string;
  blockType: string;
  field: string;
  reason: string;
}

export interface StrippedFieldDTO {
  blockId: string;
  blockType: string;
  field: string;
  reason: string;
}

export interface ImportResultDTO {
  templateKey: string;
  hasStrippedFields: boolean;
  strippedFields: StrippedFieldDTO[];
}

// ---------------------------------------------------------------------------
// Typed error helpers
// ---------------------------------------------------------------------------

/**
 * Thrown when the server returns 409 (optimistic-lock conflict).
 * Callers can use `instanceof TemplateLockConflictError` to handle it.
 */
export class TemplateLockConflictError extends Error {
  readonly status = 409;
  constructor(message = "Lock conflict — the template was modified by someone else") {
    super(message);
    this.name = "TemplateLockConflictError";
  }
}

/**
 * Thrown when the server returns 422 (publish validation failed).
 * `errors` contains the field-level issues returned by the server.
 */
export class TemplatePublishValidationError extends Error {
  readonly status = 422;
  readonly errors: PublishErrorDTO[];
  constructor(errors: PublishErrorDTO[], message = "Template has validation errors that prevent publishing") {
    super(message);
    this.name = "TemplatePublishValidationError";
    this.errors = errors;
  }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function encodeKey(key: string): string {
  return encodeURIComponent(key);
}

/**
 * Internal fetch wrapper for endpoints that can return 409 or 422 with
 * structured error bodies that need special error types.
 */
async function requestWithStructuredErrors<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    credentials: "include",
    ...init,
    headers: {
      ...(init?.body instanceof FormData ? {} : { "Content-Type": "application/json" }),
      ...(init?.headers ?? {}),
    },
  });

  if (!response.ok) {
    const body = await response.json().catch(() => null) as Record<string, unknown> | null;

    if (response.status === 409) {
      const message = (body as { error?: { message?: string } } | null)?.error?.message;
      throw new TemplateLockConflictError(message ?? undefined);
    }

    if (response.status === 422) {
      const errors = (body as { errors?: PublishErrorDTO[] } | null)?.errors ?? [];
      const message = (body as { error?: { message?: string } } | null)?.error?.message;
      throw new TemplatePublishValidationError(errors, message ?? undefined);
    }

    // Fall back to generic error for all other non-ok statuses
    const message = (body as { error?: { message?: string } } | null)?.error?.message;
    const error = new Error(message ?? `HTTP ${response.status}`);
    (error as Error & { status?: number }).status = response.status;
    throw error;
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

// ---------------------------------------------------------------------------
// API functions — 13 endpoints
// ---------------------------------------------------------------------------

/**
 * GET /templates?profileCode=<code>
 * List all templates for a given profile.
 */
export async function listTemplates(profileCode: string): Promise<TemplateListItemDTO[]> {
  const body = await request<{ items: TemplateListItemDTO[] }>(`/templates?profileCode=${encodeURIComponent(profileCode)}`);
  return body.items ?? [];
}

/**
 * POST /templates
 * Create a new draft template.
 */
export function createTemplate(profileCode: string, name: string): Promise<TemplateDraftDTO> {
  return request<TemplateDraftDTO>("/templates", {
    method: "POST",
    body: JSON.stringify({ profileCode, name }),
  });
}

/**
 * GET /templates/:key
 * Retrieve a template (draft or latest published version).
 */
export function getTemplate(key: string): Promise<TemplateDraftDTO | TemplateVersionDTO> {
  return request<TemplateDraftDTO | TemplateVersionDTO>(`/templates/${encodeKey(key)}`);
}

/**
 * PUT /templates/:key/draft
 * Save draft content. May throw TemplateLockConflictError on 409.
 */
export function saveDraft(
  key: string,
  payload: { blocks: unknown; theme?: unknown; meta?: unknown; lockVersion: number },
): Promise<TemplateDraftDTO> {
  return requestWithStructuredErrors<TemplateDraftDTO>(`/templates/${encodeKey(key)}/draft`, {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

/**
 * POST /templates/:key/publish
 * Publish the current draft. May throw TemplateLockConflictError (409) or
 * TemplatePublishValidationError (422).
 */
export function publishTemplate(key: string, lockVersion: number): Promise<TemplateVersionDTO> {
  return requestWithStructuredErrors<TemplateVersionDTO>(`/templates/${encodeKey(key)}/publish`, {
    method: "POST",
    body: JSON.stringify({ lockVersion }),
  });
}

/**
 * POST /templates/:key/edit
 * Create a new draft from a published version so it can be edited.
 */
export function editPublished(key: string): Promise<TemplateDraftDTO> {
  return request<TemplateDraftDTO>(`/templates/${encodeKey(key)}/edit`, {
    method: "POST",
    body: JSON.stringify({}),
  });
}

/**
 * POST /templates/:key/deprecate
 * Deprecate a specific published version.
 */
export function deprecateTemplate(key: string, version: number): Promise<void> {
  return request<void>(`/templates/${encodeKey(key)}/deprecate`, {
    method: "POST",
    body: JSON.stringify({ version }),
  });
}

/**
 * POST /templates/:key/clone
 * Clone an existing template under a new name.
 */
export function cloneTemplate(key: string, newName: string): Promise<TemplateDraftDTO> {
  return request<TemplateDraftDTO>(`/templates/${encodeKey(key)}/clone`, {
    method: "POST",
    body: JSON.stringify({ name: newName }),
  });
}

/**
 * DELETE /templates/:key/draft
 * Permanently delete a draft (no published version exists).
 */
export function deleteDraft(key: string): Promise<void> {
  return request<void>(`/templates/${encodeKey(key)}/draft`, {
    method: "DELETE",
  });
}

/**
 * POST /templates/:key/discard
 * Discard an in-progress draft, reverting to the last published version.
 */
export function discardDraft(key: string): Promise<void> {
  return request<void>(`/templates/${encodeKey(key)}/discard`, {
    method: "POST",
    body: JSON.stringify({}),
  });
}

/**
 * POST /templates/:key/acknowledge-stripped
 * Acknowledge that the user has reviewed stripped fields after import.
 * May throw TemplateLockConflictError on 409.
 */
export function acknowledgeStripped(key: string, lockVersion: number): Promise<TemplateDraftDTO> {
  return requestWithStructuredErrors<TemplateDraftDTO>(`/templates/${encodeKey(key)}/acknowledge-stripped`, {
    method: "POST",
    body: JSON.stringify({ lockVersion }),
  });
}

/**
 * GET /templates/:key/export?version=<n>
 * Download a template snapshot as a binary blob (JSON file download).
 */
export function exportTemplate(key: string, version: number): Promise<Blob> {
  return requestBlob(`/templates/${encodeKey(key)}/export?version=${encodeURIComponent(String(version))}`);
}

/**
 * POST /templates/:key/preview-docx
 * Render a .docx preview from the current draft's blocks.
 * Returns a Blob for browser download.
 */
export function previewTemplateDocx(key: string): Promise<Blob> {
  return requestBlob(`/templates/${encodeKey(key)}/preview-docx`, { method: "POST" });
}

/**
 * POST /templates/import?profileCode=<code>
 * Import a template from an uploaded file (multipart/form-data).
 */
export function importTemplate(profileCode: string, file: File): Promise<ImportResultDTO> {
  const form = new FormData();
  form.append("file", file);
  return request<ImportResultDTO>(`/templates/import?profileCode=${encodeURIComponent(profileCode)}`, {
    method: "POST",
    body: form,
    // Content-Type is intentionally omitted so the browser sets the correct
    // multipart boundary. The request() helper already handles FormData by
    // skipping the application/json header.
  });
}
