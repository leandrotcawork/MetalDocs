import type {
  AccessPolicyItem,
  AttachmentItem,
  DocumentBrowserContentSaveResponse,
  DocumentBrowserEditorBundleResponse,
  DocumentBrowserTemplateSnapshotItem,
  CollaborationPresenceItem,
  DocumentContentDocxResponse,
  DocumentContentNativeResponse,
  DocumentContentPdfResponse,
  DocumentContentSaveResponse,
  DocumentContentUploadResponse,
  DocumentEditorBundleResponse,
  DocumentEditLockItem,
  DocumentListItem,
  DocumentProfileGovernanceItem,
  DocumentProfileSchemaItem,
  DocumentTemplateAssignmentItem,
  DocumentTemplateItem,
  DocumentTemplateSnapshotItem,
  SearchDocumentItem,
  VersionDiffResponse,
  VersionListItem,
} from "../lib.types";
import type {
  SchemaDocumentContentSaveResponse,
  SchemaDocumentEditorBundleResponse,
  SchemaDocumentTypeBundleResponse,
} from "../features/documents/runtime/schemaRuntimeTypes";
import { normalizeDocumentTypeSchema, normalizeSchemaDocumentEditorBundle } from "../features/documents/runtime/schemaRuntimeAdapters";
import { normalizeDocumentProfileCode } from "../features/shared/documentProfile";
import { request, requestBlob } from "./client";

function normalizeStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((item): item is string => typeof item === "string");
}

function normalizeDocumentProfileSchema(value: DocumentProfileSchemaItem): DocumentProfileSchemaItem {
  return {
    profileCode: value?.profileCode ?? "",
    version: Number(value?.version ?? 0),
    isActive: Boolean(value?.isActive),
    metadataRules: Array.isArray(value?.metadataRules) ? value.metadataRules.map((rule) => ({
      name: rule?.name ?? "",
      type: rule?.type ?? "text",
      required: Boolean(rule?.required),
    })) : [],
    contentSchema: value?.contentSchema && typeof value.contentSchema === "object" ? value.contentSchema : {},
  };
}

function normalizeRuntimeSchemaBundle(value: SchemaDocumentTypeBundleResponse): SchemaDocumentTypeBundleResponse {
  return {
    typeKey: value?.typeKey ?? "",
    name: value?.name ?? "",
    description: value?.description ?? "",
    activeVersion: typeof value?.activeVersion === "number" ? value.activeVersion : null,
    schema: normalizeDocumentTypeSchema(value?.schema),
  };
}

function normalizeRuntimeEditorBundle(value: SchemaDocumentEditorBundleResponse): SchemaDocumentEditorBundleResponse {
  return normalizeSchemaDocumentEditorBundle(value);
}

function normalizeDocumentProfileGovernance(value: DocumentProfileGovernanceItem): DocumentProfileGovernanceItem {
  return {
    profileCode: value?.profileCode ?? "",
    workflowProfile: value?.workflowProfile ?? "",
    reviewIntervalDays: Number(value?.reviewIntervalDays ?? 0),
    approvalRequired: Boolean(value?.approvalRequired),
    retentionDays: Number(value?.retentionDays ?? 0),
    validityDays: Number(value?.validityDays ?? 0),
  };
}

function normalizeDocumentListItem<T extends DocumentListItem>(value: T): T {
  return {
    ...value,
    documentId: value?.documentId ?? "",
    title: value?.title ?? "",
    documentType: value?.documentType ?? "",
    documentProfile: value?.documentProfile ?? value?.documentType ?? "",
    documentFamily: value?.documentFamily ?? "",
    documentSequence: Number(value?.documentSequence ?? 0),
    documentCode: value?.documentCode ?? "",
    profileSchemaVersion: Number(value?.profileSchemaVersion ?? 0),
    processArea: value?.processArea ?? "",
    subject: value?.subject ?? "",
    ownerId: value?.ownerId ?? "",
    businessUnit: value?.businessUnit ?? "",
    department: value?.department ?? "",
    classification: value?.classification ?? "INTERNAL",
    status: value?.status ?? "DRAFT",
    tags: normalizeStringArray(value?.tags),
    effectiveAt: value?.effectiveAt ?? "",
    expiryAt: value?.expiryAt ?? "",
  } as T;
}

