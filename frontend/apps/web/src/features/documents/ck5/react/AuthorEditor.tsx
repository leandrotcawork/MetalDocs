import { useRef } from 'react';
import { CKEditor } from '@ckeditor/ckeditor5-react';
import { DecoupledEditor } from 'ckeditor5';
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

  return (
    <div className={styles.shell}>
      <div className={styles.toolbar} ref={toolbarRef} data-ck5-role="toolbar" />
      <div className={styles.editable} data-ck5-role="editable">
        <CKEditor
          editor={DecoupledEditor}
          data={initialHtml}
          config={createAuthorConfig({ language })}
          onReady={(editor) => {
            // Move the detached toolbar into our toolbar container.
            const toolbarEl = (editor.ui.view as unknown as { toolbar: { element: HTMLElement } }).toolbar.element;
            if (toolbarRef.current && toolbarEl) {
              toolbarRef.current.appendChild(toolbarEl);
            }
            // Notify parent of the initial data so callers always have current state.
            onChange(editor.getData());
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
