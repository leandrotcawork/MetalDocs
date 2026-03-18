export type DocumentStatus = "DRAFT" | "IN_REVIEW" | "APPROVED" | "PUBLISHED" | "ARCHIVED";
export type Classification = "PUBLIC" | "INTERNAL" | "CONFIDENTIAL" | "RESTRICTED";
export type ResourceScope = "document" | "document_type" | "area";
export type UserRole = "admin" | "editor" | "reviewer" | "viewer";

export interface CurrentUser {
  userId: string;
  username: string;
  email?: string;
  displayName: string;
  mustChangePassword: boolean;
  roles: UserRole[];
}

export interface ManagedUserItem {
  userId: string;
  username: string;
  email?: string;
  displayName: string;
  isActive: boolean;
  mustChangePassword: boolean;
  failedLoginAttempts: number;
  lockedUntil?: string;
  lastLoginAt?: string;
  createdAt: string;
  updatedAt: string;
  roles: UserRole[];
}

export interface DocumentTypeItem {
  code: string;
  name: string;
  description: string;
  reviewIntervalDays: number;
}

export interface DocumentFamilyItem {
  code: string;
  name: string;
  description: string;
}

export interface DocumentProfileItem {
  code: string;
  familyCode: string;
  name: string;
  description: string;
  reviewIntervalDays: number;
  activeSchemaVersion: number;
  workflowProfile: string;
  approvalRequired: boolean;
  retentionDays: number;
  validityDays: number;
}

export interface ProcessAreaItem {
  code: string;
  name: string;
  description: string;
}

export interface SubjectItem {
  code: string;
  processAreaCode: string;
  name: string;
  description: string;
}

export interface MetadataFieldRuleItem {
  name: string;
  type: string;
  required: boolean;
}

export interface DocumentProfileSchemaItem {
  profileCode: string;
  version: number;
  isActive: boolean;
  metadataRules: MetadataFieldRuleItem[];
}

export interface DocumentProfileGovernanceItem {
  profileCode: string;
  workflowProfile: string;
  reviewIntervalDays: number;
  approvalRequired: boolean;
  retentionDays: number;
  validityDays: number;
}

export interface DocumentListItem {
  documentId: string;
  title: string;
  documentType: string;
  documentProfile: string;
  documentFamily: string;
  profileSchemaVersion?: number;
  processArea?: string;
  subject?: string;
  ownerId: string;
  businessUnit: string;
  department: string;
  classification: Classification;
  status: DocumentStatus;
  tags: string[];
  effectiveAt?: string;
  expiryAt?: string;
}

export interface SearchDocumentItem extends DocumentListItem {
  createdAt: string;
}

export interface VersionListItem {
  documentId: string;
  version: number;
  contentHash: string;
  changeSummary: string;
  createdAt: string;
}

export interface VersionDiffResponse {
  documentId: string;
  fromVersion: number;
  toVersion: number;
  contentChanged: boolean;
  metadataChanged: string[];
  classificationChanged: boolean;
  effectiveAtChanged: boolean;
  expiryAtChanged: boolean;
}

export interface WorkflowApprovalItem {
  approvalId: string;
  documentId: string;
  requestedBy: string;
  assignedReviewer: string;
  decisionBy?: string;
  status: "PENDING" | "APPROVED" | "REJECTED";
  requestReason?: string;
  decisionReason?: string;
  requestedAt: string;
  decidedAt?: string;
}

export interface AttachmentItem {
  attachmentId: string;
  documentId: string;
  fileName: string;
  contentType: string;
  sizeBytes: number;
  uploadedBy: string;
  createdAt: string;
}

export interface AccessPolicyItem {
  subjectType: "user" | "role" | "group";
  subjectId: string;
  resourceScope: ResourceScope;
  resourceId: string;
  capability: "document.create" | "document.view" | "document.edit" | "document.upload_attachment" | "document.change_workflow" | "document.manage_permissions";
  effect: "allow" | "deny";
}

export interface NotificationItem {
  id: string;
  recipientUserId: string;
  eventType: string;
  resourceType: string;
  resourceId: string;
  title: string;
  message: string;
  status: "PENDING" | "SENT" | "READ";
  createdAt: string;
  readAt?: string;
}

export interface AuditEventItem {
  id: string;
  occurredAt: string;
  actorId: string;
  action: string;
  resourceType: string;
  resourceId: string;
  payload: Record<string, unknown>;
  traceId: string;
}

export interface ApiErrorEnvelope {
  error: {
    code: string;
    message: string;
    details: Record<string, unknown>;
    trace_id: string;
  };
}
