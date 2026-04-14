import { create } from "zustand";
import type { TemplateDraftDTO, PublishErrorDTO } from "../api/templates";

interface TemplatesState {
  activeTemplateKey: string | null;
  draft: TemplateDraftDTO | null;
  lockVersion: number;
  selectedBlockId: string | null;
  validationErrors: PublishErrorDTO[];
  hasStrippedFields: boolean;
  isDirty: boolean;
  permissions: {
    canEdit: boolean;
    canPublish: boolean;
    canExport: boolean;
  };

  // Actions
  setActiveTemplate: (key: string | null) => void;
  setDraft: (draft: TemplateDraftDTO | null) => void;
  setSelectedBlock: (blockId: string | null) => void;
  setValidationErrors: (errors: PublishErrorDTO[]) => void;
  markDirty: () => void;
  markClean: () => void;
  clearTemplate: () => void;
}

export const useTemplatesStore = create<TemplatesState>((set) => ({
  activeTemplateKey: null,
  draft: null,
  lockVersion: 0,
  selectedBlockId: null,
  validationErrors: [],
  hasStrippedFields: false,
  isDirty: false,
  permissions: { canEdit: true, canPublish: true, canExport: true },

  setActiveTemplate: (key) => set({ activeTemplateKey: key }),
  setDraft: (draft) =>
    set({
      draft,
      lockVersion: draft?.lockVersion ?? 0,
      hasStrippedFields: draft?.hasStrippedFields ?? false,
    }),
  setSelectedBlock: (blockId) => set({ selectedBlockId: blockId }),
  setValidationErrors: (errors) => set({ validationErrors: errors }),
  markDirty: () => set({ isDirty: true }),
  markClean: () => set({ isDirty: false }),
  clearTemplate: () =>
    set({
      activeTemplateKey: null,
      draft: null,
      lockVersion: 0,
      selectedBlockId: null,
      validationErrors: [],
      hasStrippedFields: false,
      isDirty: false,
    }),
}));
