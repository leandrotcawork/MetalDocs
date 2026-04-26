export interface DocumentProfile {
  code: string;
  tenantId: string;
  familyCode: string;
  name: string;
  description: string;
  reviewIntervalDays: number;
  defaultTemplateVersionId: string | null;
  ownerUserId: string | null;
  editableByRole: string;
  archivedAt: string | null;
  createdAt: string;
}

export interface ProcessArea {
  code: string;
  tenantId: string;
  name: string;
  description: string;
  parentCode: string | null;
  ownerUserId: string | null;
  defaultApproverRole: string | null;
  archivedAt: string | null;
  createdAt: string;
}

export interface CreateProfileRequest {
  code: string;
  familyCode: string;
  name: string;
  description?: string;
  reviewIntervalDays: number;
  editableByRole?: string;
}

export interface UpdateProfileRequest {
  familyCode: string;
  name?: string;
  description?: string;
  editableByRole?: string;
  reviewIntervalDays?: number;
}

export interface SetDefaultTemplateRequest {
  templateVersionId: string;
}

export interface CreateAreaRequest {
  code: string;
  name: string;
  description?: string;
  parentCode?: string;
  defaultApproverRole?: string;
}

export interface UpdateAreaRequest {
  name?: string;
  description?: string;
  parentCode?: string | null;
  defaultApproverRole?: string | null;
}
