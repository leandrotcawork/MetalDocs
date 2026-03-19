import type { DocumentDepartmentItem, DocumentProfileGovernanceItem, DocumentProfileItem, DocumentProfileSchemaItem, ProcessAreaItem, SubjectItem } from "../../lib.types";

export type DocumentForm = {
  title: string;
  documentType: string;
  documentProfile: string;
  processArea: string;
  subject: string;
  ownerId: string;
  businessUnit: string;
  department: string;
  classification: string;
  tags: string;
  effectiveAt: string;
  expiryAt: string;
  metadata: string;
  initialContent: string;
};

export type WizardStep = "identification" | "context" | "metadata" | "content";
export type StepStatus = "pending" | "active" | "done" | "error";

export type DocumentCreateViewProps = {
  documentForm: DocumentForm;
  documentProfiles: DocumentProfileItem[];
  processAreas: ProcessAreaItem[];
  documentDepartments: DocumentDepartmentItem[];
  subjects: SubjectItem[];
  selectedProfileSchema: DocumentProfileSchemaItem | null;
  selectedProfileGovernance: DocumentProfileGovernanceItem | null;
  onDocumentFormChange: (next: DocumentForm) => void;
  onApplyProfile: (profileCode: string, preferredProcessArea?: string) => void | Promise<void>;
  onSubmitCreateDocument: (event: React.FormEvent<HTMLFormElement>) => void | Promise<void>;
};

export const wizardSteps: Array<{ key: WizardStep; label: string; description: string }> = [
  { key: "identification", label: "Identificacao", description: "Titulo e profile documental." },
  { key: "context", label: "Contexto operacional", description: "Responsavel, area e taxonomia." },
  { key: "metadata", label: "Campos dinamicos", description: "Campos do schema." },
  { key: "content", label: "Conteudo e acesso", description: "Classificacao e vigencia." },
];

export function parseMetadata(value: string): Record<string, string> {
  try {
    const parsed = JSON.parse(value);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return {};
    }
    return Object.entries(parsed).reduce<Record<string, string>>((acc, [key, item]) => {
      const trimmedKey = key.trim();
      if (!trimmedKey) {
        return acc;
      }
      acc[trimmedKey] = typeof item === "string" ? item : JSON.stringify(item);
      return acc;
    }, {});
  } catch {
    return {};
  }
}

export function updateMetadataField(source: string, key: string, nextValue: string): string {
  const metadata = parseMetadata(source);
  metadata[key] = nextValue;
  return JSON.stringify(metadata, null, 2);
}

export function deleteMetadataField(source: string, key: string): string {
  const metadata = parseMetadata(source);
  delete metadata[key];
  return JSON.stringify(metadata, null, 2);
}

export function renameMetadataField(source: string, fromKey: string, toKey: string, nextValue?: string): string {
  const metadata = parseMetadata(source);
  const trimmedToKey = toKey.trim();
  const value = nextValue ?? metadata[fromKey] ?? "";
  delete metadata[fromKey];
  if (trimmedToKey) {
    metadata[trimmedToKey] = value;
  }
  return JSON.stringify(metadata, null, 2);
}
