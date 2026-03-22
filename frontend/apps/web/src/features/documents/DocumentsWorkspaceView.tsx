import { DocumentsHubView } from "./DocumentsHubView";
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
  currentUserId?: string;
  formatDate: (value?: string) => string;
  onSearchQueryChange: (value: string) => void;
  onCreateDocument: () => void;
  onRefreshWorkspace: () => void | Promise<void>;
  onOpenDocument: (documentId: string, nextView?: "library" | "content-builder") => void | Promise<void>;
  onOpenDocumentForHub: (documentId: string) => void | Promise<void>;
  onFileChange: (file: File | null) => void;
  onUploadAttachment: (event: React.FormEvent<HTMLFormElement>) => void | Promise<void>;
};

export function DocumentsWorkspaceView(props: DocumentsWorkspaceViewProps) {
  return (
    <DocumentsHubView
      view={props.view}
      loadState={props.loadState}
      currentUserId={props.currentUserId}
      documents={props.documents}
      documentProfiles={props.documentProfiles}
      processAreas={props.processAreas}
      selectedDocument={props.selectedDocument}
      selectedProfileGovernance={props.selectedProfileGovernance}
      searchQuery={props.searchQuery}
      formatDate={props.formatDate}
      onSearchQueryChange={props.onSearchQueryChange}
      onCreateDocument={props.onCreateDocument}
      onOpenDocument={props.onOpenDocument}
      onOpenDocumentForHub={props.onOpenDocumentForHub}
    />
  );
}