function normalizeSearchDocument(value: SearchDocumentItem): SearchDocumentItem {
  return {
    ...normalizeDocumentListItem(value),
    createdAt: value?.createdAt ?? "",
  };
}

function normalizeVersionItem(value: VersionListItem): VersionListItem {
  return {
    documentId: value?.documentId ?? "",
    version: Number(value?.version ?? 0),
    contentHash: value?.contentHash ?? "",
    changeSummary: value?.changeSummary ?? "",
    createdAt: value?.createdAt ?? "",
  };
}

function normalizeAttachmentItem(value: AttachmentItem): AttachmentItem {
  return {
    attachmentId: value?.attachmentId ?? "",
    documentId: value?.documentId ?? "",
    fileName: value?.fileName ?? "",
    contentType: value?.contentType ?? "application/octet-stream",
    sizeBytes: Number(value?.sizeBytes ?? 0),
    uploadedBy: value?.uploadedBy ?? "",
    createdAt: value?.createdAt ?? "",
  };
}

function normalizeCollaborationPresenceItem(value: CollaborationPresenceItem): CollaborationPresenceItem {
  return {
    documentId: value?.documentId ?? "",
    userId: value?.userId ?? "",
    displayName: value?.displayName ?? value?.userId ?? "",
    lastSeenAt: value?.lastSeenAt ?? "",
  };
}

function normalizeDocumentEditLockItem(value: DocumentEditLockItem): DocumentEditLockItem {
  return {
    documentId: value?.documentId ?? "",
    lockedBy: value?.lockedBy ?? "",
    displayName: value?.displayName ?? value?.lockedBy ?? "",
    lockReason: value?.lockReason ?? "",
    acquiredAt: value?.acquiredAt ?? "",
    expiresAt: value?.expiresAt ?? "",
  };
}

function normalizeDocumentTemplateSnapshot(value: DocumentTemplateSnapshotItem): DocumentTemplateSnapshotItem {
  return {
    templateKey: value?.templateKey ?? "",
    version: Number(value?.version ?? 0),
    profileCode: value?.profileCode ?? "",
    schemaVersion: Number(value?.schemaVersion ?? 0),
    definition: value?.definition && typeof value.definition === "object" && !Array.isArray(value.definition) ? value.definition : {},
  };
}

function normalizeDocumentBrowserTemplateSnapshot(value: DocumentBrowserTemplateSnapshotItem): DocumentBrowserTemplateSnapshotItem {
  if (value?.editor !== "ckeditor5") {
    throw new Error("Browser editor template snapshot has unsupported editor.");
  }

  if (value?.contentFormat !== "html") {
    throw new Error("Browser editor template snapshot has unsupported content format.");
  }

  return {
    templateKey: value?.templateKey ?? "",
    version: Number(value?.version ?? 0),
    profileCode: value?.profileCode ?? "",
    schemaVersion: Number(value?.schemaVersion ?? 0),
    editor: value.editor,
    contentFormat: value.contentFormat,
    body: typeof value?.body === "string" ? value.body : "",
  };
}

function normalizeDocumentTemplateItem(value: DocumentTemplateItem): DocumentTemplateItem {
  return {
    templateKey: value?.templateKey ?? "",
    version: Number(value?.version ?? 0),
    profileCode: value?.profileCode ?? "",
    schemaVersion: Number(value?.schemaVersion ?? 0),
    name: value?.name ?? "",
    editor: typeof value?.editor === "string" && value.editor.trim() ? value.editor : undefined,
    contentFormat: typeof value?.contentFormat === "string" && value.contentFormat.trim() ? value.contentFormat : undefined,
  };
}

function normalizeDocumentTemplateAssignmentItem(value: DocumentTemplateAssignmentItem): DocumentTemplateAssignmentItem {
  return {
    documentId: value?.documentId ?? "",
    templateKey: value?.templateKey ?? "",
    templateVersion: Number(value?.templateVersion ?? 0),
    assignedAt: value?.assignedAt ?? "",
  };
}

