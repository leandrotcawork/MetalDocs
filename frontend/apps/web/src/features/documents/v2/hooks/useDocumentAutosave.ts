import { useCallback, useEffect, useRef, useState } from 'react';
import { presignAutosave, commitAutosave } from '../api/documentsV2';
import { deletePending, putPending, getAllPending } from './useIndexedDBRestore';

export type AutosaveStatus = 'idle' | 'dirty' | 'saving' | 'saved' | 'stale' | 'session_lost' | 'error';

export interface AutosaveArgs {
  documentID: string;
  sessionID: string;
  baseRevisionID: string;
  onAdvanceBase: (newRevisionID: string) => void;
  onSessionLost: (reason: 'stale_base' | 'session_inactive' | 'force_released') => void;
}

const SYNC_DEBOUNCE_MS = 15_000;

async function sha256Hex(buf: ArrayBuffer): Promise<string> {
  const digest = await crypto.subtle.digest('SHA-256', buf);
  return Array.from(new Uint8Array(digest)).map((b) => b.toString(16).padStart(2, '0')).join('');
}

export function useDocumentAutosave(args: AutosaveArgs) {
  const pending = useRef<ArrayBuffer | null>(null);
  const pendingHash = useRef<string>('');
  const formSnapshot = useRef<unknown>(null);
  const timer = useRef<number | null>(null);
  const [status, setStatus] = useState<AutosaveStatus>('idle');

  const flush = useCallback(async () => {
    if (!pending.current) return;
    setStatus('saving');
    const buf = pending.current;
    const hash = pendingHash.current;
    try {
      // Persist to IndexedDB BEFORE hitting network -- crash recovery.
      await putPending({
        document_id: args.documentID,
        session_id: args.sessionID,
        base_revision_id: args.baseRevisionID,
        content_hash: hash,
        buffer: buf,
        created_at: Date.now(),
      });
      const presigned = await presignAutosave(args.documentID, {
        session_id: args.sessionID,
        base_revision_id: args.baseRevisionID,
        content_hash: hash,
      });
      await fetch(presigned.UploadURL, {
        method: 'PUT',
        headers: { 'content-type': 'application/vnd.openxmlformats-officedocument.wordprocessingml.document' },
        body: buf,
      });
      // Server re-computes content_hash from S3; client does NOT send a hash.
      const commit = await commitAutosave(args.documentID, {
        session_id: args.sessionID,
        pending_upload_id: presigned.PendingUploadID,
        form_data_snapshot: formSnapshot.current,
      });
      await deletePending(args.documentID, hash);
      pending.current = null; pendingHash.current = '';
      args.onAdvanceBase(commit.revision_id);
      setStatus('saved');
    } catch (e: any) {
      if (e?.status === 409) {
        const body = e?.body ? (() => { try { return JSON.parse(e.body); } catch { return {}; } })() : {};
        if (body?.error === 'stale_base') { args.onSessionLost('stale_base'); setStatus('stale'); return; }
        if (body?.error === 'session_inactive' || body?.error === 'session_not_holder') {
          args.onSessionLost('session_inactive'); setStatus('session_lost'); return;
        }
      }
      if (e?.status === 410) {
        // upload_missing or expired_upload: the S3 object is gone.
        try { await deletePending(args.documentID, hash); } catch { /* ignore */ }
        pending.current = null; pendingHash.current = '';
        setStatus('error'); return;
      }
      if (e?.status === 422) {
        // content_hash_mismatch: discard local pending.
        try { await deletePending(args.documentID, hash); } catch { /* ignore */ }
        pending.current = null; pendingHash.current = '';
        setStatus('error'); return;
      }
      setStatus('error');
    }
  }, [args]);

  const schedule = useCallback(() => {
    if (timer.current) window.clearTimeout(timer.current);
    timer.current = window.setTimeout(flush, SYNC_DEBOUNCE_MS);
  }, [flush]);

  const queue = useCallback(async (buf: ArrayBuffer, snapshot: unknown) => {
    pending.current = buf;
    formSnapshot.current = snapshot;
    pendingHash.current = await sha256Hex(buf);
    setStatus('dirty');
    schedule();
  }, [schedule]);

  // Recovery on mount: if IndexedDB has a pending blob not yet committed,
  // replay it (if session matches base we still hold).
  useEffect(() => {
    (async () => {
      const leftovers = await getAllPending(args.documentID);
      for (const p of leftovers) {
        if (p.session_id !== args.sessionID) continue;
        pending.current = p.buffer;
        pendingHash.current = p.content_hash;
        await flush();
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [args.documentID, args.sessionID]);

  useEffect(() => () => { if (timer.current) window.clearTimeout(timer.current); }, []);

  return { status, queue, flush };
}
