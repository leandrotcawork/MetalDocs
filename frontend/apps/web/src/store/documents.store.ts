import { create } from "zustand";
import type {
  AccessPolicyItem,
  AttachmentItem,
  AuditEventItem,
  CollaborationPresenceItem,
  DocumentEditLockItem,
  DocumentListItem,
  SearchDocumentItem,
  VersionDiffResponse,
  VersionListItem,
  WorkflowApprovalItem,
} from "../lib.types";
import type { ContentMode } from "../components/create/documentCreateTypes";
import type { LoadState } from "./auth.store";

type ContentStatus = "idle" | "saving" | "ready" | "error";

type DocumentFormState = {
  title: string;
  documentType: string;
  documentProfile: string;
  processArea: string;
  subject: string;
  ownerId: string;
  businessUnit: string;
  department: string;
  classification: string;
  audienceMode: string;
  audienceDepartment: string;
  audienceDepartments: string[];
  audienceProcessArea: string;
  tags: string;
  effectiveAt: string;
  expiryAt: string;
  metadata: string;
  initialContent: string;
};

const emptyDocumentForm: DocumentFormState = {
  title: "",
  documentType: "po",
  documentProfile: "po",
  processArea: "",
  subject: "",
  ownerId: "",
  businessUnit: "Quality",
  department: "operacoes",
  classification: "INTERNAL",
  audienceMode: "DEPARTMENT",
  audienceDepartment: "operacoes",
  audienceDepartments: ["operacoes"],
  audienceProcessArea: "",
  tags: "",
  effectiveAt: "",
  expiryAt: "",
  metadata: "{}",
  initialContent: "",
};

interface DocumentsStore {
  loadState: LoadState;
  documents: SearchDocumentItem[];
  selectedDocument: DocumentListItem | null;
  versions: VersionListItem[];
  versionDiff: VersionDiffResponse | null;
  approvals: WorkflowApprovalItem[];
  attachments: AttachmentItem[];
  collaborationPresence: CollaborationPresenceItem[];
  documentEditLock: DocumentEditLockItem | null;
  policies: AccessPolicyItem[];
  auditEvents: AuditEventItem[];
  documentForm: DocumentFormState;
  contentMode: ContentMode;
  contentFile: File | null;
  contentPdfUrl: string;
  contentDocxUrl: string;
  contentStatus: ContentStatus;
  contentError: string;
  selectedFile: File | null;
  policyResourceId: string;
  setLoadState: (loadState: LoadState) => void;
  setDocuments: (documents: SearchDocumentItem[]) => void;
  setSelectedDocument: (selectedDocument: DocumentListItem | null) => void;
  setVersions: (versions: VersionListItem[]) => void;
  setVersionDiff: (versionDiff: VersionDiffResponse | null) => void;
  setApprovals: (approvals: WorkflowApprovalItem[]) => void;
  setAttachments: (attachments: AttachmentItem[]) => void;
  setCollaborationPresence: (collaborationPresence: CollaborationPresenceItem[]) => void;
  setDocumentEditLock: (documentEditLock: DocumentEditLockItem | null) => void;
  setPolicies: (policies: AccessPolicyItem[]) => void;
  setAuditEvents: (auditEvents: AuditEventItem[]) => void;
  setDocumentForm: (documentForm: DocumentFormState | ((current: DocumentFormState) => DocumentFormState)) => void;
  setContentMode: (contentMode: ContentMode) => void;
  setContentFile: (contentFile: File | null) => void;
  setContentPdfUrl: (contentPdfUrl: string) => void;
  setContentDocxUrl: (contentDocxUrl: string) => void;
  setContentStatus: (contentStatus: ContentStatus) => void;
  setContentError: (contentError: string) => void;
  setSelectedFile: (selectedFile: File | null) => void;
  setPolicyResourceId: (policyResourceId: string) => void;
}

export const useDocumentsStore = create<DocumentsStore>((set) => ({
  loadState: "idle",
  documents: [],
  selectedDocument: null,
  versions: [],
  versionDiff: null,
  approvals: [],
  attachments: [],
  collaborationPresence: [],
  documentEditLock: null,
  policies: [],
  auditEvents: [],
  documentForm: emptyDocumentForm,
  contentMode: "native",
  contentFile: null,
  contentPdfUrl: "",
  contentDocxUrl: "",
  contentStatus: "idle",
  contentError: "",
  selectedFile: null,
  policyResourceId: "",
  setLoadState: (loadState) => set({ loadState }),
  setDocuments: (documents) => set({ documents }),
  setSelectedDocument: (selectedDocument) => set({ selectedDocument }),
  setVersions: (versions) => set({ versions }),
  setVersionDiff: (versionDiff) => set({ versionDiff }),
  setApprovals: (approvals) => set({ approvals }),
  setAttachments: (attachments) => set({ attachments }),
  setCollaborationPresence: (collaborationPresence) => set({ collaborationPresence }),
  setDocumentEditLock: (documentEditLock) => set({ documentEditLock }),
  setPolicies: (policies) => set({ policies }),
  setAuditEvents: (auditEvents) => set({ auditEvents }),
  setDocumentForm: (documentForm) =>
    set((state) => ({
      documentForm: typeof documentForm === "function" ? documentForm(state.documentForm) : documentForm,
    })),
  setContentMode: (contentMode) => set({ contentMode }),
  setContentFile: (contentFile) => set({ contentFile }),
  setContentPdfUrl: (contentPdfUrl) => set({ contentPdfUrl }),
  setContentDocxUrl: (contentDocxUrl) => set({ contentDocxUrl }),
  setContentStatus: (contentStatus) => set({ contentStatus }),
  setContentError: (contentError) => set({ contentError }),
  setSelectedFile: (selectedFile) => set({ selectedFile }),
  setPolicyResourceId: (policyResourceId) => set({ policyResourceId }),
}));

export type { DocumentFormState };
