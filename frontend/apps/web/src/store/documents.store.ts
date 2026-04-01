import { create } from "zustand";
import type { SchemaDocumentEditorState } from "../features/documents/runtime/schemaRuntimeTypes";
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
type DocumentsHubView = "overview" | "collection" | "detail";
type DocumentsHubMode = "card" | "list";
type DocumentsHubStatus = "all" | "draft" | "review" | "approved";
type RecentDocumentItem = SearchDocumentItem & { openedAt: string };

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

export const emptyDocumentForm: DocumentFormState = {
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
  documentsHubView: DocumentsHubView;
  documentsHubMode: DocumentsHubMode;
  documentsHubStatus: DocumentsHubStatus;
  documentsHubArea: string;
  documentsHubProfile: string;
  recentDocuments: RecentDocumentItem[];
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
  selectedDocumentTypeKey: string;
  schemaDocumentEditor: SchemaDocumentEditorState;
  setLoadState: (loadState: LoadState) => void;
  setDocuments: (documents: SearchDocumentItem[]) => void;
  setSelectedDocument: (selectedDocument: DocumentListItem | null) => void;
  setDocumentsHubView: (documentsHubView: DocumentsHubView) => void;
  setDocumentsHubMode: (documentsHubMode: DocumentsHubMode) => void;
  setDocumentsHubStatus: (documentsHubStatus: DocumentsHubStatus) => void;
  setDocumentsHubArea: (documentsHubArea: string) => void;
  setDocumentsHubProfile: (documentsHubProfile: string) => void;
  setRecentDocuments: (recentDocuments: RecentDocumentItem[]) => void;
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
  setSelectedDocumentTypeKey: (selectedDocumentTypeKey: string) => void;
  setSchemaDocumentEditor: (
    schemaDocumentEditor:
      | SchemaDocumentEditorState
      | ((current: SchemaDocumentEditorState) => SchemaDocumentEditorState),
  ) => void;
}

export const useDocumentsStore = create<DocumentsStore>((set) => ({
  loadState: "idle",
  documents: [],
  selectedDocument: null,
  documentsHubView: "overview",
  documentsHubMode: "card",
  documentsHubStatus: "all",
  documentsHubArea: "all",
  documentsHubProfile: "all",
  recentDocuments: [],
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
  selectedDocumentTypeKey: "",
  schemaDocumentEditor: {
    documentId: "",
    typeKey: "",
    schema: null,
    values: {},
    version: null,
    pdfUrl: "",
    status: "idle",
    error: "",
    bundle: null,
    document: null,
  },
  setLoadState: (loadState) => set({ loadState }),
  setDocuments: (documents) => set({ documents }),
  setSelectedDocument: (selectedDocument) => set({ selectedDocument }),
  setDocumentsHubView: (documentsHubView) => set({ documentsHubView }),
  setDocumentsHubMode: (documentsHubMode) => set({ documentsHubMode }),
  setDocumentsHubStatus: (documentsHubStatus) => set({ documentsHubStatus }),
  setDocumentsHubArea: (documentsHubArea) => set({ documentsHubArea }),
  setDocumentsHubProfile: (documentsHubProfile) => set({ documentsHubProfile }),
  setRecentDocuments: (recentDocuments) => set({ recentDocuments }),
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
  setSelectedDocumentTypeKey: (selectedDocumentTypeKey) => set({ selectedDocumentTypeKey }),
  setSchemaDocumentEditor: (schemaDocumentEditor) =>
    set((state) => ({
      schemaDocumentEditor:
        typeof schemaDocumentEditor === "function" ? schemaDocumentEditor(state.schemaDocumentEditor) : schemaDocumentEditor,
    })),
}));

export type { DocumentFormState, DocumentsHubMode, DocumentsHubStatus, DocumentsHubView, RecentDocumentItem };
