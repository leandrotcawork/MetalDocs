import { DocumentsWorkspace } from "../../components/DocumentsWorkspace";
import type {
  AccessPolicyItem,
  AttachmentItem,
  AuditEventItem,
  CollaborationPresenceItem,
  DocumentEditLockItem,
  DocumentListItem,
  DocumentProfileGovernanceItem,
  DocumentProfileItem,
  ProcessAreaItem,
  SearchDocumentItem,
  VersionDiffResponse,
  VersionListItem,
  WorkflowApprovalItem,
} from "../../lib.types";

type DocumentsWorkspaceViewProps = {
  view: "library" | "my-docs" | "recent";
  loadState: "idle" | "loading" | "ready" | "error";
  documentProfiles: DocumentProfileItem[];
  processAreas: ProcessAreaItem[];
  documents: SearchDocumentItem[];
  selectedDocument: DocumentListItem | null;
  selectedProfileGovernance: DocumentProfileGovernanceItem | null;
  versions: VersionListItem[];
  versionDiff: VersionDiffResponse | null;
  approvals: WorkflowApprovalItem[];
  attachments: AttachmentItem[];
  collaborationPresence: CollaborationPresenceItem[];
  documentEditLock: DocumentEditLockItem | null;
  policies: AccessPolicyItem[];
  auditEvents: AuditEventItem[];
  selectedFile: File | null;
  policyScope: "document" | "document_type" | "area";
  policyResourceId: string;
  searchQuery: string;
  formatDate: (value?: string) => string;
  onRefreshWorkspace: () => void | Promise<void>;
  onOpenDocument: (documentId: string) => void | Promise<void>;
  onFileChange: (file: File | null) => void;
  onUploadAttachment: (event: React.FormEvent<HTMLFormElement>) => void | Promise<void>;
};

export function DocumentsWorkspaceView(props: DocumentsWorkspaceViewProps) {
  return <DocumentsWorkspace {...props} />;
}

