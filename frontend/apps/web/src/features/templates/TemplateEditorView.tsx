import { useCallback, useEffect, useRef } from "react";
import type { PartialBlock } from "@blocknote/core";
import { MDDMEditor } from "../documents/mddm-editor/MDDMEditor";
import { MetadataBar } from "./MetadataBar";
import { useTemplateDraft } from "./useTemplateDraft";
import { useTemplatesStore } from "../../store/templates.store";

type TemplateEditorViewProps = {
  profileCode: string;
  templateKey: string;
};

export function TemplateEditorView({ profileCode, templateKey }: TemplateEditorViewProps) {
  // editorRef holds the BlockNote editor instance surfaced by onEditorReady
  const editorRef = useRef<{ document: unknown[] } | null>(null);

  const { draft, isLoading, error, saveDraft, publish, discardDraft } = useTemplateDraft({ templateKey });

  const { isDirty, markDirty, markClean, clearTemplate, validationErrors } = useTemplatesStore();

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
      <MetadataBar
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

      {/* Validation errors panel (shown below MetadataBar when publish fails) */}
      {validationErrors.length > 0 && (
        <div
          data-testid="validation-errors-panel"
          style={{
            padding: "0.5rem 1rem",
            background: "rgba(239,68,68,0.1)",
            borderBottom: "1px solid rgba(239,68,68,0.3)",
            fontSize: "12px",
            color: "#fca5a5",
            flexShrink: 0,
          }}
        >
          <strong>Erros de validacao ({validationErrors.length}):</strong>
          <ul style={{ margin: "4px 0 0 0", paddingLeft: "1.25rem" }}>
            {validationErrors.slice(0, 5).map((e, i) => (
              <li key={i}>
                [{e.blockType}] {e.field}: {e.reason}
              </li>
            ))}
            {validationErrors.length > 5 && (
              <li>...e mais {validationErrors.length - 5} erro(s)</li>
            )}
          </ul>
        </div>
      )}

      {/* Editor surface — fills remaining height */}
      <div style={{ flex: 1, overflow: "hidden", position: "relative" }}>
        <MDDMEditor
          initialContent={Array.isArray(draft.blocks) ? (draft.blocks as PartialBlock[]) : undefined}
          onEditorReady={handleEditorReady}
          onChange={handleChange}
        />
      </div>
    </div>
  );
}
