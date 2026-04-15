import { CKEditor } from '@ckeditor/ckeditor5-react';
import { ClassicEditor } from 'ckeditor5';
import 'ckeditor5/ckeditor5.css';
import { createFillConfig } from '../config/editorConfig';
import styles from './FillEditor.module.css';

export interface FillEditorProps {
  documentHtml: string;
  onChange: (html: string) => void;
  onReady?: (editor: ClassicEditor) => void;
  language?: string;
}

export function FillEditor({ documentHtml, onChange, onReady, language = 'en' }: FillEditorProps) {
  return (
    <div className={styles.shell}>
      <div className={styles.editable}>
        <CKEditor
          editor={ClassicEditor}
          data={documentHtml}
          config={createFillConfig({ language })}
          onReady={(editor) => {
            // Land the caret on the first restricted-editing exception so the
            // user can start typing immediately.
            try {
              editor.execute('goToNextRestrictedEditingException');
            } catch {
              // No exceptions in this document, or command unavailable mid-init.
              // Safe to ignore; happens when the template has no fillable regions.
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
