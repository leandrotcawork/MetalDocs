import { useState, useCallback, useRef, useEffect } from 'react';
import type { DecoupledEditor } from 'ckeditor5';
import { AuthorEditor } from './AuthorEditor';
import { saveTemplate, loadTemplate } from '../persistence';
import { applyPerCellExceptions } from '../plugins/MddmDataTablePlugin';
import { installAuthorHook, clearHooks } from './windowHooks';
import { PublishButton } from './components/PublishButton';
import type { TemplateDraftStatus } from '../persistence/templatePublishApi';

function isTemplateDraftStatus(value: unknown): value is TemplateDraftStatus {
  return value === 'draft' || value === 'pending_review' || value === 'published';
}

export interface AuthorPageProps {
  tplId: string;
}

export function AuthorPage({ tplId }: AuthorPageProps) {
  const [html, setHtml] = useState<string | null>(null);
  const [manifest, setManifest] = useState<{ fields: Array<{ id: string; label?: string; type: string; required?: boolean }> }>({ fields: [] });
  const [draftStatus, setDraftStatus] = useState<TemplateDraftStatus>('draft');
  const editorRef = useRef<DecoupledEditor | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const manifestRef = useRef(manifest);
  manifestRef.current = manifest;
  const templateKey = tplId;

  const onChange = useCallback(
    (next: string) => {
      const editor = editorRef.current;
      if (debounceRef.current) clearTimeout(debounceRef.current);
      debounceRef.current = setTimeout(() => {
        if (editorRef.current) applyPerCellExceptions(editorRef.current);
      }, 500);
      const finalHtml = editor ? editor.getData() : next;
      void saveTemplate(tplId, finalHtml, manifestRef.current);
    },
    [tplId],
  );

  const onReady = useCallback(
    (editor: DecoupledEditor) => {
      editorRef.current = editor;
      applyPerCellExceptions(editor);
      installAuthorHook(editor, (h) => onChange(h ?? editor.getData()));
    },
    [onChange],
  );

  useEffect(() => {
    let cancelled = false;
    Promise.resolve(loadTemplate(tplId))
      .then((rec) => {
        if (cancelled) return;
        if (rec) {
          setHtml(rec.contentHtml && rec.contentHtml.length > 0 ? rec.contentHtml : '<p></p>');
          setManifest(rec.manifest);
          if (isTemplateDraftStatus(rec.draft_status)) {
            setDraftStatus(rec.draft_status);
          }
        } else {
          setHtml('<p></p>');
        }
      })
      .catch((e) => {
        console.error('[AuthorPage] loadTemplate failed', e);
        if (!cancelled) setHtml('<p></p>');
      });
    return () => {
      cancelled = true;
    };
  }, [tplId]);

  useEffect(() => clearHooks, []);

  return (
    <div data-testid="ck5-author-page" style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      <div
        style={{
          padding: 12,
          margin: 0,
          borderBottom: '1px solid #ddd',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: 12,
        }}
      >
        <h1 style={{ margin: 0 }}>Author - {tplId}</h1>
        <PublishButton templateKey={templateKey} draftStatus={draftStatus} onStatusChange={setDraftStatus} />
      </div>
      <div style={{ flex: 1, overflow: 'hidden' }}>
        {html === null ? (
          <div style={{ padding: 24 }}>Carregando template…</div>
        ) : (
          <AuthorEditor initialHtml={html} onChange={onChange} onReady={onReady} />
        )}
      </div>
    </div>
  );
}
