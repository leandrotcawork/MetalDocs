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
  UserRole,
} from "./lib.types";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

const allowedRoles = new Set<UserRole>(["admin", "editor", "reviewer", "viewer"]);

function normalizeRoles(value: unknown): UserRole[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((item): item is UserRole => typeof item === "string" && allowedRoles.has(item as UserRole));
}

function normalizeCurrentUser(value: CurrentUser): CurrentUser {
  return {
    userId: value?.userId ?? "",
    username: value?.username ?? "",
    email: value?.email ?? "",
    displayName: value?.displayName ?? value?.username ?? "",
    mustChangePassword: Boolean(value?.mustChangePassword),
    roles: normalizeRoles(value?.roles),
  };
}

function normalizeManagedUser(value: ManagedUserItem): ManagedUserItem {
  return {
    userId: value?.userId ?? "",
    username: value?.username ?? "",
    email: value?.email ?? "",
    displayName: value?.displayName ?? value?.username ?? "",
    isActive: Boolean(value?.isActive),
    mustChangePassword: Boolean(value?.mustChangePassword),
    failedLoginAttempts: Number(value?.failedLoginAttempts ?? 0),
    lockedUntil: value?.lockedUntil ?? "",
    lastLoginAt: value?.lastLoginAt ?? "",
    createdAt: value?.createdAt ?? "",
    updatedAt: value?.updatedAt ?? "",
    roles: normalizeRoles(value?.roles),
  };
}

function normalizeStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((item): item is string => typeof item === "string");
}

function normalizeDocumentProfile(value: DocumentProfileItem): DocumentProfileItem {
  return {
    code: value?.code ?? "",
    familyCode: value?.familyCode ?? "",
    name: value?.name ?? value?.code ?? "",
    description: value?.description ?? "",
    reviewIntervalDays: Number(value?.reviewIntervalDays ?? 0),
    activeSchemaVersion: Number(value?.activeSchemaVersion ?? 0),
    workflowProfile: value?.workflowProfile ?? "",
    approvalRequired: Boolean(value?.approvalRequired),
    retentionDays: Number(value?.retentionDays ?? 0),
    validityDays: Number(value?.validityDays ?? 0),
  };
}

function normalizeProcessArea(value: ProcessAreaItem): ProcessAreaItem {
  return {
    code: value?.code ?? "",
    name: value?.name ?? value?.code ?? "",
    description: value?.description ?? "",
  };
}

function normalizeMetadataRule(value: DocumentProfileSchemaItem["metadataRules"][number]) {
  return {
    name: value?.name ?? "",
    type: value?.type ?? "text",
    required: Boolean(value?.required),
  };
}

