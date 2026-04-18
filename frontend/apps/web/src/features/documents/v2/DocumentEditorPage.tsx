import { useCallback, useEffect, useMemo, useState } from 'react';
import { useDocumentSession } from './hooks/useDocumentSession';
import { useDocumentAutosave } from './hooks/useDocumentAutosave';
import { getDocument, finalizeDocument, signedRevisionURL } from './api/documentsV2';
import { CheckpointsPanel } from './CheckpointsPanel';
import { ExportMenu } from './ExportMenu';
import styles from './styles/DocumentEditorPage.module.css';

export type DocumentEditorPageProps = {
  documentID: string;
  onDone: () => void;
};

export function DocumentEditorPage({ documentID, onDone }: DocumentEditorPageProps): React.ReactElement {
  const session = useDocumentSession(documentID);
  const [documentName, setDocumentName] = useState('');
  const [currentRevisionID, setCurrentRevisionID] = useState('');
  const [buffer, setBuffer] = useState<ArrayBuffer | null>(null);
  const [error, setError] = useState('');

  const loadDocument = useCallback(async () => {
    const doc = await getDocument(documentID);
    setDocumentName(doc.Name ?? doc.name ?? 'Document');
    setCurrentRevisionID(doc.CurrentRevisionID ?? doc.current_revision_id ?? '');
  }, [documentID]);

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
        setError('');
        const doc = await getDocument(documentID);
        const name = doc.Name ?? doc.name ?? 'Document';
        const revisionID = doc.CurrentRevisionID ?? doc.current_revision_id ?? '';
        setDocumentName(name);
        setCurrentRevisionID(revisionID);
        await fetchRevisionBuffer(revisionID);
      } catch {
        setError('Failed to load document.');
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
          setCurrentRevisionID(newRevisionID);
        },
        onSessionLost: () => {
          setError('Writer session lost.');
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

  async function handleFinalize() {
    if (session.state.phase !== 'writer') return;
    try {
      setError('');
      await autosave.flush();
      await finalizeDocument(documentID);
      await session.release();
      onDone();
    } catch {
      setError('Failed to finalize document.');
    }
  }

  async function handleRestored(newRevisionID: string) {
    try {
      setError('');
      await fetchRevisionBuffer(newRevisionID);
      session.setLastAck(newRevisionID);
      setCurrentRevisionID(newRevisionID);
    } catch {
      setError('Failed to refresh document after restore.');
    }
  }

  return (
    <div className={styles.page} data-editor-root>
      <header className={styles.header}>
        <strong>{documentName || 'Document'}</strong>
        <span className={styles.status} data-status={autosave.status}>{autosave.status}</span>
        <button type="button" onClick={() => void handleFinalize()} disabled={session.state.phase !== 'writer'}>
          Finalize
        </button>
      </header>
      {session.state.phase === 'readonly' && (
        <div className={styles.banner}>Readonly session. Another user is editing this document.</div>
      )}
      {session.state.phase === 'lost' && (
        <div className={styles.banner}>Session lost. Reload to reacquire writer access.</div>
      )}
      {error && <div className={styles.banner}>{error}</div>}
      <div className={styles.split}>
        <div className={styles.editor} data-editor-placeholder>
          Document editor surface (W4)
          {buffer ? <div>Revision loaded: {currentRevisionID}</div> : null}
        </div>
        <CheckpointsPanel
          documentID={documentID}
          disabled={session.state.phase !== 'writer'}
          onRestored={(newRevisionID) => {
            void handleRestored(newRevisionID);
          }}
        />
      </div>
      <ExportMenu
        documentID={documentID}
        canExport={session.state.phase === 'writer' || session.state.phase === 'readonly'}
      />
    </div>
  );
}
