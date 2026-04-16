import { useState, useCallback, useRef, useEffect } from 'react';
import type { DecoupledEditor } from 'ckeditor5';
import { AuthorEditor } from './AuthorEditor';
import { saveTemplate, loadTemplate } from '../persistence';
import { applyPerCellExceptions } from '../plugins/MddmDataTablePlugin';
import { installAuthorHook, clearHooks } from './windowHooks';

export interface AuthorPageProps {
  tplId: string;
}

export function AuthorPage({ tplId }: AuthorPageProps) {
  const [html, setHtml] = useState<string>('<p>New template</p>');
  const [manifest, setManifest] = useState<{ fields: Array<{ id: string; label?: string; type: string; required?: boolean }> }>({ fields: [] });
  const editorRef = useRef<DecoupledEditor | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const onReady = useCallback((editor: DecoupledEditor) => {
    editorRef.current = editor;
    applyPerCellExceptions(editor);
    installAuthorHook(editor, (html) => onChange(html ?? editor.getData()));
  }, []);

  const onChange = useCallback(
    (next: string) => {
      const editor = editorRef.current;
      if (debounceRef.current) clearTimeout(debounceRef.current);
      debounceRef.current = setTimeout(() => {
        if (editorRef.current) applyPerCellExceptions(editorRef.current);
      }, 500);
      const finalHtml = editor ? editor.getData() : next;
      setHtml(finalHtml);
      void saveTemplate(tplId, finalHtml, manifest);
    },
    [tplId, manifest],
  );

  useEffect(() => {
    Promise.resolve(loadTemplate(tplId))
      .then((rec) => {
        if (rec) {
          setHtml(rec.contentHtml);
          setManifest(rec.manifest);
        }
      })
      .catch((e) => {
        console.error('[AuthorPage] loadTemplate failed', e);
      });
  }, [tplId]);

  useEffect(() => clearHooks, []);

  return (
    <div data-testid="ck5-author-page" style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      <h1 style={{ padding: 12, margin: 0, borderBottom: '1px solid #ddd' }}>Author - {tplId}</h1>
      <div style={{ flex: 1, overflow: 'hidden' }}>
        <AuthorEditor initialHtml={html} onChange={onChange} onReady={onReady} />
      </div>
    </div>
  );
}
