import { RegistryExplorer } from "../../components/RegistryExplorer";
import type {
  DocumentProfileGovernanceItem,
  DocumentProfileItem,
  DocumentProfileSchemaItem,
  MetadataFieldRuleItem,
  ProcessAreaItem,
  SubjectItem,
} from "../../lib.types";

type RegistryExplorerViewProps = {
  loadState: "idle" | "loading" | "ready" | "error";
  documentProfiles: DocumentProfileItem[];
  processAreas: ProcessAreaItem[];
  subjects: SubjectItem[];
  selectedProfileCode: string;
  selectedProfileSchema: DocumentProfileSchemaItem | null;
  selectedProfileSchemas: DocumentProfileSchemaItem[];
  selectedProfileGovernance: DocumentProfileGovernanceItem | null;
  showAdmin: boolean;
  onRefreshWorkspace: () => void | Promise<void>;
  onSelectProfile: (profileCode: string) => void | Promise<void>;
  onCreateProcessArea: (payload: { code: string; name: string; description: string }) => void | Promise<void>;
  onUpdateProcessArea: (payload: { code: string; name: string; description: string }) => void | Promise<void>;
  onDeleteProcessArea: (code: string) => void | Promise<void>;
  onCreateSubject: (payload: { code: string; processAreaCode: string; name: string; description: string }) => void | Promise<void>;
  onUpdateSubject: (payload: { code: string; processAreaCode: string; name: string; description: string }) => void | Promise<void>;
  onDeleteSubject: (code: string) => void | Promise<void>;
  onCreateDocumentProfile: (payload: { code: string; familyCode: string; name: string; alias: string; description: string; reviewIntervalDays: number }) => void | Promise<void>;
  onUpdateDocumentProfile: (payload: { code: string; familyCode: string; name: string; alias: string; description: string; reviewIntervalDays: number }) => void | Promise<void>;
  onDeleteDocumentProfile: (code: string) => void | Promise<void>;
  onUpdateDocumentProfileGovernance: (payload: { profileCode: string; workflowProfile: string; reviewIntervalDays: number; approvalRequired: boolean; retentionDays: number; validityDays: number }) => void | Promise<void>;
  onUpsertDocumentProfileSchema: (payload: { profileCode: string; version: number; isActive: boolean; metadataRules: MetadataFieldRuleItem[] }) => void | Promise<void>;
  onActivateDocumentProfileSchema: (payload: { profileCode: string; version: number }) => void | Promise<void>;
};

export function RegistryExplorerView(props: RegistryExplorerViewProps) {
  return <RegistryExplorer {...props} />;
}
