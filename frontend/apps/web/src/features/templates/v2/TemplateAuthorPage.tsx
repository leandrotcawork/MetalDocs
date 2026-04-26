import '@eigenpal/docx-js-editor/styles.css';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { DocxEditor, type DocxEditorRef } from '@eigenpal/docx-js-editor/react';
import { createEmptyDocument } from '@eigenpal/docx-js-editor/core';
import { toast } from 'sonner';
import { filterTransactionGuard } from '../../../editor-adapters/filter-transaction-guard';
import { type TemplateSchemas, type VersionDTO, submitForReview } from './api/templatesV2';
import { PlaceholderChip, usePlaceholderDrop } from '../placeholder-chip';
import { PlaceholderInspector } from '../placeholder-inspector';
import { CompositionConfigPanel } from '../composition-config-panel';
import type { Placeholder, SubBlockDef } from '../placeholder-types';
import { slugifyLabel } from '../placeholder-types';
import { useTemplateDraft } from './hooks/useTemplateDraft';
import { useTemplateAutosave } from './hooks/useTemplateAutosave';
import { useTemplateSchemas } from './hooks/useTemplateSchemas';
import { VersionActionPanel } from './VersionActionPanel';
import styles from './TemplateAuthorPage.module.css';

export type TemplateAuthorPageProps = {
  templateId: string;
  versionNum: number;
  onNavigateToVersion?: (templateId: string, versionNum: number) => void;
  onBack?: () => void;
};

type RailItem = {
  key: string;
  tip: string;
  kbd?: string;
  icon: JSX.Element;
};

const DEFAULT_RESOLVERS = [
  { key: 'DocCode', version: 1 },
  { key: 'RevisionNumber', version: 1 },
  { key: 'EffectiveDate', version: 1 },
  { key: 'Author', version: 1 },
  { key: 'ApprovalDate', version: 1 },
];

const SUB_BLOCK_CATALOGUE: SubBlockDef[] = [
  { key: 'doc-header', label: 'Document Header', params: [] },
  { key: 'approval-footer', label: 'Approval Footer', params: [] },
  { key: 'revision-history', label: 'Revision History', params: [] },
];

const EMPTY_COMPOSITION = { headerSubBlocks: [], footerSubBlocks: [], subBlockParams: {} };

