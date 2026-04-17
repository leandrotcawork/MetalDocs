import { useMemo, useRef } from 'react';
import { CKEditor } from '@ckeditor/ckeditor5-react';
import { DecoupledEditor } from 'ckeditor5';
import type { DecoupledEditorUIView } from 'ckeditor5';
import 'ckeditor5/ckeditor5.css';
import { createAuthorConfig } from '../config/editorConfig';
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

  return (
    <div className={styles.shell}>
      <div className={styles.toolbar} ref={toolbarRef} data-ck5-role="toolbar" />
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