function normalizeAccessPolicyItem(value: AccessPolicyItem): AccessPolicyItem {
  return {
    subjectType: value?.subjectType ?? "user",
    subjectId: value?.subjectId ?? "",
    resourceScope: value?.resourceScope ?? "document",
    resourceId: value?.resourceId ?? "",
    capability: value?.capability ?? "document.view",
    effect: value?.effect ?? "deny",
  };
}

function normalizeVersionDiff(value: VersionDiffResponse): VersionDiffResponse {
  return {
    documentId: value?.documentId ?? "",
    fromVersion: Number(value?.fromVersion ?? 0),
    toVersion: Number(value?.toVersion ?? 0),
    contentChanged: Boolean(value?.contentChanged),
    metadataChanged: normalizeStringArray(value?.metadataChanged),
    classificationChanged: Boolean(value?.classificationChanged),
    effectiveAtChanged: Boolean(value?.effectiveAtChanged),
    expiryAtChanged: Boolean(value?.expiryAtChanged),
  };
}

function normalizeDocumentEditorBundle(value: DocumentEditorBundleResponse): DocumentEditorBundleResponse {
  const document = normalizeDocumentListItem(value?.document);
  const templateSnapshot = value?.templateSnapshot ? normalizeDocumentTemplateSnapshot(value.templateSnapshot) : undefined;
  const draftToken = typeof value?.draftToken === "string" && value.draftToken.trim() ? value.draftToken.trim() : "";
  const documentProfileCode = normalizeDocumentProfileCode(document.documentProfile);
  const templateProfileCode = normalizeDocumentProfileCode(templateSnapshot?.profileCode);
  const hasTemplateSnapshot = Boolean(templateSnapshot);
  const hasDraftToken = Boolean(draftToken);

  if (hasTemplateSnapshot !== hasDraftToken) {
    throw new Error("Governed canvas bundle missing template snapshot or draft token.");
  }

  if (templateProfileCode && templateProfileCode !== documentProfileCode) {
    throw new Error("Governed canvas template snapshot profile mismatch.");
  }

  return {
    document,
    versions: Array.isArray(value?.versions) ? value.versions.map(normalizeVersionItem) : [],
    schema: normalizeDocumentProfileSchema(value?.schema),
    governance: normalizeDocumentProfileGovernance(value?.governance),
    templateSnapshot,
    draftToken: draftToken || undefined,
    presence: Array.isArray(value?.presence) ? value.presence.map(normalizeCollaborationPresenceItem) : [],
    editLock: value?.editLock ? normalizeDocumentEditLockItem(value.editLock) : undefined,
  };
}

function normalizeDocumentBrowserEditorBundle(value: DocumentBrowserEditorBundleResponse): DocumentBrowserEditorBundleResponse {
  const document = normalizeDocumentListItem(value?.document);
  const templateSnapshot = normalizeDocumentBrowserTemplateSnapshot(value?.templateSnapshot);
  const draftToken = typeof value?.draftToken === "string" ? value.draftToken.trim() : "";
  const body = typeof value?.body === "string" ? value.body : "";
  const documentProfileCode = normalizeDocumentProfileCode(document.documentProfile);
  const templateProfileCode = normalizeDocumentProfileCode(templateSnapshot.profileCode);

  if (!draftToken) {
    throw new Error("Browser editor bundle missing draft token.");
  }

  if (!templateSnapshot.templateKey || !templateSnapshot.version) {
    throw new Error("Browser editor bundle missing template snapshot.");
  }

  if (templateProfileCode && templateProfileCode !== documentProfileCode) {
    throw new Error("Browser editor template snapshot profile mismatch.");
  }

  return {
    document,
    versions: Array.isArray(value?.versions) ? value.versions.map(normalizeVersionItem) : [],
    governance: normalizeDocumentProfileGovernance(value?.governance),
    templateSnapshot,
    body,
    draftToken,
  };
}

