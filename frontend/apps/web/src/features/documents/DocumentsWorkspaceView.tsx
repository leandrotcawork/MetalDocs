import { DocumentsHubView } from "./DocumentsHubView";
import type {
  DocumentListItem,
  DocumentProfileGovernanceItem,
  DocumentProfileItem,
  ManagedUserItem,
  ProcessAreaItem,
  SearchDocumentItem,
} from "../../lib.types";

type DocumentsWorkspaceViewProps = {
  view: "library" | "my-docs" | "recent";
  loadState: "idle" | "loading" | "ready" | "error";
  documentProfiles: DocumentProfileItem[];
  processAreas: ProcessAreaItem[];
  documents: SearchDocumentItem[];
  managedUsers: ManagedUserItem[];
  selectedDocument: DocumentListItem | null;
  selectedProfileGovernance: DocumentProfileGovernanceItem | null;
  searchQuery: string;
  currentUserId?: string;
  formatDate: (value?: string) => string;
  onSearchQueryChange: (value: string) => void;
  onCreateDocument: () => void;
  onRefreshWorkspace: () => void | Promise<void>;
  onOpenDocument: (documentId: string, nextView?: "library" | "content-builder") => void | Promise<void>;
  onOpenDocumentForHub: (documentId: string) => void | Promise<void>;
};

export function DocumentsWorkspaceView(props: DocumentsWorkspaceViewProps) {
  return (
    <DocumentsHubView
      view={props.view}
      loadState={props.loadState}
      currentUserId={props.currentUserId}
      managedUsers={props.managedUsers}
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
      onRefreshDocuments={props.onRefreshWorkspace}
    />
  );
}