export function TemplateAuthorPage({ templateId, versionNum, onNavigateToVersion: _nav, onBack }: TemplateAuthorPageProps) {
  const draft = useTemplateDraft(templateId, versionNum);
  const autosave = useTemplateAutosave(templateId, versionNum);
  const schemaState = useTemplateSchemas(templateId, versionNum);
  const editorRef = useRef<DocxEditorRef>(null);
  const schemaSnapshotRef = useRef<string | null>(null);
  const blankDoc = useMemo(() => createEmptyDocument(), []);
  const editorPlugins = useMemo(() => [filterTransactionGuard()], []);
  const [submitting, setSubmitting] = useState(false);
  const [submitErr, setSubmitErr] = useState<string | null>(null);
  const [liveVersion, setLiveVersion] = useState<VersionDTO | null>(null);
  const [leftActive, setLeftActive] = useState<string>('variables');
  const [rightActive, setRightActive] = useState<string>('inspector');
  const [localSchemas, setLocalSchemas] = useState<TemplateSchemas | null>(null);
  const [selectedType, setSelectedType] = useState<'placeholder' | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const queueDocx = autosave.queueDocx;
  const handleEditorChange = useCallback(() => {
    editorRef.current?.save().then((buffer) => {
      if (buffer) {
        queueDocx(buffer);
      }
    }).catch(() => {
      // ignore autosave buffer serialization errors
    });
  }, [queueDocx]);

  useEffect(() => {
    setLiveVersion(draft.version ?? null);
  }, [draft.version]);

  useEffect(() => {
    if (!schemaState.schemas) return;
    setLocalSchemas(schemaState.schemas);
    schemaSnapshotRef.current = JSON.stringify(schemaState.schemas);
  }, [schemaState.schemas]);

  const currentVersion = liveVersion ?? draft.version ?? null;
  const isDraft = currentVersion?.status === 'draft';

  useEffect(() => {
    if (!localSchemas || !isDraft) return;
    const nextSnapshot = JSON.stringify(localSchemas);
    if (schemaSnapshotRef.current === nextSnapshot) return;
    const timer = window.setTimeout(() => {
      void schemaState.save(localSchemas).then(() => {
        schemaSnapshotRef.current = nextSnapshot;
      }).catch(() => {
        // ignore schema save errors here; hook exposes error state
      });
    }, 400);
    return () => window.clearTimeout(timer);
  }, [isDraft, localSchemas, schemaState]);

  const insertTokenAtCursor = useCallback((token: string) => {
    const pagedRef = editorRef.current?.getEditorRef();
    if (!pagedRef) { toast.error('Editor not ready'); return; }
    const view = pagedRef.getView();
    if (!view) { toast.error('Editor not ready'); return; }
    const { state } = view;
    const tr = state.tr.insertText(token, state.selection.from, state.selection.to);
    view.dispatch(tr);
    view.focus();
  }, []);

  const insertPlaceholder = useCallback((id: string, name: string) => {
    insertTokenAtCursor(name ? `{${name}}` : `{{${id}}}`);
  }, [insertTokenAtCursor]);

  const placeholderDrop = usePlaceholderDrop(insertPlaceholder);

  const handleCanvasDrop = useCallback((e: React.DragEvent) => {
    placeholderDrop.onDrop(e);
  }, [placeholderDrop]);

  const updatePlaceholder = useCallback((updated: Placeholder) => {
    if (!isDraft) return;
    setLocalSchemas((prev) => {
      if (!prev) return prev;
      return {
        ...prev,
        placeholders: prev.placeholders.map((p) => (p.id === updated.id ? updated : p)),
      };
    });
  }, [isDraft]);

  const updateComposition = useCallback((updated: TemplateSchemas['composition']) => {
    if (!isDraft) return;
    setLocalSchemas((prev) => {
      if (!prev) return prev;
      return { ...prev, composition: updated };
    });
  }, [isDraft]);

  function uniqueName(base: string, existing: Placeholder[]): string {
    const taken = new Set(existing.map((p) => p.name).filter(Boolean) as string[]);
    if (!taken.has(base)) return base;
    let i = 2;
    while (taken.has(`${base}_${i}`)) i++;
    return `${base}_${i}`;
  }

  const addPlaceholder = useCallback(() => {
    if (!isDraft) return;
    const existing = localSchemas?.placeholders ?? [];
    const next: Placeholder = {
      id: crypto.randomUUID(),
      name: uniqueName(slugifyLabel('New placeholder'), existing),
      label: 'New placeholder',
      type: 'text',
    };
    setLocalSchemas((prev) => {
      if (!prev) return prev;
      return { ...prev, placeholders: [...prev.placeholders, next] };
    });
    setSelectedType('placeholder');
    setSelectedId(next.id);
    setRightActive('inspector');
  }, [isDraft]);

  const selectedPlaceholder = selectedType === 'placeholder'
    ? localSchemas?.placeholders.find((p) => p.id === selectedId) ?? null
    : null;

  // Eigenpal closes its popovers on any capture-phase scroll event. Scrolling
  // inside its own listbox (e.g. to reach font size 48) triggers the close.
  // Swallow scroll events that originate inside an eigenpal dropdown before
  // their capture listener sees them — this effect mounts before the editor's,
  // so our listener runs first in the capture order.
  useEffect(() => {
    const guard = (e: Event) => {
      const t = e.target as Element | null;
      if (!t || typeof t.closest !== 'function') return;
      if (t.closest('[role="listbox"]') || t.closest('[data-testid$="-dropdown"]')) {
        e.stopImmediatePropagation();
      }
    };
    window.addEventListener('scroll', guard, true);
    return () => window.removeEventListener('scroll', guard, true);
  }, []);

  async function handleSubmitForReview() {
    setSubmitErr(null);
    setSubmitting(true);
    try {
      if (autosave.hasPending()) await autosave.flush();
      const updated = await submitForReview(templateId, versionNum);
      setLiveVersion(updated);
      setSubmitErr('Submitted for review.');
    } catch (e) {
      setSubmitErr(e instanceof Error ? e.message : String(e));
    } finally {
      setSubmitting(false);
    }
  }

  if (draft.loading || (schemaState.loading && !localSchemas)) return <div className={styles.loading}>Loading template...</div>;
  if (draft.error) return <div role="alert" className={styles.error}>{draft.error}</div>;
  if (schemaState.error && !localSchemas) return <div role="alert" className={styles.error}>{schemaState.error}</div>;

  const statusPillClass =
    currentVersion?.status === 'draft' ? styles.draft :
    currentVersion?.status === 'in_review' ? styles.inReview :
    currentVersion?.status === 'approved' ? styles.approved :
    currentVersion?.status === 'published' ? styles.published :
    '';

  const leftRailItems: (RailItem | { divider: true })[] = [
    { key: 'variables', tip: 'Variables',                icon: IconBraces },
    { key: 'layout',  tip: 'Layout (soon)',              icon: IconLayout },
    { key: 'media',   tip: 'Media (soon)',               icon: IconImage },
    { divider: true },
    { key: 'outline', tip: 'Outline',                    icon: IconOutline },
    { key: 'search',  tip: 'Find',           kbd: '⌘F', icon: IconSearch },
  ];
  const rightRailItems: (RailItem | { divider: true })[] = [
    { key: 'inspector', tip: 'Inspector',        icon: IconInspector },
    { key: 'composition', tip: 'Composition',    icon: IconBlocks },
    { key: 'variables', tip: 'Variables',        icon: IconBraces },
    { key: 'comments',  tip: 'Comments',         icon: IconComment },
    { divider: true },
    { key: 'versions',  tip: 'Versions',         icon: IconGitBranch },
  ];

  const autosaveNode = (() => {
    if (autosave.status === 'saving') {
      return <span className={styles.autosaveStatus}><span className={styles.autosaveDot} aria-hidden="true" /> Saving…</span>;
    }
    if (autosave.status === 'error') {
      return <span className={styles.autosaveStatus} style={{ color: '#dc2626' }}>Save failed</span>;
    }
    if (autosave.status === 'saved') {
      return <span className={styles.autosaveStatus}><IconCheck className={styles.autosaveCheck} /> Saved</span>;
    }
    return <span className={styles.autosaveStatus} />;
  })();

  return (
    <div className={styles.page}>
      <div className={styles.body}>
        <aside className={`${styles.rail} ${styles.railLeft}`}>
          {onBack && (
            <>
              <button className={styles.railBackBtn} onClick={onBack} aria-label="Voltar para templates">
                {IconChevronLeft}
                <span className={styles.railTip}>Templates</span>
              </button>
              <div className={styles.railDivider} />
            </>
          )}
          {leftRailItems.map((it, i) =>
            'divider' in it ? (
              <div key={`d${i}`} className={styles.railDivider} />
            ) : (
              <button
                key={it.key}
                type="button"
                aria-label={it.tip}
                className={`${styles.railBtn} ${leftActive === it.key ? styles.isActive : ''}`}
                onClick={() => setLeftActive(it.key)}
              >
                {it.icon}
                <span className={styles.railTip}>{it.tip}{it.kbd ? `  ${it.kbd}` : ''}</span>
              </button>
            )
          )}
        </aside>

        {leftActive === 'variables' && (
          <aside className={styles.sidePanel}>
            <section className={styles.panelSection}>
              <div className={styles.panelHeader}>Placeholders</div>
              <div className={styles.chipList}>
                {(localSchemas?.placeholders ?? []).map((placeholder) => (
                  <PlaceholderChip
                    key={placeholder.id}
                    placeholder={placeholder}
                    onInsert={(p) => {
                      setSelectedType('placeholder');
                      setSelectedId(p.id);
                      setRightActive('inspector');
                    }}
                  />
                ))}
              </div>
              <button type="button" className={styles.addBtn} onClick={addPlaceholder} disabled={!isDraft}>
                + Add placeholder
              </button>
            </section>
          </aside>
        )}

        <main
          className={styles.canvas}
          onDragOver={(e) => {
            placeholderDrop.onDragOver(e);
          }}
          onDrop={handleCanvasDrop}
        >
          <div className={styles.editorWrapper}>
            <div className={styles.overlayTitle}>
              <span className={styles.docTitle}>{draft.template?.name ?? 'Untitled template'}</span>
              <span className={styles.docSep}>·</span>
              <span className={styles.docMeta}>Template</span>
              <span className={styles.versionBadge}>REV{String(versionNum).padStart(2, '0')}</span>
              {currentVersion?.status && (
                <span className={`${styles.statusPill} ${statusPillClass}`}>{currentVersion.status.replace('_', ' ')}</span>
              )}
            </div>
            <div className={styles.overlayRight}>
              {autosaveNode}
              {isDraft && (
                <button
                  className={styles.editorSubmitBtn}
                  onClick={() => void handleSubmitForReview()}
                  disabled={submitting}
                >
                  {IconSend} {submitting ? 'Enviando…' : 'Solicitar Revisão'}
                </button>
              )}
            </div>
            {submitErr && (
              <div
                role="alert"
                className={styles.overlayAlert}
                style={{ color: submitErr === 'Submitted for review.' ? '#065f46' : '#dc2626' }}
              >
                {submitErr}
              </div>
            )}
            <DocxEditor
              ref={editorRef}
              documentBuffer={draft.docxBytes ?? undefined}
              document={draft.docxBytes ? undefined : blankDoc}
              readOnly={!isDraft}
              onChange={handleEditorChange}
              externalPlugins={editorPlugins}
            />
          </div>
        </main>

        <aside className={styles.rightPanel}>
          {rightActive === 'inspector' && (
            <>
              {selectedPlaceholder ? (
                <fieldset className={styles.inspectorFieldset} disabled={!isDraft}>
                  <PlaceholderInspector
                    value={selectedPlaceholder}
                    resolvers={DEFAULT_RESOLVERS}
                    onChange={(updated) => {
                      if (!isDraft) return;
                      updatePlaceholder(updated);
                    }}
                  />
                </fieldset>
              ) : (
                <div className={styles.panelHeader}>Select a chip to inspect</div>
              )}
            </>
          )}
          {rightActive === 'composition' && (
            <fieldset className={styles.inspectorFieldset} disabled={!isDraft}>
              <CompositionConfigPanel
                value={localSchemas?.composition ?? EMPTY_COMPOSITION}
                subBlockCatalogue={SUB_BLOCK_CATALOGUE}
                onChange={(updated) => {
                  if (!isDraft) return;
                  updateComposition(updated);
                }}
              />
            </fieldset>
          )}
          {rightActive !== 'inspector' && rightActive !== 'composition' && (
            <div className={styles.panelHeader} />
          )}
        </aside>

        <aside className={`${styles.rail} ${styles.railRight}`}>
          {rightRailItems.map((it, i) =>
            'divider' in it ? (
              <div key={`d${i}`} className={styles.railDivider} />
            ) : (
              <button
                key={it.key}
                type="button"
                aria-label={it.tip}
                className={`${styles.railBtn} ${rightActive === it.key ? styles.isActive : ''}`}
                onClick={() => setRightActive(it.key)}
              >
                {it.icon}
                <span className={styles.railTip}>{it.tip}</span>
              </button>
            )
          )}
        </aside>
      </div>

      {currentVersion && ['in_review', 'approved', 'published'].includes(currentVersion.status) && (
        <VersionActionPanel
          version={currentVersion}
          onVersionUpdate={(v) => setLiveVersion(v)}
        />
      )}
    </div>
  );
}

