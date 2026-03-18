import type {
  AccessPolicyItem,
  ApiErrorEnvelope,
  AttachmentItem,
  CurrentUser,
  DocumentFamilyItem,
  DocumentListItem,
  DocumentProfileGovernanceItem,
  DocumentProfileItem,
  DocumentProfileSchemaItem,
  DocumentTypeItem,
  ManagedUserItem,
  ProcessAreaItem,
  SearchDocumentItem,
  VersionListItem,
  WorkflowApprovalItem,
} from "./lib.types";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    credentials: "include",
    ...init,
    headers: {
      ...(init?.body instanceof FormData ? {} : { "Content-Type": "application/json" }),
      ...(init?.headers ?? {}),
    },
  });

  if (!response.ok) {
    const errorPayload = (await response.json().catch(() => null)) as ApiErrorEnvelope | null;
    const error = new Error(errorPayload?.error.message ?? `HTTP ${response.status}`);
    (error as Error & { status?: number }).status = response.status;
    throw error;
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

export const api = {
  currentApiBaseUrl: API_BASE_URL,
  login: (body: { identifier: string; password: string }) => request<{ user: CurrentUser; expiresAt: string }>("/auth/login", { method: "POST", body: JSON.stringify(body) }),
  logout: () => request<void>("/auth/logout", { method: "POST" }),
  me: () => request<CurrentUser>("/auth/me"),
  changePassword: (body: { currentPassword: string; newPassword: string }) => request<{ changed: boolean; user: CurrentUser }>("/auth/change-password", { method: "POST", body: JSON.stringify(body) }),
  listUsers: () => request<{ items: ManagedUserItem[] }>("/iam/users"),
  createUser: (body: Record<string, unknown>) => request<{ userId: string }>("/iam/users", { method: "POST", body: JSON.stringify(body) }),
  updateUser: (userId: string, body: Record<string, unknown>) => request<{ userId: string; updated: boolean }>(`/iam/users/${userId}`, { method: "PATCH", body: JSON.stringify(body) }),
  assignRole: (userId: string, body: Record<string, unknown>) => request<{ userId: string; role: string; displayName: string }>(`/iam/users/${userId}/roles`, { method: "POST", body: JSON.stringify(body) }),
  listDocumentTypes: () => request<{ items: DocumentTypeItem[] }>("/document-types"),
  listDocumentFamilies: () => request<{ items: DocumentFamilyItem[] }>("/document-families"),
  listDocumentProfiles: () => request<{ items: DocumentProfileItem[] }>("/document-profiles"),
  getDocumentProfileSchema: (profileCode: string) => request<DocumentProfileSchemaItem>(`/document-profiles/${encodeURIComponent(profileCode)}/schema`),
  getDocumentProfileGovernance: (profileCode: string) => request<DocumentProfileGovernanceItem>(`/document-profiles/${encodeURIComponent(profileCode)}/governance`),
  listProcessAreas: () => request<{ items: ProcessAreaItem[] }>("/process-areas"),
  listDocuments: () => request<{ items: DocumentListItem[] }>("/documents"),
  searchDocuments: (params: URLSearchParams) => request<{ items: SearchDocumentItem[] }>(`/search/documents?${params.toString()}`),
  getDocument: (documentId: string) => request<DocumentListItem>(`/documents/${documentId}`),
  createDocument: (body: Record<string, unknown>) => request<{ documentId: string; version: number; status: string; documentType: string; documentProfile: string; documentFamily: string; profileSchemaVersion: number; processArea?: string; subject?: string }>("/documents", { method: "POST", body: JSON.stringify(body) }),
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
