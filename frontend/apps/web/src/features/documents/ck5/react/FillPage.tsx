import { useState, useCallback, useEffect } from 'react';
import type { ClassicEditor } from 'ckeditor5';
import { FillEditor } from './FillEditor';
import { loadDocument, saveDocument, loadTemplate } from '../persistence/localStorageStub';
import { installFillHook, clearHooks } from './windowHooks';

export interface FillPageProps {
  tplId: string;
  docId: string;
}

export function FillPage({ tplId, docId }: FillPageProps) {
  const seed = loadDocument(docId) ?? loadTemplate(tplId)?.contentHtml ?? '<p>Empty</p>';
  const [html, setHtml] = useState<string>(seed);

  const onChange = useCallback(
    (next: string) => {
      setHtml(next);
      saveDocument(docId, next);
    },
    [docId],
  );

  const onReady = useCallback(
    (editor: ClassicEditor) => {
      installFillHook(editor, onChange);
    },
    [onChange],
  );

  useEffect(() => clearHooks, []);

  return (
    <div data-testid="ck5-fill-page" style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      <h1 style={{ padding: 12, margin: 0, borderBottom: '1px solid #ddd' }}>Fill - {docId}</h1>
      <div style={{ flex: 1, overflow: 'hidden' }}>
        <FillEditor documentHtml={html} onChange={onChange} onReady={onReady} />
      </div>
    </div>
  );
}
