import { useEffect, useRef } from "react";
import { CKEditor } from "@ckeditor/ckeditor5-react";
import { editorClass, editorConfig } from "../lib/editorConfig";

type EditorCanvasProps = {
  initialData: string;
  onReady?: (editor: any) => void;
  onChange?: (html: string) => void;
};

export function EditorCanvas({ initialData, onReady, onChange }: EditorCanvasProps) {
  const toolbarHostRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    return () => {
      if (toolbarHostRef.current) {
        toolbarHostRef.current.innerHTML = "";
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
            if (toolbarElement && toolbarHostRef.current && !toolbarHostRef.current.contains(toolbarElement)) {
              toolbarHostRef.current.appendChild(toolbarElement);
            }
            onReady?.(editor);
          }}
          onChange={(_, editor) => {
            onChange?.(editor.getData());
          }}
        />
      </div>
    </div>
  );
}
