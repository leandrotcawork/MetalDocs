import { useMemo, useRef, useState } from 'react';
import { CKEditor } from '@ckeditor/ckeditor5-react';
import { DecoupledEditor } from 'ckeditor5';
import type { ClassicEditor } from 'ckeditor5';
import type { DecoupledEditorUIView } from 'ckeditor5';
import 'ckeditor5/ckeditor5.css';
import { createAuthorConfig } from '../config/editorConfig';
import { PageCounter } from './PageCounter';
import { PaginationDebugOverlay } from './PaginationDebugOverlay';
import styles from './AuthorEditor.module.css';

export interface AuthorEditorProps {
  initialHtml: string;
  onChange: (html: string) => void;
  onReady?: (editor: DecoupledEditor) => void;
  language?: string;
}

export function AuthorEditor({ initialHtml, onChange, onReady, language = 'en' }: AuthorEditorProps) {
  const toolbarRef = useRef<HTMLDivElement>(null);
  // Memoize config — the CKEditor React wrapper re-runs its mount-sync cycle
  // when config object identity changes, which can double-fire editor.data.set
  // and trip CKEditor's "unexpected-error" in DataController.set.
  const config = useMemo(() => createAuthorConfig({ language }), [language]);
  const [editor, setEditor] = useState<ClassicEditor | null>(null);
  const debugFlag = typeof window !== 'undefined' && new URLSearchParams(window.location.search).get('debug') === 'pagination';

  return (
    <div className={styles.shell}>
      <div className={styles.toolbar} data-ck5-role="toolbar">
        <div ref={toolbarRef} />
        <PageCounter editor={editor} />
        <PaginationDebugOverlay
          logs={{ exactMatches: 0, minorDrift: 0, majorDrift: 0, orphanedEditor: 0, serverOnly: 0 }}
          debugFlag={debugFlag}
        />
      </div>
      <div className={styles.editable} data-ck5-role="editable">
        <CKEditor
          editor={DecoupledEditor}
          data={initialHtml}
          config={config}
          onReady={(editor) => {
            const view = editor.ui.view as DecoupledEditorUIView;
            if (toolbarRef.current && view.toolbar.element) {
              toolbarRef.current.replaceChildren(view.toolbar.element);
            }
            setEditor(editor as unknown as ClassicEditor);
            onReady?.(editor);
          }}
          onChange={(_event, editor) => {
            onChange(editor.getData());
          }}
        />
      </div>
    </div>
  );
}
