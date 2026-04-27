import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { MetalDocsEditor, type MetalDocsEditorRef } from '@metaldocs/editor-ui';
import type { Comment } from '@metaldocs/editor-ui';
import { toast } from 'sonner';
import { useDocumentSession } from './hooks/useDocumentSession';
import { useDocumentAutosave } from './hooks/useDocumentAutosave';
import { useDocumentComments } from './hooks/useDocumentComments';
import { getDocument, finalizeDocument, renameDocument, signedRevisionURL } from './api/documentsV2';
import type { DocumentResponse } from './api/documentsV2';
import { CheckpointsDialog } from './CheckpointsDialog';
import { ExportMenuButton } from './ExportMenuButton';
import { StateBadge } from '../../approval/components/StateBadge';
import type { ApprovalState } from '../../approval/api/approvalTypes';
import styles from './styles/DocumentEditorPage.module.css';

export type DocumentEditorPageProps = {
  documentID: string;
  onDone: () => void;
};

export function DocumentEditorPage({ documentID, onDone }: DocumentEditorPageProps): React.ReactElement {
  const session = useDocumentSession(documentID);
  const [doc, setDoc] = useState<DocumentResponse | null>(null);
  const [documentName, setDocumentName] = useState('');
  const [buffer, setBuffer] = useState<ArrayBuffer | null | undefined>(undefined);
  const [checkpointsOpen, setCheckpointsOpen] = useState(false);
  const editorRef = useRef<MetalDocsEditorRef>(null);

  const fetchRevisionBuffer = useCallback(async (revisionID: string) => {
    if (!revisionID) {
      setBuffer(null);
      return;
    }
    const signedRes = await fetch(signedRevisionURL(documentID, revisionID));
    if (!signedRes.ok) throw Object.assign(new Error(`http_${signedRes.status}`), { status: signedRes.status });
    const signedPayload = await signedRes.json() as { url?: string };
    if (!signedPayload.url) {
      throw new Error('missing_signed_url');
    }
    const fileRes = await fetch(signedPayload.url);
    if (!fileRes.ok) throw Object.assign(new Error(`http_${fileRes.status}`), { status: fileRes.status });
    setBuffer(await fileRes.arrayBuffer());
  }, [documentID]);

  useEffect(() => {
    void (async () => {
      try {
        setBuffer(undefined);
        const loadedDoc = await getDocument(documentID);
        const name = loadedDoc.Name ?? loadedDoc.name ?? 'Document';
        const revisionID = loadedDoc.CurrentRevisionID ?? loadedDoc.current_revision_id ?? '';
        setDoc(loadedDoc);
        setDocumentName(name);
        await fetchRevisionBuffer(revisionID);
      } catch {
        toast.error('Failed to load document.');
      }
    })();
  }, [documentID, fetchRevisionBuffer]);

  const sessionPhase = session.state.phase;
  const sessionID = sessionPhase === 'writer' ? session.state.sessionID : '';
  const lastAckRevisionID = sessionPhase === 'writer' ? session.state.lastAckRevisionID : '';
  const { setLastAck } = session;

  const autosaveArgs = useMemo(() => {
    if (sessionPhase === 'writer') {
      return {
        documentID,
        sessionID,
        baseRevisionID: lastAckRevisionID,
        onAdvanceBase: (newRevisionID: string) => {
          setLastAck(newRevisionID);
        },
        onSessionLost: () => {
          toast.error('Writer session lost.');
        },
      };
    }
    return {
      documentID,
      sessionID: '',
      baseRevisionID: '',
      onAdvanceBase: () => {},
      onSessionLost: () => {},
    };
  }, [documentID, sessionPhase, sessionID, lastAckRevisionID, setLastAck]);

  const autosave = useDocumentAutosave(autosaveArgs);

  const prevAutosaveStatus = useRef(autosave.status);
  useEffect(() => {
    if (autosave.status === prevAutosaveStatus.current) {
      return;
    }
    prevAutosaveStatus.current = autosave.status;
    if (autosave.status === 'error' || autosave.status === 'session_lost' || autosave.status === 'stale') {
      toast.error(`Autosave ${autosave.status.replace('_', ' ')}.`);
    }
  }, [autosave.status]);

  const prevSessionPhase = useRef(sessionPhase);
  useEffect(() => {
    if (sessionPhase !== prevSessionPhase.current && (sessionPhase === 'readonly' || sessionPhase === 'lost')) {
      toast.warning(
        sessionPhase === 'readonly'
          ? 'Readonly session. Another user is editing this document.'
          : 'Session lost. Reload to reacquire writer access.',
      );
    }
    prevSessionPhase.current = sessionPhase;
  }, [sessionPhase]);

  const handleRename = useCallback((name: string) => {
    setDocumentName(name);
    void renameDocument(documentID, name).catch(() => {
      toast.error('Failed to rename document.');
    });
  }, [documentID]);

  async function handleSave() {
    if (!editorRef.current) return;
    if (!doc) return;
    const buf = await editorRef.current.getDocumentBuffer();
    if (!buf) return;
    await autosave.queue(buf, doc.FormDataJSON ?? doc.form_data ?? null);
  }

  async function handleFinalize() {
    if (session.state.phase !== 'writer' || !doc) return;
    try {
      await autosave.flush();
      await finalizeDocument(documentID);
      await session.release();
      onDone();
    } catch {
      toast.error('Failed to finalize document.');
    }
  }

  async function handleRestored(newRevisionID: string) {
    try {
      await fetchRevisionBuffer(newRevisionID);
      const refreshedDoc = await getDocument(documentID);
      setDoc(refreshedDoc);
      setDocumentName(refreshedDoc.Name ?? refreshedDoc.name ?? 'Document');
      session.setLastAck(newRevisionID);
    } catch {
      toast.error('Failed to refresh document after restore.');
    }
  }

  const docStatus = doc?.Status ?? doc?.status ?? '';
  const docCode = doc?.Code ?? doc?.code ?? '';
  const VALID_STATES: readonly ApprovalState[] = ['draft', 'under_review', 'approved', 'scheduled', 'published', 'superseded', 'rejected', 'obsolete', 'cancelled'];
  const badgeState: ApprovalState | null = (VALID_STATES as readonly string[]).includes(docStatus) ? (docStatus as ApprovalState) : null;
  const userID = doc?.CreatedBy ?? doc?.created_by ?? '';
  const authorDisplay = String(userID);
  const commentsHook = useDocumentComments(documentID, authorDisplay);
  const canMountEditor = !!doc
    && session.state.phase !== 'idle'
    && session.state.phase !== 'acquiring'
    && buffer !== undefined;

  return (
    <div className={styles.page} data-editor-root>
      <div className={styles.body}>

        <aside className={`${styles.rail} ${styles.railLeft}`}>
          <button
            className={styles.railBackBtn}
            onClick={onDone}
            aria-label="Voltar"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="15 18 9 12 15 6" />
            </svg>
            <span className={styles.railTip}>Voltar</span>
          </button>
        </aside>

        <main className={styles.canvas}>
          <div className={styles.editorWrapper}>

            <div className={styles.overlayTitle}>
              <span className={styles.docTitle}>{documentName || 'Document'}</span>
              {docCode && (
                <>
                  <span className={styles.docSep}>·</span>
                  <span className={styles.docMeta} style={{ fontFamily: 'monospace' }}>{docCode}</span>
                </>
              )}
              {badgeState && <StateBadge state={badgeState} size="sm" />}
            </div>

            <div className={styles.overlayRight}>
              {autosave.status === 'saving' && (
                <span className={styles.autosaveStatus}>
                  <span className={styles.autosaveDot} aria-hidden="true" />
                  Saving…
                </span>
              )}
              {autosave.status === 'error' && (
                <span className={styles.autosaveStatus} style={{ color: '#dc2626' }}>Save failed</span>
              )}
              {autosave.status === 'saved' && (
                <span className={styles.autosaveStatus}>✓ Saved</span>
              )}
              <button
                type="button"
                className={styles.editorSubmitBtn}
                onClick={() => setCheckpointsOpen(true)}
              >
                Checkpoints
              </button>
              <ExportMenuButton
                documentID={documentID}
                canExport={sessionPhase === 'writer' || sessionPhase === 'readonly'}
              />
              <button
                type="button"
                className={styles.editorSubmitBtn}
                onClick={() => void handleFinalize()}
                disabled={session.state.phase !== 'writer' || docStatus !== 'draft'}
              >
                Finalizar
              </button>
            </div>

            {canMountEditor ? (
              <MetalDocsEditor
                ref={editorRef}
                mode={session.state.phase === 'writer' ? 'document-edit' : 'readonly'}
                documentBuffer={buffer ?? undefined}
                author={authorDisplay}
                comments={commentsHook.comments}
                onCommentsChange={commentsHook.setComments}
                onCommentAdd={(c: Comment) => void commentsHook.add(c)}
                onCommentResolve={(c: Comment) => void (c.done ? commentsHook.resolve(c) : commentsHook.reopen(c))}
                onCommentDelete={(c: Comment) => void commentsHook.remove(c)}
                onCommentReply={(reply: Comment, parent: Comment) => void commentsHook.reply(reply, parent)}
                documentName={documentName}
                documentNameEditable={session.state.phase === 'writer'}
                onDocumentNameChange={handleRename}
                onAutoSave={handleSave}
              />
            ) : null}

          </div>
        </main>

      </div>

      <CheckpointsDialog
        open={checkpointsOpen}
        onClose={() => setCheckpointsOpen(false)}
        documentID={documentID}
        disabled={session.state.phase !== 'writer'}
        onRestored={(rev) => {
          setCheckpointsOpen(false);
          void handleRestored(rev);
        }}
      />
    </div>
  );
}
