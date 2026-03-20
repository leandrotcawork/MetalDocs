import type {
  AccessPolicyItem,
  AuditEventItem,
  ApiErrorEnvelope,
  AttachmentItem,
  CollaborationPresenceItem,
  CurrentUser,
  DocumentFamilyItem,
  DocumentEditLockItem,
  DocumentListItem,
  DocumentProfileGovernanceItem,
  DocumentProfileItem,
  DocumentProfileSchemaItem,
  DocumentTypeItem,
  DocumentDepartmentItem,
  DocumentContentDocxResponse,
  DocumentContentNativeResponse,
  DocumentContentPdfResponse,
  DocumentContentSaveResponse,
  DocumentContentUploadResponse,
  ManagedUserItem,
  NotificationItem,
  ProcessAreaItem,
  SearchDocumentItem,
  SubjectItem,
  VersionDiffResponse,
  VersionListItem,
  WorkflowApprovalItem,
  UserRole,
} from "./lib.types";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

export type OperationsStreamSnapshot = {
  pendingNotifications: number;
  pendingApprovals: number;
  documentsInReview: number;
  totalDocuments: number;
  generatedAt: string;
};

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
  const fallbackName = value?.name ?? value?.code ?? "";
  return {
    code: value?.code ?? "",
    familyCode: value?.familyCode ?? "",
    name: fallbackName,
    alias: value?.alias?.trim?.() || fallbackName,
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

function normalizeDocumentDepartment(value: DocumentDepartmentItem): DocumentDepartmentItem {
  return {
    code: value?.code ?? "",
    name: value?.name ?? value?.code ?? "",
    description: value?.description ?? "",
  };
}

function normalizeSubject(value: SubjectItem): SubjectItem {
  return {
    code: value?.code ?? "",
    processAreaCode: value?.processAreaCode ?? "",
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
    contentSchema: value?.contentSchema && typeof value.contentSchema === "object" ? value.contentSchema : {},
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

function normalizeNotificationItem(value: NotificationItem): NotificationItem {
  return {
    id: value?.id ?? "",
    recipientUserId: value?.recipientUserId ?? "",
    eventType: value?.eventType ?? "",
    resourceType: value?.resourceType ?? "",
    resourceId: value?.resourceId ?? "",
    title: value?.title ?? "",
    message: value?.message ?? "",
    status: value?.status ?? "PENDING",
    createdAt: value?.createdAt ?? "",
    readAt: value?.readAt ?? "",
  };
}

function normalizeAuditEventItem(value: AuditEventItem): AuditEventItem {
  return {
    id: value?.id ?? "",
    occurredAt: value?.occurredAt ?? "",
    actorId: value?.actorId ?? "",
    action: value?.action ?? "",
    resourceType: value?.resourceType ?? "",
    resourceId: value?.resourceId ?? "",
    payload: typeof value?.payload === "object" && value?.payload !== null ? value.payload : {},
    traceId: value?.traceId ?? "",
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

type RequestTrace = {
  id: number;
  method: string;
  path: string;
  startedAt: number;
};

let traceId = 0;
let activeTrace: { name: string; startedAt: number; items: RequestTrace[] } | null = null;

function isTraceEnabled() {
  if (typeof window === "undefined") return false;
  if (window.location.search.includes("trace=1")) return true;
  return localStorage.getItem("md_trace") === "1";
}

function traceStart(name: string) {
  if (!isTraceEnabled()) return;
  activeTrace = { name, startedAt: performance.now(), items: [] };
}

function traceStop() {
  if (!activeTrace) return;
  const total = performance.now() - activeTrace.startedAt;
  console.groupCollapsed(`[md-trace] ${activeTrace.name} (${total.toFixed(0)}ms)`);
  activeTrace.items
    .sort((a, b) => a.startedAt - b.startedAt)
    .forEach((item) => {
      console.log(`${item.method} ${item.path}`, `+${(item.startedAt - activeTrace!.startedAt).toFixed(0)}ms`);
    });
  console.groupEnd();
  activeTrace = null;
}

function traceRequestStart(method: string, path: string) {
  if (!activeTrace) return;
  const item: RequestTrace = {
    id: traceId++,
    method,
    path,
    startedAt: performance.now(),
  };
  activeTrace.items.push(item);
  return item.id;
}

function traceRequestEnd(id?: number) {
  if (!activeTrace || id === undefined) return;
  const item = activeTrace.items.find((it) => it.id === id);
  if (item) {
    item.startedAt = item.startedAt;
  }
}

export function startApiTrace(name: string) {
  traceStart(name);
}

export function stopApiTrace() {
  traceStop();
}

export function markUx(label: string) {
  if (!isTraceEnabled()) return;
  console.log(`[md-ux] ${label} @ ${performance.now().toFixed(0)}ms`);
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const method = (init?.method ?? "GET").toUpperCase();
  const traceItemId = traceRequestStart(method, path);
  const response = await fetch(`${API_BASE_URL}${path}`, {
    credentials: "include",
    ...init,
    headers: {
      ...(init?.body instanceof FormData ? {} : { "Content-Type": "application/json" }),
      ...(init?.headers ?? {}),
    },
  });
  traceRequestEnd(traceItemId);

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

async function requestBlob(path: string, init?: RequestInit): Promise<Blob> {
  const method = (init?.method ?? "GET").toUpperCase();
  const traceItemId = traceRequestStart(method, path);
  const response = await fetch(`${API_BASE_URL}${path}`, {
    credentials: "include",
    ...init,
    headers: {
      ...(init?.headers ?? {}),
    },
  });
  traceRequestEnd(traceItemId);

  if (!response.ok) {
    const errorPayload = (await response.json().catch(() => null)) as ApiErrorEnvelope | null;
    const error = new Error(errorPayload?.error.message ?? `HTTP ${response.status}`);
    (error as Error & { status?: number }).status = response.status;
    throw error;
  }

  return response.blob();
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
  replaceUserRoles: (userId: string, body: Record<string, unknown>) => request<{ userId: string; displayName: string; roles: string[] }>(`/iam/users/${userId}/roles`, { method: "PUT", body: JSON.stringify(body) }),
  adminResetPassword: (userId: string, body: Record<string, unknown>) => request<{ userId: string; reset: boolean; mustChangePassword: boolean }>(`/iam/users/${userId}/reset-password`, { method: "POST", body: JSON.stringify(body) }),
  unlockUser: (userId: string) => request<{ userId: string; unlocked: boolean }>(`/iam/users/${userId}/unlock`, { method: "POST" }),
  listDocumentTypes: () => request<{ items: DocumentTypeItem[] }>("/document-types"),
  listDocumentFamilies: () => request<{ items: DocumentFamilyItem[] }>("/document-families"),
  listDocumentProfiles: async () => {
    const response = await request<{ items: DocumentProfileItem[] }>("/document-profiles");
    return { items: Array.isArray(response.items) ? response.items.map(normalizeDocumentProfile) : [] };
  },
  createDocumentProfile: (body: Record<string, unknown>) => request<{ code: string }>("/document-profiles", { method: "POST", body: JSON.stringify(body) }),
  updateDocumentProfile: (code: string, body: Record<string, unknown>) => request<{ code: string }>(`/document-profiles/${encodeURIComponent(code)}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteDocumentProfile: (code: string) => request<void>(`/document-profiles/${encodeURIComponent(code)}`, { method: "DELETE" }),
  getDocumentProfileSchema: async (profileCode: string) => {
    const response = await request<{ items: DocumentProfileSchemaItem[] }>(`/document-profiles/${encodeURIComponent(profileCode)}/schema`);
    const items = Array.isArray(response.items) ? response.items.map(normalizeDocumentProfileSchema) : [];
    return items.find((item) => item.isActive) ?? items[0] ?? null;
  },
  listDocumentProfileSchemas: async (profileCode: string) => {
    const response = await request<{ items: DocumentProfileSchemaItem[] }>(`/document-profiles/${encodeURIComponent(profileCode)}/schema`);
    return { items: Array.isArray(response.items) ? response.items.map(normalizeDocumentProfileSchema) : [] };
  },
  upsertDocumentProfileSchema: (profileCode: string, body: Record<string, unknown>) => request<{ code: string }>(`/document-profiles/${encodeURIComponent(profileCode)}/schema`, { method: "POST", body: JSON.stringify(body) }),
  activateDocumentProfileSchema: (profileCode: string, version: number) => request<{ code: string }>(`/document-profiles/${encodeURIComponent(profileCode)}/schema/${encodeURIComponent(String(version))}/activate`, { method: "PUT" }),
  getDocumentProfileGovernance: async (profileCode: string) => normalizeDocumentProfileGovernance(await request<DocumentProfileGovernanceItem>(`/document-profiles/${encodeURIComponent(profileCode)}/governance`)),
  updateDocumentProfileGovernance: (profileCode: string, body: Record<string, unknown>) => request<{ code: string }>(`/document-profiles/${encodeURIComponent(profileCode)}/governance`, { method: "PUT", body: JSON.stringify(body) }),
  listProcessAreas: async () => {
    const response = await request<{ items: ProcessAreaItem[] }>("/process-areas");
    return { items: Array.isArray(response.items) ? response.items.map(normalizeProcessArea) : [] };
  },
  listDocumentDepartments: async () => {
    const response = await request<{ items: DocumentDepartmentItem[] }>("/document-departments");
    return { items: Array.isArray(response.items) ? response.items.map(normalizeDocumentDepartment) : [] };
  },
  createProcessArea: (body: Record<string, unknown>) => request<{ code: string }>("/process-areas", { method: "POST", body: JSON.stringify(body) }),
  updateProcessArea: (code: string, body: Record<string, unknown>) => request<{ code: string }>(`/process-areas/${encodeURIComponent(code)}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteProcessArea: (code: string) => request<void>(`/process-areas/${encodeURIComponent(code)}`, { method: "DELETE" }),
  listSubjects: async (params?: URLSearchParams) => {
    const query = params?.toString();
    const response = await request<{ items: SubjectItem[] }>(`/document-subjects${query ? `?${query}` : ""}`);
    return { items: Array.isArray(response.items) ? response.items.map(normalizeSubject) : [] };
  },
  createSubject: (body: Record<string, unknown>) => request<{ code: string }>("/document-subjects", { method: "POST", body: JSON.stringify(body) }),
  updateSubject: (code: string, body: Record<string, unknown>) => request<{ code: string }>(`/document-subjects/${encodeURIComponent(code)}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteSubject: (code: string) => request<void>(`/document-subjects/${encodeURIComponent(code)}`, { method: "DELETE" }),
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
  getVersionDiff: async (documentId: string, fromVersion: number, toVersion: number) => {
    const response = await request<VersionDiffResponse>(`/documents/${documentId}/versions/diff?fromVersion=${encodeURIComponent(String(fromVersion))}&toVersion=${encodeURIComponent(String(toVersion))}`);
    return normalizeVersionDiff(response);
  },
  listApprovals: async (documentId: string) => {
    const response = await request<{ items: WorkflowApprovalItem[] }>(`/workflow/documents/${documentId}/approvals`);
    return { items: Array.isArray(response.items) ? response.items.map(normalizeApprovalItem) : [] };
  },
  transitionWorkflow: (documentId: string, body: Record<string, unknown>) => request<Record<string, unknown>>(`/workflow/documents/${documentId}/transitions`, { method: "POST", body: JSON.stringify(body) }),
  listAttachments: async (documentId: string) => {
    const response = await request<{ items: AttachmentItem[] }>(`/documents/${documentId}/attachments`);
    return { items: Array.isArray(response.items) ? response.items.map(normalizeAttachmentItem) : [] };
  },
  heartbeatDocumentPresence: (documentId: string, body?: Record<string, unknown>) => request<{ ok: boolean }>(`/documents/${documentId}/collaboration/presence`, { method: "POST", body: JSON.stringify(body ?? {}) }),
  listDocumentPresence: async (documentId: string) => {
    const response = await request<{ items: CollaborationPresenceItem[] }>(`/documents/${documentId}/collaboration/presence`);
    return { items: Array.isArray(response.items) ? response.items.map(normalizeCollaborationPresenceItem) : [] };
  },
  acquireDocumentEditLock: async (documentId: string, body?: Record<string, unknown>) =>
    normalizeDocumentEditLockItem(await request<DocumentEditLockItem>(`/documents/${documentId}/collaboration/lock`, { method: "POST", body: JSON.stringify(body ?? {}) })),
  getDocumentEditLock: async (documentId: string) =>
    normalizeDocumentEditLockItem(await request<DocumentEditLockItem>(`/documents/${documentId}/collaboration/lock`)),
  releaseDocumentEditLock: (documentId: string) => request<void>(`/documents/${documentId}/collaboration/lock`, { method: "DELETE" }),
  uploadAttachment: (documentId: string, file: File) => {
    const formData = new FormData();
    formData.append("file", file);
    return request<AttachmentItem>(`/documents/${documentId}/attachments`, { method: "POST", body: formData });
  },
  getAttachmentDownloadURL: (documentId: string, attachmentId: string) => request<{ attachmentId: string; downloadUrl: string; expiresAt: string }>(`/documents/${documentId}/attachments/${attachmentId}/download-url`),
  getDocumentContentNative: (documentId: string) => request<DocumentContentNativeResponse>(`/documents/${documentId}/content/native`),
  saveDocumentContentNative: (documentId: string, body: Record<string, unknown>) =>
    request<DocumentContentSaveResponse>(`/documents/${documentId}/content/native`, { method: "POST", body: JSON.stringify(body) }),
  getDocumentContentPdf: (documentId: string) => request<DocumentContentPdfResponse>(`/documents/${documentId}/content/pdf`),
  renderDocumentContentPdf: (documentId: string) =>
    request<DocumentContentPdfResponse>(`/documents/${documentId}/content/render-pdf`, { method: "POST" }),
  getDocumentContentDocx: (documentId: string) => request<DocumentContentDocxResponse>(`/documents/${documentId}/content/docx`),
  uploadDocumentContentDocx: (documentId: string, file: File) => {
    const formData = new FormData();
    formData.append("file", file);
    return request<DocumentContentUploadResponse>(`/documents/${documentId}/content/upload`, { method: "POST", body: formData });
  },
  downloadProfileTemplateDocx: (profileCode: string) =>
    requestBlob(`/document-profiles/${encodeURIComponent(profileCode)}/template/docx`),
  downloadDocumentTemplateDocx: (documentId: string) =>
    requestBlob(`/documents/${encodeURIComponent(documentId)}/template/docx`),
  listAccessPolicies: async (resourceScope: string, resourceId: string) => {
    const response = await request<{ items: AccessPolicyItem[] }>(`/access-policies?resourceScope=${encodeURIComponent(resourceScope)}&resourceId=${encodeURIComponent(resourceId)}`);
    return { items: Array.isArray(response.items) ? response.items.map(normalizeAccessPolicyItem) : [] };
  },
  replaceAccessPolicies: (body: Record<string, unknown>) => request<{ replacedCount: number }>("/access-policies", { method: "PUT", body: JSON.stringify(body) }),
  listNotifications: async (params?: URLSearchParams) => {
    const query = params?.toString();
    const response = await request<{ items: NotificationItem[] }>(`/notifications${query ? `?${query}` : ""}`);
    return { items: Array.isArray(response.items) ? response.items.map(normalizeNotificationItem) : [] };
  },
  listAuditEvents: async (params?: URLSearchParams) => {
    const query = params?.toString();
    const response = await request<{ items: AuditEventItem[] }>(`/audit/events${query ? `?${query}` : ""}`);
    return { items: Array.isArray(response.items) ? response.items.map(normalizeAuditEventItem) : [] };
  },
  markNotificationRead: (notificationId: string) => request<{ id: string; status: string; readAt: string }>(`/notifications/${encodeURIComponent(notificationId)}/read`, { method: "POST" }),
  subscribeOperationsStream: (
    onSnapshot: (snapshot: OperationsStreamSnapshot) => void,
    onError?: (error: Event) => void,
  ) => {
    const stream = new EventSource(`${API_BASE_URL}/operations/stream`, { withCredentials: true });
    const listener = (event: MessageEvent<string>) => {
      try {
        const payload = JSON.parse(event.data) as OperationsStreamSnapshot;
        onSnapshot({
          pendingNotifications: Number(payload?.pendingNotifications ?? 0),
          pendingApprovals: Number(payload?.pendingApprovals ?? 0),
          documentsInReview: Number(payload?.documentsInReview ?? 0),
          totalDocuments: Number(payload?.totalDocuments ?? 0),
          generatedAt: payload?.generatedAt ?? "",
        });
      } catch {
        // Ignore malformed payload and keep stream alive.
      }
    };

    stream.addEventListener("snapshot", listener as EventListener);
    stream.onerror = (error) => {
      if (onError) {
        onError(error);
      }
    };

    return () => {
      stream.removeEventListener("snapshot", listener as EventListener);
      stream.close();
    };
  },
};
