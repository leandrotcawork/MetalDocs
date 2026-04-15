import { useEffect, useRef } from "react";
import { CKEditor } from "@ckeditor/ckeditor5-react";
import { editorClass, editorConfig } from "../lib/editorConfig";

type EditorCanvasProps = {
  initialData: string;
  onReady?: (editor: any) => void;
  onChange?: (html: string) => void;
  onDebouncedChange?: (html: string) => void;
  debounceMs?: number;
};

export function EditorCanvas({
  initialData,
  onReady,
  onChange,
  onDebouncedChange,
  debounceMs = 300,
}: EditorCanvasProps) {
  const toolbarHostRef = useRef<HTMLDivElement | null>(null);
  const latestHtmlRef = useRef(initialData);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
      if (toolbarHostRef.current) {
        toolbarHostRef.current.replaceChildren();
      }
    };
  }, []);

  return (
    <div className="studio-editor">
      <div ref={toolbarHostRef} className="studio-editor-toolbar" data-testid="ck5-toolbar-host" />
      <div className="studio-editor-paper" data-testid="ck5-paper">
        <CKEditor
          editor={editorClass as any}
          config={editorConfig}
          data={initialData}
          onReady={(editor) => {
            const toolbarElement = editor.ui.view.toolbar.element;
            if (toolbarElement && toolbarHostRef.current) {
              toolbarHostRef.current.replaceChildren(toolbarElement);
            }
            onReady?.(editor);
          }}
          onChange={(_, editor) => {
            const nextHtml = editor.getData();
            latestHtmlRef.current = nextHtml;
            onChange?.(nextHtml);

            if (onDebouncedChange) {
              if (debounceTimerRef.current) {
                clearTimeout(debounceTimerRef.current);
              }
              debounceTimerRef.current = setTimeout(() => {
                onDebouncedChange(latestHtmlRef.current);
              }, debounceMs);
            }
          }}
        />
      </div>
    </div>
  );
}
