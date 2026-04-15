import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { PartialBlock } from "@blocknote/core";
import { MDDMEditor } from "../documents/mddm-editor/MDDMEditor";
import { MetadataBar } from "./MetadataBar";
import { PropertySidebar } from "./PropertySidebar";
import { BlockPalette } from "./BlockPalette";
import { ValidationPanel } from "./ValidationPanel";
import { StrippedFieldsBanner } from "./StrippedFieldsBanner";
import { useTemplateDraft } from "./useTemplateDraft";
import { useTemplatesStore } from "../../store/templates.store";
import type { TemplateDraftDTO } from "../../api/templates";
import { readTemplatePageSettings, writeTemplatePageSettings, type TemplatePageSettings } from "./page-settings";
import styles from "./TemplateEditorView.module.css";

type TemplateEditorViewProps = {
  profileCode: string;
  templateKey: string;
};

export function TemplateEditorView({ profileCode, templateKey }: TemplateEditorViewProps) {
  // editorRef holds the BlockNote editor instance surfaced by onEditorReady
  const editorRef = useRef<{ document: unknown[] } | null>(null);
  const [editorInstance, setEditorInstance] = useState<{ document: unknown[] } | null>(null);

  const { draft, isLoading, error, saveDraft, publish, discardDraft, replaceDraft, updateDraftMeta } = useTemplateDraft({ templateKey });

  const {
    isDirty,
    markDirty,
    markClean,
    clearTemplate,
    validationErrors,
    setValidationErrors,
    selectedBlockId,
    setSelectedBlock,
  } = useTemplatesStore();

  // Cleanup store when the view unmounts
  useEffect(() => {
    return () => {
      clearTemplate();
    };
  }, [clearTemplate]);

  const handleEditorReady = useCallback((editor: unknown) => {
    const resolved = editor as { document: unknown[] };
    editorRef.current = resolved;
    setEditorInstance(resolved);
  }, []);

  const handleChange = useCallback((_blocks: unknown[]) => {
    // Called on every edit — mark the draft dirty so MetadataBar shows the indicator.
    // We read from editorRef.current.document on save/publish, not from this arg.
    markDirty();
  }, [markDirty]);

  const handleSave = useCallback(async () => {
    const blocks = editorRef.current?.document ?? [];
    await saveDraft(blocks);
    markClean();
  }, [saveDraft, markClean]);

  const handlePublish = useCallback(async () => {
    const blocks = editorRef.current?.document ?? [];
    await publish(blocks);
    // publish() navigates away on success; no markClean needed here
  }, [publish]);

  const handleNameChange = useCallback((_name: string) => {
    // Name editing is stored locally in MetadataBar for now.
    // Phase 11 will wire this to a PATCH /templates/:key/rename endpoint
    // once that endpoint is available. For now it's a display-only update.
  }, []);

  const handleValidationSelectBlock = useCallback((blockId: string) => {
    setSelectedBlock(blockId);
    // Scroll the editor to the selected block
    try {
      const editor = editorRef.current as any;
      if (editor?.setTextCursorPosition) {
        const block = editor.getBlock?.(blockId);
        if (block) {
          editor.setTextCursorPosition(block, "start");
        }
      }
    } catch {
      // Ignore scroll errors — selection is still set in the store
    }
  }, [setSelectedBlock]);

  const handleValidationDismiss = useCallback(() => {
    setValidationErrors([]);
  }, [setValidationErrors]);

  const handleStrippedAcknowledged = useCallback((updatedDraft: TemplateDraftDTO) => {
    replaceDraft(updatedDraft);
  }, [replaceDraft]);

  const pageSettings = useMemo(() => readTemplatePageSettings(draft?.meta), [draft?.meta]);

  const handlePageSettingsChange = useCallback((nextPageSettings: TemplatePageSettings) => {
    updateDraftMeta((currentMeta) => writeTemplatePageSettings(currentMeta, nextPageSettings));
  }, [updateDraftMeta]);

  if (isLoading) {
    return (
      <div className={styles.loadingState}>
        Carregando template...
      </div>
    );
  }

  if (error || !draft) {
    return (
      <div className={styles.errorState}>
        {error ?? "Template nao encontrado."}
      </div>
    );
  }

  return (
    <div className={styles.layout} data-testid="template-editor-layout">
      {/* Top bar */}
      <MetadataBar
        templateKey={templateKey}
        templateName={draft.name}
        profileCode={profileCode}
        status={draft.status}
        lockVersion={draft.lockVersion}
        hasStrippedFields={draft.hasStrippedFields}
        isDirty={isDirty}
        onSave={() => void handleSave()}
        onPublish={() => void handlePublish()}
        onDiscard={() => void discardDraft()}
        onNameChange={handleNameChange}
      />

      {/* Stripped-fields banner (below MetadataBar, above editor) */}
      {draft.hasStrippedFields && (
        <StrippedFieldsBanner
          templateKey={templateKey}
          lockVersion={draft.lockVersion}
          onAcknowledged={handleStrippedAcknowledged}
        />
      )}

      {/* Editor + Sidebar row — fills remaining height, position:relative for ValidationPanel */}
      <div className={styles.workspaceRow} data-testid="template-editor-sidebars">
        {/* Left: Block Palette */}
        <BlockPalette editor={editorInstance} />

        {/* Center: MDDM Editor */}
        <div className={styles.documentPane} data-testid="template-editor-document-pane">
          <MDDMEditor
            initialContent={Array.isArray(draft.blocks) ? (draft.blocks as PartialBlock[]) : undefined}
            pageSettings={pageSettings}
            onEditorReady={handleEditorReady}
            onChange={handleChange}
            onSelectionChange={setSelectedBlock}
          />
        </div>

        {/* Right: Property Sidebar */}
        <PropertySidebar
          editor={editorInstance}
          selectedBlockId={selectedBlockId}
          pageSettings={pageSettings}
          onPageSettingsChange={handlePageSettingsChange}
        />

        {/* Bottom: Validation panel (slides up from bottom, absolute within the row) */}
        {validationErrors.length > 0 && (
          <ValidationPanel
            errors={validationErrors}
            onSelectBlock={handleValidationSelectBlock}
            onDismiss={handleValidationDismiss}
          />
        )}
      </div>
    </div>
  );
}
