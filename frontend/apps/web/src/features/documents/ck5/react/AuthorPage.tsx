import { useState, useCallback, useRef } from 'react';
import type { DecoupledEditor } from 'ckeditor5';
import { AuthorEditor } from './AuthorEditor';
import { saveTemplate, loadTemplate } from '../persistence/localStorageStub';
import { applyPerCellExceptions } from '../plugins/MddmDataTablePlugin';

export interface AuthorPageProps {
  tplId: string;
}

export function AuthorPage({ tplId }: AuthorPageProps) {
  const existing = loadTemplate(tplId);
  const [html, setHtml] = useState<string>(existing?.contentHtml ?? '<p>New template</p>');
  const editorRef = useRef<DecoupledEditor | null>(null);

  const onReady = useCallback((editor: DecoupledEditor) => {
    editorRef.current = editor;
  }, []);

  const onChange = useCallback(
    (next: string) => {
      const editor = editorRef.current;
      if (editor) applyPerCellExceptions(editor);
      const finalHtml = editor ? editor.getData() : next;
      setHtml(finalHtml);
      saveTemplate(tplId, finalHtml, existing?.manifest ?? { fields: [] });
    },
    [tplId, existing],
  );

  return (
    <div data-testid="ck5-author-page" style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      <h1 style={{ padding: 12, margin: 0, borderBottom: '1px solid #ddd' }}>Author - {tplId}</h1>
      <div style={{ flex: 1, overflow: 'hidden' }}>
        <AuthorEditor initialHtml={html} onChange={onChange} onReady={onReady} />
      </div>
    </div>
  );
}
