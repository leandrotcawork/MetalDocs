import { useEffect, useRef } from "react";
import { CKEditor } from "@ckeditor/ckeditor5-react";
import { editorClass, getEditorConfig, type EditingTemplateMode } from "../lib/editorConfig";

type EditorCanvasProps = {
  initialData: string;
  mode: EditingTemplateMode;
  insertCommand?: {
    id: number;
    html: string;
  } | null;
  onInsertApplied?: () => void;
  onReady?: (editor: any) => void;
  onChange?: (html: string) => void;
  onDebouncedChange?: (html: string) => void;
  debounceMs?: number;
};

export function EditorCanvas({
  initialData,
  mode,
  insertCommand,
  onInsertApplied,
  onReady,
  onChange,
  onDebouncedChange,
  debounceMs = 300,
}: EditorCanvasProps) {
  const editorRef = useRef<any>(null);
  const toolbarHostRef = useRef<HTMLDivElement | null>(null);
  const latestHtmlRef = useRef(initialData);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastInsertIdRef = useRef<number | null>(null);

  useEffect(() => {
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
      if (toolbarHostRef.current) {
        toolbarHostRef.current.replaceChildren();
      }
      editorRef.current = null;
    };
  }, []);

  useEffect(() => {
    const editor = editorRef.current;
    if (!editor) {
      return;
    }
    if (initialData !== latestHtmlRef.current) {
      editor.setData(initialData);
      latestHtmlRef.current = initialData;
    }
  }, [initialData]);

  useEffect(() => {
    if (!insertCommand || lastInsertIdRef.current === insertCommand.id) {
      return;
    }
    const editor = editorRef.current;
    if (!editor) {
      return;
    }

    let cancelled = false;
    const currentHtml = editor.getData();
    const mergedHtml = `${currentHtml}${insertCommand.html}`;

    void Promise.resolve(editor.setData(mergedHtml)).then(() => {
      if (cancelled) {
        return;
      }
      const nextHtml = editor.getData();
      latestHtmlRef.current = nextHtml;
      lastInsertIdRef.current = insertCommand.id;
      onChange?.(nextHtml);
      onDebouncedChange?.(nextHtml);
      onInsertApplied?.();
    });

    return () => {
      cancelled = true;
    };
  }, [insertCommand, onChange, onDebouncedChange, onInsertApplied]);

  return (
    <div className="studio-editor">
      <div ref={toolbarHostRef} className="studio-editor-toolbar" data-testid="ck5-toolbar-host" />
      <div className="studio-editor-paper" data-testid="ck5-paper">
        <CKEditor
          key={mode}
          editor={editorClass as any}
          config={getEditorConfig(mode)}
          data={initialData}
          onReady={(editor) => {
            editorRef.current = editor;
            const toolbarElement = editor.ui.view.toolbar?.element;
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