/* Inline SVG icons, Lucide-style */

const svgBase = {
  width: 18,
  height: 18,
  viewBox: '0 0 24 24',
  fill: 'none',
  stroke: 'currentColor',
  strokeWidth: 1.75,
  strokeLinecap: 'round' as const,
  strokeLinejoin: 'round' as const,
};

function IconCheck({ className }: { className?: string }) {
  return (
    <svg {...svgBase} width={14} height={14} className={className}><path d="M20 6 9 17l-5-5" /></svg>
  );
}
const IconSend = (
  <svg {...svgBase} width={14} height={14}><path d="m22 2-7 20-4-9-9-4Z" /><path d="M22 2 11 13" /></svg>
);
const IconBlocks = (
  <svg {...svgBase}><rect x="3" y="3" width="7" height="7" rx="1" /><rect x="14" y="3" width="7" height="7" rx="1" /><rect x="3" y="14" width="7" height="7" rx="1" /><rect x="14" y="14" width="7" height="7" rx="1" /></svg>
);
const IconLayout = (
  <svg {...svgBase}><rect x="3" y="3" width="18" height="18" rx="2" /><path d="M3 9h18" /><path d="M9 21V9" /></svg>
);
const IconImage = (
  <svg {...svgBase}><rect x="3" y="3" width="18" height="18" rx="2" /><circle cx="9" cy="9" r="2" /><path d="m21 15-5-5L5 21" /></svg>
);
const IconOutline = (
  <svg {...svgBase}><path d="M21 12h-8" /><path d="M21 6h-8" /><path d="M21 18h-8" /><path d="M3 6h.01" /><path d="M3 12h.01" /><path d="M3 18h.01" /></svg>
);
const IconSearch = (
  <svg {...svgBase}><circle cx="11" cy="11" r="7" /><path d="m21 21-4.3-4.3" /></svg>
);
const IconInspector = (
  <svg {...svgBase}><rect x="3" y="3" width="18" height="18" rx="2" /><path d="M15 3v18" /></svg>
);
const IconBraces = (
  <svg {...svgBase}><path d="M8 3H7a2 2 0 0 0-2 2v5a2 2 0 0 1-2 2 2 2 0 0 1 2 2v5a2 2 0 0 0 2 2h1" /><path d="M16 21h1a2 2 0 0 0 2-2v-5a2 2 0 0 1 2-2 2 2 0 0 1-2-2V5a2 2 0 0 0-2-2h-1" /></svg>
);
const IconComment = (
  <svg {...svgBase}><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" /></svg>
);
const IconGitBranch = (
  <svg {...svgBase}><line x1="6" y1="3" x2="6" y2="15" /><circle cx="18" cy="6" r="3" /><circle cx="6" cy="18" r="3" /><path d="M18 9a9 9 0 0 1-9 9" /></svg>
);
const IconChevronLeft = (
  <svg {...svgBase} width={14} height={14}><path d="M15 18l-6-6 6-6" /></svg>
);
const IconFileDoc = (
  <svg width={12} height={12} viewBox="0 0 15 15" fill="none" stroke="rgba(255,255,255,0.9)" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round">
    <path d="M3 2h6.5L12 4.5V13H3V2z" />
    <path d="M9.5 2v2.5H12" />
    <path d="M5 7h5M5 9.5h5M5 12h3" />
  </svg>
);
const IconMenu = (
  <svg {...svgBase} width={16} height={16}><line x1="3" y1="6" x2="21" y2="6" /><line x1="3" y1="12" x2="21" y2="12" /></svg>
);
const IconHistory = (
  <svg {...svgBase} width={16} height={16}><circle cx="12" cy="12" r="9" /><polyline points="12 7 12 12 15 15" /></svg>
);
const IconShare = (
  <svg {...svgBase} width={16} height={16}><circle cx="18" cy="5" r="3" /><circle cx="6" cy="12" r="3" /><circle cx="18" cy="19" r="3" /><line x1="8.59" y1="13.51" x2="15.42" y2="17.49" /><line x1="15.41" y1="6.51" x2="8.59" y2="10.49" /></svg>
);
