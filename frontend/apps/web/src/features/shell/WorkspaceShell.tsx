import { DocumentWorkspaceShell } from "../../components/DocumentWorkspaceShell";
import type { WorkspaceView } from "../../components/DocumentWorkspaceShell";
import type { DocumentProfileItem, ProcessAreaItem, SearchDocumentItem } from "../../lib.types";

type WorkspaceShellProps = {
  userDisplayName: string;
  userRoleLabel: string;
  activeView: WorkspaceView;
  searchValue: string;
  notificationsPending: number;
  documentCount: number;
  reviewCount: number;
  registryCount: number;
  showAdmin: boolean;
  documentProfiles: DocumentProfileItem[];
  processAreas: ProcessAreaItem[];
  documents: SearchDocumentItem[];
  isRefreshing: boolean;
  flushContent?: boolean;
  editMode?: boolean;
  onSearchChange: (value: string) => void;
  onNavigate: (view: WorkspaceView) => void;
  onPrimaryAction: () => void;
  onRefreshWorkspace: () => void | Promise<void>;
  onLogout: () => void | Promise<void>;
  children: React.ReactNode;
};

export function WorkspaceShell(props: WorkspaceShellProps) {
  return (
    <DocumentWorkspaceShell
      userDisplayName={props.userDisplayName}
      userRoleLabel={props.userRoleLabel}
      organizationLabel="Metal Nobre"
      activeView={props.activeView}
      searchValue={props.searchValue}
      notificationsPending={props.notificationsPending}
      documentCount={props.documentCount}
      reviewCount={props.reviewCount}
      registryCount={props.registryCount}
      showAdmin={props.showAdmin}
      documentProfiles={props.documentProfiles}
      processAreas={props.processAreas}
      documents={props.documents}
      onSearchChange={props.onSearchChange}
      onNavigate={props.onNavigate}
      onPrimaryAction={props.onPrimaryAction}
      onRefreshWorkspace={props.onRefreshWorkspace}
      isRefreshing={props.isRefreshing}
      flushContent={props.flushContent}
      editMode={props.editMode}
      onLogout={props.onLogout}
    >
      {props.children}
    </DocumentWorkspaceShell>
  );
}
