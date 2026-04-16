import { useState, useCallback, useEffect } from 'react';
import type { ClassicEditor } from 'ckeditor5';
import { FillEditor } from './FillEditor';
import { loadDocument, saveDocument, loadTemplate } from '../persistence';
import { installFillHook, clearHooks } from './windowHooks';

export interface FillPageProps {
  tplId: string;
  docId: string;
}

export function FillPage({ tplId, docId }: FillPageProps) {
  const [html, setHtml] = useState<string>('<p>Empty</p>');

  const onChange = useCallback(
    (next: string) => {
      setHtml(next);
      void saveDocument(docId, next);
    },
    [docId],
  );

  const onReady = useCallback(
    (editor: ClassicEditor) => {
      installFillHook(editor, (html) => onChange(html ?? editor.getData()));
    },
    [onChange],
  );

  useEffect(() => clearHooks, []);

  useEffect(() => {
    let cancelled = false;

    async function loadInitialContent(): Promise<void> {
      const docHtml = await Promise.resolve(loadDocument(docId));
      if (cancelled) return;
      if (docHtml) {
        setHtml(docHtml);
        return;
      }

      const template = await Promise.resolve(loadTemplate(tplId));
      if (cancelled) return;
      if (template?.contentHtml) setHtml(template.contentHtml);
    }

    void loadInitialContent().catch((e) => {
      console.error('[FillPage] initial load failed', e);
    });

    return () => {
      cancelled = true;
    };
  }, [docId, tplId]);

  return (
    <div data-testid="ck5-fill-page" style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      <h1 style={{ padding: 12, margin: 0, borderBottom: '1px solid #ddd' }}>Fill - {docId}</h1>
      <div style={{ flex: 1, overflow: 'hidden' }}>
        <FillEditor documentHtml={html} onChange={onChange} onReady={onReady} />
      </div>
    </div>
  );
}
