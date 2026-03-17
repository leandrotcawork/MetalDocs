import type {
  AccessPolicyItem,
  AttachmentItem,
  DocumentListItem,
  DocumentTypeItem,
  SearchDocumentItem,
  VersionListItem,
  WorkflowApprovalItem,
  ApiErrorEnvelope,
} from "./lib.types";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://192.168.0.3:8080/api/v1";
const USER_ID = import.meta.env.VITE_USER_ID ?? "admin-local";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers: {
      "X-User-Id": USER_ID,
      ...(init?.body instanceof FormData ? {} : { "Content-Type": "application/json" }),
      ...(init?.headers ?? {}),
    },
  });

  if (!response.ok) {
    const errorPayload = (await response.json().catch(() => null)) as ApiErrorEnvelope | null;
    throw new Error(errorPayload?.error.message ?? `HTTP ${response.status}`);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

export const api = {
  currentUserId: USER_ID,
  listDocumentTypes: () => request<{ items: DocumentTypeItem[] }>("/document-types"),
  listDocuments: () => request<{ items: DocumentListItem[] }>("/documents"),
  searchDocuments: (params: URLSearchParams) => request<{ items: SearchDocumentItem[] }>(`/search/documents?${params.toString()}`),
  getDocument: (documentId: string) => request<DocumentListItem>(`/documents/${documentId}`),
  createDocument: (body: Record<string, unknown>) => request<{ documentId: string; version: number; status: string; documentType: string }>("/documents", { method: "POST", body: JSON.stringify(body) }),
  listVersions: (documentId: string) => request<{ items: VersionListItem[] }>(`/documents/${documentId}/versions`),
  addVersion: (documentId: string, body: Record<string, unknown>) => request<VersionListItem>(`/documents/${documentId}/versions`, { method: "POST", body: JSON.stringify(body) }),
  listApprovals: (documentId: string) => request<{ items: WorkflowApprovalItem[] }>(`/workflow/documents/${documentId}/approvals`),
  transitionWorkflow: (documentId: string, body: Record<string, unknown>) => request<Record<string, unknown>>(`/workflow/documents/${documentId}/transitions`, { method: "POST", body: JSON.stringify(body) }),
  listAttachments: (documentId: string) => request<{ items: AttachmentItem[] }>(`/documents/${documentId}/attachments`),
  uploadAttachment: (documentId: string, file: File) => {
    const formData = new FormData();
    formData.append("file", file);
    return request<AttachmentItem>(`/documents/${documentId}/attachments`, { method: "POST", body: formData });
  },
  getAttachmentDownloadURL: (documentId: string, attachmentId: string) => request<{ attachmentId: string; downloadUrl: string; expiresAt: string }>(`/documents/${documentId}/attachments/${attachmentId}/download-url`),
  listAccessPolicies: (resourceScope: string, resourceId: string) => request<{ items: AccessPolicyItem[] }>(`/access-policies?resourceScope=${encodeURIComponent(resourceScope)}&resourceId=${encodeURIComponent(resourceId)}`),
  replaceAccessPolicies: (body: Record<string, unknown>) => request<{ replacedCount: number }>("/access-policies", { method: "PUT", body: JSON.stringify(body) }),
};