function normalizeDocumentProfileSchema(value: DocumentProfileSchemaItem): DocumentProfileSchemaItem {
  return {
    profileCode: value?.profileCode ?? "",
    version: Number(value?.version ?? 0),
    isActive: Boolean(value?.isActive),
    metadataRules: Array.isArray(value?.metadataRules) ? value.metadataRules.map(normalizeMetadataRule) : [],
  };
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

function normalizeApprovalItem(value: WorkflowApprovalItem): WorkflowApprovalItem {
  return {
    approvalId: value?.approvalId ?? "",
    documentId: value?.documentId ?? "",
    requestedBy: value?.requestedBy ?? "",
    assignedReviewer: value?.assignedReviewer ?? "",
    decisionBy: value?.decisionBy ?? "",
    status: value?.status ?? "PENDING",
    requestReason: value?.requestReason ?? "",
    decisionReason: value?.decisionReason ?? "",
    requestedAt: value?.requestedAt ?? "",
    decidedAt: value?.decidedAt ?? "",
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
  login: async (body: { identifier: string; password: string }) => {
    const response = await request<{ user: CurrentUser; expiresAt: string }>("/auth/login", { method: "POST", body: JSON.stringify(body) });
    return { ...response, user: normalizeCurrentUser(response.user) };
  },
  logout: () => request<void>("/auth/logout", { method: "POST" }),
  me: async () => normalizeCurrentUser(await request<CurrentUser>("/auth/me")),
  changePassword: async (body: { currentPassword: string; newPassword: string }) => {
    const response = await request<{ changed: boolean; user: CurrentUser }>("/auth/change-password", { method: "POST", body: JSON.stringify(body) });
    return { ...response, user: normalizeCurrentUser(response.user) };
  },
  listUsers: async () => {
    const response = await request<{ items: ManagedUserItem[] }>("/iam/users");
    return { items: Array.isArray(response.items) ? response.items.map(normalizeManagedUser) : [] };
  },
  createUser: (body: Record<string, unknown>) => request<{ userId: string }>("/iam/users", { method: "POST", body: JSON.stringify(body) }),
  updateUser: (userId: string, body: Record<string, unknown>) => request<{ userId: string; updated: boolean }>(`/iam/users/${userId}`, { method: "PATCH", body: JSON.stringify(body) }),
  assignRole: (userId: string, body: Record<string, unknown>) => request<{ userId: string; role: string; displayName: string }>(`/iam/users/${userId}/roles`, { method: "POST", body: JSON.stringify(body) }),
  listDocumentTypes: () => request<{ items: DocumentTypeItem[] }>("/document-types"),
  listDocumentFamilies: () => request<{ items: DocumentFamilyItem[] }>("/document-families"),
  listDocumentProfiles: async () => {
    const response = await request<{ items: DocumentProfileItem[] }>("/document-profiles");
    return { items: Array.isArray(response.items) ? response.items.map(normalizeDocumentProfile) : [] };
  },
  getDocumentProfileSchema: async (profileCode: string) => {
    const response = await request<{ items: DocumentProfileSchemaItem[] }>(`/document-profiles/${encodeURIComponent(profileCode)}/schema`);
    const items = Array.isArray(response.items) ? response.items.map(normalizeDocumentProfileSchema) : [];
    return items.find((item) => item.isActive) ?? items[0] ?? null;
  },
  getDocumentProfileGovernance: async (profileCode: string) => normalizeDocumentProfileGovernance(await request<DocumentProfileGovernanceItem>(`/document-profiles/${encodeURIComponent(profileCode)}/governance`)),
  listProcessAreas: async () => {
    const response = await request<{ items: ProcessAreaItem[] }>("/process-areas");
    return { items: Array.isArray(response.items) ? response.items.map(normalizeProcessArea) : [] };
  },
  listDocuments: async () => {
    const response = await request<{ items: DocumentListItem[] }>("/documents");
    return { items: Array.isArray(response.items) ? response.items.map(normalizeDocumentListItem) : [] };
  },
  searchDocuments: async (params: URLSearchParams) => {
    const response = await request<{ items: SearchDocumentItem[] }>(`/search/documents?${params.toString()}`);
    return { items: Array.isArray(response.items) ? response.items.map(normalizeSearchDocument) : [] };
  },
  getDocument: async (documentId: string) => normalizeDocumentListItem(await request<DocumentListItem>(`/documents/${documentId}`)),
  createDocument: (body: Record<string, unknown>) => request<{ documentId: string; version: number; status: string; documentType: string; documentProfile: string; documentFamily: string; profileSchemaVersion: number; processArea?: string; subject?: string }>("/documents", { method: "POST", body: JSON.stringify(body) }),
  listVersions: async (documentId: string) => {
    const response = await request<{ items: VersionListItem[] }>(`/documents/${documentId}/versions`);
    return { items: Array.isArray(response.items) ? response.items.map(normalizeVersionItem) : [] };
  },
  addVersion: (documentId: string, body: Record<string, unknown>) => request<VersionListItem>(`/documents/${documentId}/versions`, { method: "POST", body: JSON.stringify(body) }),
  listApprovals: async (documentId: string) => {
    const response = await request<{ items: WorkflowApprovalItem[] }>(`/workflow/documents/${documentId}/approvals`);
    return { items: Array.isArray(response.items) ? response.items.map(normalizeApprovalItem) : [] };
  },
  transitionWorkflow: (documentId: string, body: Record<string, unknown>) => request<Record<string, unknown>>(`/workflow/documents/${documentId}/transitions`, { method: "POST", body: JSON.stringify(body) }),
  listAttachments: async (documentId: string) => {
    const response = await request<{ items: AttachmentItem[] }>(`/documents/${documentId}/attachments`);
    return { items: Array.isArray(response.items) ? response.items.map(normalizeAttachmentItem) : [] };
  },
  uploadAttachment: (documentId: string, file: File) => {
    const formData = new FormData();
    formData.append("file", file);
    return request<AttachmentItem>(`/documents/${documentId}/attachments`, { method: "POST", body: formData });
  },
  getAttachmentDownloadURL: (documentId: string, attachmentId: string) => request<{ attachmentId: string; downloadUrl: string; expiresAt: string }>(`/documents/${documentId}/attachments/${attachmentId}/download-url`),
  listAccessPolicies: async (resourceScope: string, resourceId: string) => {
    const response = await request<{ items: AccessPolicyItem[] }>(`/access-policies?resourceScope=${encodeURIComponent(resourceScope)}&resourceId=${encodeURIComponent(resourceId)}`);
    return { items: Array.isArray(response.items) ? response.items.map(normalizeAccessPolicyItem) : [] };
  },
  replaceAccessPolicies: (body: Record<string, unknown>) => request<{ replacedCount: number }>("/access-policies", { method: "PUT", body: JSON.stringify(body) }),
};
