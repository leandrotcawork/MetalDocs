import { create } from "zustand";
import type {
  DocumentDepartmentItem,
  DocumentProfileGovernanceItem,
  DocumentProfileItem,
  DocumentProfileSchemaItem,
  ProcessAreaItem,
  SubjectItem,
} from "../lib.types";

interface RegistryStore {
  documentProfiles: DocumentProfileItem[];
  processAreas: ProcessAreaItem[];
  documentDepartments: DocumentDepartmentItem[];
  subjects: SubjectItem[];
  selectedProfileSchema: DocumentProfileSchemaItem | null;
  selectedProfileSchemas: DocumentProfileSchemaItem[];
  selectedProfileGovernance: DocumentProfileGovernanceItem | null;
  setDocumentProfiles: (
    documentProfiles: DocumentProfileItem[] | ((current: DocumentProfileItem[]) => DocumentProfileItem[]),
  ) => void;
  setProcessAreas: (processAreas: ProcessAreaItem[]) => void;
  setDocumentDepartments: (documentDepartments: DocumentDepartmentItem[]) => void;
  setSubjects: (subjects: SubjectItem[]) => void;
  setSelectedProfileSchema: (selectedProfileSchema: DocumentProfileSchemaItem | null) => void;
  setSelectedProfileSchemas: (selectedProfileSchemas: DocumentProfileSchemaItem[]) => void;
  setSelectedProfileGovernance: (selectedProfileGovernance: DocumentProfileGovernanceItem | null) => void;
}

export const useRegistryStore = create<RegistryStore>((set) => ({
  documentProfiles: [],
  processAreas: [],
  documentDepartments: [],
  subjects: [],
  selectedProfileSchema: null,
  selectedProfileSchemas: [],
  selectedProfileGovernance: null,
  setDocumentProfiles: (documentProfiles) =>
    set((state) => ({
      documentProfiles: typeof documentProfiles === "function" ? documentProfiles(state.documentProfiles) : documentProfiles,
    })),
  setProcessAreas: (processAreas) => set({ processAreas }),
  setDocumentDepartments: (documentDepartments) => set({ documentDepartments }),
  setSubjects: (subjects) => set({ subjects }),
  setSelectedProfileSchema: (selectedProfileSchema) => set({ selectedProfileSchema }),
  setSelectedProfileSchemas: (selectedProfileSchemas) => set({ selectedProfileSchemas }),
  setSelectedProfileGovernance: (selectedProfileGovernance) => set({ selectedProfileGovernance }),
}));