export async function listDocuments() {
  const response = await request<{ items: DocumentListItem[] }>("/documents");
  return { items: Array.isArray(response.items) ? response.items.map(normalizeDocumentListItem) : [] };
}

export async function searchDocuments(params: URLSearchParams) {
  const response = await request<{ items: SearchDocumentItem[] }>(`/search/documents?${params.toString()}`);
  return { items: Array.isArray(response.items) ? response.items.map(normalizeSearchDocument) : [] };
}

export async function getDocument(documentId: string) {
  return normalizeDocumentListItem(await request<DocumentListItem>(`/documents/${documentId}`));
}

export async function getDocumentEditorBundle(documentId: string) {
  return normalizeDocumentEditorBundle(
    await request<DocumentEditorBundleResponse>(`/documents/${encodeURIComponent(documentId)}/editor-bundle`),
  );
}

export async function getDocumentBrowserEditorBundle(documentId: string) {
  return normalizeDocumentBrowserEditorBundle(
    await request<DocumentBrowserEditorBundleResponse>(`/documents/${encodeURIComponent(documentId)}/browser-editor-bundle`),
  );
}

export async function listDocumentTemplates(profileCode?: string) {
  const params = new URLSearchParams();
  if (typeof profileCode === "string" && profileCode.trim()) {
    params.set("profileCode", profileCode.trim());
  }
  const suffix = params.toString();
  const response = await request<{ items: DocumentTemplateItem[] }>(`/document-templates${suffix ? `?${suffix}` : ""}`);
  return { items: Array.isArray(response.items) ? response.items.map(normalizeDocumentTemplateItem) : [] };
}

export async function assignDocumentTemplate(documentId: string, body: { templateKey: string; templateVersion: number }) {
  return normalizeDocumentTemplateAssignmentItem(
    await request<DocumentTemplateAssignmentItem>(`/documents/${encodeURIComponent(documentId)}/template-assignment`, {
      method: "PUT",
      body: JSON.stringify(body),
    }),
  );
}

export async function fetchDocumentTypeBundle(typeKey: string) {
  return normalizeRuntimeSchemaBundle(
    await request<SchemaDocumentTypeBundleResponse>(`/document-types/${encodeURIComponent(typeKey)}/bundle`),
  );
}

export async function fetchDocumentEditorBundle(documentId: string) {
  const response = await request<SchemaDocumentEditorBundleResponse>(`/documents/${encodeURIComponent(documentId)}/editor-bundle`);
  return normalizeRuntimeEditorBundle(response);
}

export async function saveDocumentContent(documentId: string, values: Record<string, unknown>) {
  return request<SchemaDocumentContentSaveResponse>(`/documents/${encodeURIComponent(documentId)}/content`, {
    method: "PUT",
    body: JSON.stringify({ values }),
  });
}

export function createDocument(body: Record<string, unknown>) {
  return request<{
    documentId: string;
    version: number;
    status: string;
    documentType: string;
    documentProfile: string;
    documentFamily: string;
    documentSequence: number;
    documentCode: string;
    profileSchemaVersion: number;
    processArea?: string;
    subject?: string;
  }>("/documents", { method: "POST", body: JSON.stringify(body) });
}

export async function listVersions(documentId: string) {
  const response = await request<{ items: VersionListItem[] }>(`/documents/${documentId}/versions`);
  return { items: Array.isArray(response.items) ? response.items.map(normalizeVersionItem) : [] };
}

export function addVersion(documentId: string, body: Record<string, unknown>) {
  return request<VersionListItem>(`/documents/${documentId}/versions`, { method: "POST", body: JSON.stringify(body) });
}

export async function getVersionDiff(documentId: string, fromVersion: number, toVersion: number) {
  const response = await request<VersionDiffResponse>(
    `/documents/${documentId}/versions/diff?fromVersion=${encodeURIComponent(String(fromVersion))}&toVersion=${encodeURIComponent(String(toVersion))}`,
  );
  return normalizeVersionDiff(response);
}

export async function listAttachments(documentId: string) {
  const response = await request<{ items: AttachmentItem[] }>(`/documents/${documentId}/attachments`);
  return { items: Array.isArray(response.items) ? response.items.map(normalizeAttachmentItem) : [] };
}

