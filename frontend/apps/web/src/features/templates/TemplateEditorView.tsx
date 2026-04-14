import { useEffect } from "react";
import { useTemplatesStore } from "../../store/templates.store";

type TemplateEditorViewProps = {
  profileCode: string;
  templateKey: string;
};

// Placeholder — Phase 10 replaces this with the real editor
export function TemplateEditorView({ profileCode, templateKey }: TemplateEditorViewProps) {
  const clearTemplate = useTemplatesStore((s) => s.clearTemplate);

  useEffect(() => {
    return () => {
      clearTemplate();
    };
  }, [clearTemplate]);

  return (
    <div style={{ padding: 24 }}>
      <h2>Template editor (coming in Phase 10)</h2>
      <p>Profile: {profileCode}</p>
      <p>Template: {templateKey}</p>
    </div>
  );
}
