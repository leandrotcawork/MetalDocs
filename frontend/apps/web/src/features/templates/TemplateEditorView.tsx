import { useCallback, useEffect, useRef } from "react";
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

type TemplateEditorViewProps = {
  profileCode: string;
  templateKey: string;
};

export function TemplateEditorView({ profileCode, templateKey }: TemplateEditorViewProps) {
  // editorRef holds the BlockNote editor instance surfaced by onEditorReady
  const editorRef = useRef<{ document: unknown[] } | null>(null);

  const { draft, isLoading, error, saveDraft, publish, discardDraft } = useTemplateDraft({ templateKey });

  const {
    isDirty,
    markDirty,
    markClean,
    clearTemplate,
    validationErrors,
    setValidationErrors,
    selectedBlockId,
    setSelectedBlock,
    setDraft,
  } = useTemplatesStore();

  // Cleanup store when the view unmounts
  useEffect(() => {
    return () => {
      clearTemplate();
    };
  }, [clearTemplate]);

  const handleEditorReady = useCallback((editor: unknown) => {
    editorRef.current = editor as { document: unknown[] };
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
    setDraft(updatedDraft);
  }, [setDraft]);

  if (isLoading) {
    return (
      <div style={{ display: "flex", alignItems: "center", justifyContent: "center", height: "100vh", fontSize: "14px", color: "rgba(255,255,255,0.5)" }}>
        Carregando template...
      </div>
    );
  }

  if (error || !draft) {
    return (
      <div style={{ display: "flex", alignItems: "center", justifyContent: "center", height: "100vh", fontSize: "14px", color: "var(--color-error, #f87171)" }}>
        {error ?? "Template nao encontrado."}
      </div>
    );
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100vh", overflow: "hidden" }}>
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
      <div style={{ flex: 1, overflow: "hidden", display: "flex", position: "relative" }}>
        {/* Left: Block Palette */}
        <BlockPalette editor={editorRef.current} />

        {/* Center: MDDM Editor */}
        <div style={{ flex: 1, overflow: "hidden", position: "relative" }}>
          <MDDMEditor
            initialContent={Array.isArray(draft.blocks) ? (draft.blocks as PartialBlock[]) : undefined}
            onEditorReady={handleEditorReady}
            onChange={handleChange}
            onSelectionChange={setSelectedBlock}
          />
        </div>

        {/* Right: Property Sidebar */}
        <PropertySidebar
          editor={editorRef.current}
          selectedBlockId={selectedBlockId}
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