export function heartbeatDocumentPresence(documentId: string, body?: Record<string, unknown>) {
  return request<{ ok: boolean }>(`/documents/${documentId}/collaboration/presence`, {
    method: "POST",
    body: JSON.stringify(body ?? {}),
  });
}

export async function listDocumentPresence(documentId: string) {
  const response = await request<{ items: CollaborationPresenceItem[] }>(
    `/documents/${documentId}/collaboration/presence`,
  );
  return { items: Array.isArray(response.items) ? response.items.map(normalizeCollaborationPresenceItem) : [] };
}

export async function acquireDocumentEditLock(documentId: string, body?: Record<string, unknown>) {
  return normalizeDocumentEditLockItem(
    await request<DocumentEditLockItem>(`/documents/${documentId}/collaboration/lock`, {
      method: "POST",
      body: JSON.stringify(body ?? {}),
    }),
  );
}

export async function getDocumentEditLock(documentId: string) {
  return normalizeDocumentEditLockItem(
    await request<DocumentEditLockItem>(`/documents/${documentId}/collaboration/lock`),
  );
}

export function releaseDocumentEditLock(documentId: string) {
  return request<void>(`/documents/${documentId}/collaboration/lock`, { method: "DELETE" });
}

export function uploadAttachment(documentId: string, file: File) {
  const formData = new FormData();
  formData.append("file", file);
  return request<AttachmentItem>(`/documents/${documentId}/attachments`, { method: "POST", body: formData });
}

export function getAttachmentDownloadURL(documentId: string, attachmentId: string) {
  return request<{ attachmentId: string; downloadUrl: string; expiresAt: string }>(
    `/documents/${documentId}/attachments/${attachmentId}/download-url`,
  );
}

export function getDocumentContentNative(documentId: string) {
  return request<DocumentContentNativeResponse>(`/documents/${documentId}/content/native`);
}

export function saveDocumentContentNative(documentId: string, body: { content: Record<string, unknown>; draftToken?: string }) {
  return request<DocumentContentSaveResponse>(`/documents/${documentId}/content/native`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function saveDocumentBrowserContent(documentId: string, body: { body: string; draftToken: string }) {
  return request<DocumentBrowserContentSaveResponse>(
    `/documents/${encodeURIComponent(documentId)}/content/browser`,
    {
      method: "POST",
      body: JSON.stringify(body),
    },
  );
}

export function getDocumentContentPdf(documentId: string) {
  return request<DocumentContentPdfResponse>(`/documents/${documentId}/content/pdf`);
}

export function renderDocumentContentPdf(documentId: string) {
  return request<DocumentContentPdfResponse>(`/documents/${documentId}/content/render-pdf`, { method: "POST" });
}

export function getDocumentContentDocx(documentId: string) {
  return request<DocumentContentDocxResponse>(`/documents/${documentId}/content/docx`);
}

export function uploadDocumentContentDocx(documentId: string, file: File) {
  const formData = new FormData();
  formData.append("file", file);
  return request<DocumentContentUploadResponse>(`/documents/${documentId}/content/upload`, {
    method: "POST",
    body: formData,
  });
}

export function downloadDocumentTemplateDocx(documentId: string) {
  return requestBlob(`/documents/${encodeURIComponent(documentId)}/template/docx`);
}

export function exportDocumentDocx(documentId: string) {
  return requestBlob(`/documents/${encodeURIComponent(documentId)}/export/docx`, {
    method: "POST",
  });
}

export async function listAccessPolicies(resourceScope: string, resourceId: string) {
  const response = await request<{ items: AccessPolicyItem[] }>(
    `/access-policies?resourceScope=${encodeURIComponent(resourceScope)}&resourceId=${encodeURIComponent(resourceId)}`,
  );
  return { items: Array.isArray(response.items) ? response.items.map(normalizeAccessPolicyItem) : [] };
}

export function replaceAccessPolicies(body: Record<string, unknown>) {
  return request<{ replacedCount: number }>("/access-policies", { method: "PUT", body: JSON.stringify(body) });
}
