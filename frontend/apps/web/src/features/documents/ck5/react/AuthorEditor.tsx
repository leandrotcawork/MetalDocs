import { useEffect, useMemo, useRef, useState } from 'react';
import { CKEditor } from '@ckeditor/ckeditor5-react';
import { DecoupledEditor } from 'ckeditor5';
import type { ClassicEditor } from 'ckeditor5';
import type { DecoupledEditorUIView } from 'ckeditor5';
import 'ckeditor5/ckeditor5.css';
import { createAuthorConfig } from '../config/editorConfig';
import { PageCounter } from './PageCounter';
import { PageFooters } from './PageFooters';
import { usePredictivePageCount } from './usePredictivePageCount';
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
  const paperWrapperRef = useRef<HTMLDivElement | null>(null);
  const [editorRoot, setEditorRoot] = useState<HTMLElement | null>(null);
  const [portalTarget, setPortalTarget] = useState<HTMLElement | null>(null);
  const pages = usePredictivePageCount(editorRoot);

  useEffect(() => {
    if (!editor) return;
    const root = editor.editing?.view?.getDomRoot?.() as HTMLElement | null | undefined;
    setEditorRoot(root ?? null);
    setPortalTarget(paperWrapperRef.current);
  }, [editor]);
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
        <div ref={paperWrapperRef} className={styles.paperWrapper}>
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
          <PageFooters pages={pages} portalTarget={portalTarget} />
        </div>
      </div>
    </div>
  );
}
