export interface ControlledDocument {
  id: string;
  tenantId: string;
  profileCode: string;
  processAreaCode: string;
  departmentCode: string | null;
  code: string;
  sequenceNum: number | null;
  title: string;
  ownerUserId: string;
  overrideTemplateVersionId: string | null;
  status: 'active' | 'obsolete' | 'superseded';
  createdAt: string;
  updatedAt: string;
}

export interface CreateControlledDocumentRequest {
  profileCode: string;
  processAreaCode: string;
  title: string;
  ownerUserId: string;
  departmentCode?: string;
  overrideTemplateVersionId?: string;
  overrideTemplateReason?: string;
  manualCode?: string;
  manualCodeReason?: string;
}
