export type DocumentStatus = "DRAFT" | "IN_REVIEW" | "APPROVED" | "PUBLISHED" | "ARCHIVED";
export type Classification = "PUBLIC" | "INTERNAL" | "CONFIDENTIAL" | "RESTRICTED";
export type ResourceScope = "document" | "document_type" | "area";

export interface DocumentTypeItem {
  code: string;
  name: string;
  description: string;
  reviewIntervalDays: number;
}

export interface DocumentListItem {
  documentId: string;
  title: string;
  documentType: string;
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

export interface ApiErrorEnvelope {
  error: {
    code: string;
    message: string;
    details: Record<string, unknown>;
    trace_id: string;
  };
}
